#include "task.h"
#include "executor.h"

#include <algorithm>
#include <cassert>
#include <stdexcept>
#include <utility>

void Task::AddDependency(TaskPtr dep) {
    if (dep.get() == this) {
        throw std::runtime_error("self dependency");
    }

    auto self = shared_from_this();
    std::scoped_lock lock(mutex_, dep->mutex_);

    if (state_ != State::Created) {
        throw std::runtime_error("cannot add dependency after activation");
    }

    ++dependency_count_;
    ++unfinished_dependencies_;
    dep->dependents_.push_back(self);
    dependency_sources_.push_back(dep);

    if (dep->state_ == State::Completed) {
        --unfinished_dependencies_;
    } else if (dep->state_ == State::Failed || dep->state_ == State::Cancelled) {
        --unfinished_dependencies_;
        failed_dependency_ = true;
    }
}

void Task::Cancel() {
    std::vector<std::shared_ptr<Task>> dependents;
    std::vector<std::shared_ptr<Task>> trigger_targets;

    {
        std::lock_guard lock(mutex_);
        if (state_ != State::Created && state_ != State::Queued) {
            return;
        }
        state_ = State::Cancelled;
        Discard();
        cv_.notify_all();
        self_hold_.reset();
        dependents = CollectDependentsLocked();
        trigger_targets = CollectTriggerTargetsLocked();
    }

    if (NeedsDetachFromSources()) {
        DetachFromSources();
    }

    for (auto& dep : dependents) {
        dep->OnDependencyResolved(false);
    }
    for (auto& target : trigger_targets) {
        target->OnTriggerResolved(false);
    }
}

std::exception_ptr Task::GetError() {
    std::lock_guard lock(mutex_);
    return error_;
}

bool Task::IsFinished() const {
    std::lock_guard lock(mutex_);
    return IsTerminalLocked();
}

bool Task::IsCancelled() const {
    std::lock_guard lock(mutex_);
    return state_ == State::Cancelled;
}

void Task::Wait() {
    std::unique_lock lock(mutex_);
    cv_.wait(lock, [this] { return IsTerminalLocked(); });
}

void Task::RequestSubmit(const std::shared_ptr<Executor>& owner) {
    std::shared_ptr<Executor> exec;
    bool enqueue = false;
    bool cancel = false;
    std::vector<std::shared_ptr<Task>> dependents;
    std::vector<std::shared_ptr<Task>> trigger_targets;

    {
        std::lock_guard lock(mutex_);
        if (IsTerminalLocked()) {
            return;
        }
        owner_ = owner;
        submit_requested_ = true;
        if (!self_hold_) {
            self_hold_ = shared_from_this();
        }

        if (ShouldCancelLocked()) {
            state_ = State::Cancelled;
            Discard();
            cv_.notify_all();
            self_hold_.reset();
            dependents = CollectDependentsLocked();
            trigger_targets = CollectTriggerTargetsLocked();
            cancel = true;
        } else if (state_ == State::Created && ShouldStartLocked()) {
            state_ = State::Queued;
            self_hold_.reset();
            exec = owner_.lock();
            enqueue = true;
        }
    }

    if (cancel) {
        if (NeedsDetachFromSources()) {
            DetachFromSources();
        }
        for (auto& dep : dependents) {
            dep->OnDependencyResolved(false);
        }
        for (auto& target : trigger_targets) {
            target->OnTriggerResolved(false);
        }
        return;
    }

    if (enqueue && exec) {
        exec->OnTaskReady(shared_from_this());
    }
}

void Task::AddTimeTrigger(const std::shared_ptr<Executor>& owner) {
    {
        std::lock_guard lock(mutex_);
        if (IsTerminalLocked()) {
            return;
        }
        owner_ = owner;
        ++active_triggers_;
        if (!self_hold_) {
            self_hold_ = shared_from_this();
        }
    }
}

void Task::AddTaskTrigger(const std::shared_ptr<Executor>& owner, const TaskPtr& trigger) {
    bool fire = false;
    bool cancelled = false;

    {
        std::scoped_lock lock(mutex_, trigger->mutex_);
        if (IsTerminalLocked()) {
            return;
        }
        owner_ = owner;
        if (trigger->state_ == State::Completed || trigger->state_ == State::Failed) {
            fire = true;
        } else if (trigger->state_ == State::Cancelled) {
            cancelled = true;
        } else {
            ++active_triggers_;
            if (!self_hold_) {
                self_hold_ = shared_from_this();
            }
            trigger->trigger_targets_.push_back(shared_from_this());
            trigger_sources_.push_back(trigger);
        }
    }

    if (fire) {
        OnTriggerResolved(true);
    } else if (cancelled) {
        OnTriggerResolved(false);
    }
}

void Task::FireTimeTrigger() {
    std::shared_ptr<Executor> exec;
    bool enqueue = false;
    bool cancel = false;
    std::vector<std::shared_ptr<Task>> dependents;
    std::vector<std::shared_ptr<Task>> trigger_targets;

    {
        std::lock_guard lock(mutex_);
        if (IsTerminalLocked()) {
            return;
        }
        if (active_triggers_ > 0) {
            --active_triggers_;
        }
        fired_trigger_ = true;

        if (ShouldCancelLocked()) {
            state_ = State::Cancelled;
            Discard();
            cv_.notify_all();
            self_hold_.reset();
            dependents = CollectDependentsLocked();
            trigger_targets = CollectTriggerTargetsLocked();
            cancel = true;
        } else if (state_ == State::Created && ShouldStartLocked()) {
            state_ = State::Queued;
            self_hold_.reset();
            exec = owner_.lock();
            enqueue = true;
        }
    }

    if (cancel) {
        if (NeedsDetachFromSources()) {
            DetachFromSources();
        }
        for (auto& dep : dependents) {
            dep->OnDependencyResolved(false);
        }
        for (auto& target : trigger_targets) {
            target->OnTriggerResolved(false);
        }
        return;
    }

    if (enqueue && exec) {
        exec->OnTaskReady(shared_from_this());
    }
}

void Task::OnTriggerResolved(bool fired) {
    std::shared_ptr<Executor> exec;
    bool enqueue = false;
    bool cancel = false;
    std::vector<std::shared_ptr<Task>> dependents;
    std::vector<std::shared_ptr<Task>> trigger_targets;

    {
        std::lock_guard lock(mutex_);
        if (IsTerminalLocked()) {
            return;
        }
        if (active_triggers_ > 0) {
            --active_triggers_;
        }
        if (fired) {
            fired_trigger_ = true;
        }

        if (ShouldCancelLocked()) {
            state_ = State::Cancelled;
            Discard();
            cv_.notify_all();
            self_hold_.reset();
            dependents = CollectDependentsLocked();
            trigger_targets = CollectTriggerTargetsLocked();
            cancel = true;
        } else if (state_ == State::Created && ShouldStartLocked()) {
            state_ = State::Queued;
            self_hold_.reset();
            exec = owner_.lock();
            enqueue = true;
        }
    }

    if (cancel) {
        if (NeedsDetachFromSources()) {
            DetachFromSources();
        }
        for (auto& dep : dependents) {
            dep->OnDependencyResolved(false);
        }
        for (auto& target : trigger_targets) {
            target->OnTriggerResolved(false);
        }
        return;
    }

    if (enqueue && exec) {
        exec->OnTaskReady(shared_from_this());
    }
}

void Task::OnDependencyResolved(bool success) {
    std::shared_ptr<Executor> exec;
    bool enqueue = false;
    bool cancel = false;
    std::vector<std::shared_ptr<Task>> dependents;
    std::vector<std::shared_ptr<Task>> trigger_targets;

    {
        std::lock_guard lock(mutex_);
        if (IsTerminalLocked()) {
            return;
        }
        if (unfinished_dependencies_ > 0) {
            --unfinished_dependencies_;
        }
        if (!success) {
            failed_dependency_ = true;
        }

        if (ShouldCancelLocked()) {
            state_ = State::Cancelled;
            Discard();
            cv_.notify_all();
            self_hold_.reset();
            dependents = CollectDependentsLocked();
            trigger_targets = CollectTriggerTargetsLocked();
            cancel = true;
        } else if (state_ == State::Created && ShouldStartLocked()) {
            state_ = State::Queued;
            self_hold_.reset();
            exec = owner_.lock();
            enqueue = true;
        }
    }

    if (cancel) {
        if (NeedsDetachFromSources()) {
            DetachFromSources();
        }
        for (auto& dep : dependents) {
            dep->OnDependencyResolved(false);
        }
        for (auto& target : trigger_targets) {
            target->OnTriggerResolved(false);
        }
        return;
    }

    if (enqueue && exec) {
        exec->OnTaskReady(shared_from_this());
    }
}

bool Task::TryRun() {
    std::lock_guard lock(mutex_);
    if (state_ != State::Queued) {
        return false;
    }
    state_ = State::Running;
    return true;
}

void Task::FinishSuccess() {
    std::vector<std::shared_ptr<Task>> dependents;
    std::vector<std::shared_ptr<Task>> trigger_targets;

    {
        std::lock_guard lock(mutex_);
        assert(state_ == State::Running);
        state_ = State::Completed;
        Discard();
        cv_.notify_all();
        self_hold_.reset();
        dependents = CollectDependentsLocked();
        trigger_targets = CollectTriggerTargetsLocked();
    }

    if (NeedsDetachFromSources()) {
        DetachFromSources();
    }

    for (auto& dep : dependents) {
        dep->OnDependencyResolved(true);
    }
    for (auto& target : trigger_targets) {
        target->OnTriggerResolved(true);
    }
}

void Task::FinishFailure(std::exception_ptr error) {
    std::vector<std::shared_ptr<Task>> dependents;
    std::vector<std::shared_ptr<Task>> trigger_targets;

    {
        std::lock_guard lock(mutex_);
        assert(state_ == State::Running);
        error_ = error;
        state_ = State::Failed;
        Discard();
        cv_.notify_all();
        self_hold_.reset();
        dependents = CollectDependentsLocked();
        trigger_targets = CollectTriggerTargetsLocked();
    }

    if (NeedsDetachFromSources()) {
        DetachFromSources();
    }

    for (auto& dep : dependents) {
        dep->OnDependencyResolved(false);
    }
    for (auto& target : trigger_targets) {
        target->OnTriggerResolved(true);
    }
}

bool Task::ShouldStartLocked() const {
    if (state_ != State::Created) {
        return false;
    }
    if (fired_trigger_) {
        return true;
    }
    if (dependency_count_ == 0) {
        return submit_requested_;
    }
    return !failed_dependency_ && unfinished_dependencies_ == 0 &&
           (submit_requested_ || active_triggers_ > 0 || fired_trigger_);
}

bool Task::ShouldCancelLocked() const {
    return state_ == State::Created && failed_dependency_ && !fired_trigger_ &&
           active_triggers_ == 0 && submit_requested_;
}

bool Task::IsTerminalLocked() const {
    return state_ == State::Completed || state_ == State::Failed || state_ == State::Cancelled;
}

std::vector<std::shared_ptr<Task>> Task::CollectDependentsLocked() {
    std::vector<std::shared_ptr<Task>> result;
    result.reserve(dependents_.size());
    for (auto& weak : dependents_) {
        if (auto p = weak.lock()) {
            result.push_back(std::move(p));
        }
    }
    dependents_.clear();
    return result;
}

std::vector<std::shared_ptr<Task>> Task::CollectTriggerTargetsLocked() {
    std::vector<std::shared_ptr<Task>> result;
    result.reserve(trigger_targets_.size());
    for (auto& weak : trigger_targets_) {
        if (auto p = weak.lock()) {
            result.push_back(std::move(p));
        }
    }
    trigger_targets_.clear();
    active_triggers_ = 0;
    return result;
}

void Task::DetachFromSources() {
    std::vector<std::shared_ptr<Task>> deps;
    std::vector<std::shared_ptr<Task>> triggers;

    {
        std::lock_guard lock(mutex_);
        for (auto& weak : dependency_sources_) {
            if (auto src = weak.lock()) {
                deps.push_back(std::move(src));
            }
        }
        for (auto& weak : trigger_sources_) {
            if (auto src = weak.lock()) {
                triggers.push_back(std::move(src));
            }
        }
        dependency_sources_.clear();
        trigger_sources_.clear();
    }

    for (auto& src : deps) {
        std::lock_guard lock(src->mutex_);
        auto& vec = src->dependents_;
        vec.erase(std::remove_if(vec.begin(), vec.end(),
                                 [this](const std::weak_ptr<Task>& p) {
                                     auto sp = p.lock();
                                     return !sp || sp.get() == this;
                                 }),
                  vec.end());
    }

    for (auto& src : triggers) {
        std::lock_guard lock(src->mutex_);
        auto& vec = src->trigger_targets_;
        vec.erase(std::remove_if(vec.begin(), vec.end(),
                                 [this](const std::weak_ptr<Task>& p) {
                                     auto sp = p.lock();
                                     return !sp || sp.get() == this;
                                 }),
                  vec.end());
    }
}

bool Task::TryFastSubmit() {
    std::lock_guard lock(mutex_);
    if (state_ != State::Created) {
        return false;
    }
    if (dependency_count_ != 0 || active_triggers_ != 0 || fired_trigger_ || failed_dependency_) {
        return false;
    }
    submit_requested_ = true;
    state_ = State::Queued;
    return true;
}