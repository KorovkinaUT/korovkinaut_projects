#pragma once

#include "declaration.h"

#include <condition_variable>
#include <exception>
#include <memory>
#include <mutex>
#include <vector>

class Task : public std::enable_shared_from_this<Task> {
    friend class Executor;

public:
    virtual ~Task() = default;
    virtual void Run() = 0;

    void AddDependency(TaskPtr dep);
    void Cancel();

    std::exception_ptr GetError();
    bool IsFinished() const;
    bool IsCancelled() const;
    void Wait();

protected:
    enum class State { Created, Queued, Running, Completed, Failed, Cancelled };

    virtual void Discard() {
    }

    virtual bool NeedsDetachFromSources() const {
        return false;
    }

private:
    void RequestSubmit(const std::shared_ptr<Executor>& owner);
    void AddTimeTrigger(const std::shared_ptr<Executor>& owner);
    void AddTaskTrigger(const std::shared_ptr<Executor>& owner, const TaskPtr& trigger);

    void FireTimeTrigger();
    void OnTriggerResolved(bool fired);
    void OnDependencyResolved(bool success);

    bool TryFastSubmit();
    bool TryRun();
    void FinishSuccess();
    void FinishFailure(std::exception_ptr error);

    bool ShouldStartLocked() const;
    bool ShouldCancelLocked() const;
    bool IsTerminalLocked() const;

    std::vector<std::shared_ptr<Task>> CollectDependentsLocked();
    std::vector<std::shared_ptr<Task>> CollectTriggerTargetsLocked();
    void DetachFromSources();

private:
    mutable std::mutex mutex_;
    std::condition_variable cv_;
    State state_ = State::Created;
    std::exception_ptr error_;

    std::weak_ptr<Executor> owner_;
    bool submit_requested_ = false;
    bool fired_trigger_ = false;
    size_t active_triggers_ = 0;

    size_t dependency_count_ = 0;
    size_t unfinished_dependencies_ = 0;
    bool failed_dependency_ = false;

    std::vector<std::weak_ptr<Task>> dependents_;
    std::vector<std::weak_ptr<Task>> trigger_targets_;

    std::vector<std::weak_ptr<Task>> dependency_sources_;
    std::vector<std::weak_ptr<Task>> trigger_sources_;

    TaskPtr self_hold_;
};