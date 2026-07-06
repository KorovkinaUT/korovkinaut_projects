#include "executor.h"

#include <utility>
#include <vector>

Executor::Executor() = default;

Executor::Executor(uint32_t num_threads) : num_threads_(num_threads) {
}

Executor::~Executor() {
    Stop();
    WaitStop();
}

void Executor::Start() {
    std::lock_guard lock(mutex_);
    if (started_) {
        return;
    }

    started_ = true;
    stopping_.store(false, std::memory_order_release);

    threads_.reserve(num_threads_);
    for (uint32_t i = 0; i < num_threads_; ++i) {
        threads_.emplace_back([this] { WorkerLoop(); });
    }
}

void Executor::Stop() {
    std::queue<TaskPtr> queued;

    {
        std::lock_guard lock(mutex_);
        if (stopping_.exchange(true, std::memory_order_acq_rel)) {
            return;
        }
        queued.swap(ready_queue_);
        if (in_flight_ >= queued.size()) {
            in_flight_ -= queued.size();
        } else {
            in_flight_ = 0;
        }
        if (in_flight_ == 0) {
            idle_cv_.notify_all();
        }
    }

    while (!queued.empty()) {
        queued.front()->Cancel();
        queued.pop();
    }

    cv_.notify_all();
    timer_cv_.notify_all();
}

void Executor::WaitStop() {
    std::vector<std::thread> workers;
    std::thread timer;

    {
        std::lock_guard lock(mutex_);
        if (joined_) {
            return;
        }
        joined_ = true;
        workers.swap(threads_);
        timer = std::move(timer_thread_);
    }

    for (auto& worker : workers) {
        if (worker.joinable()) {
            worker.join();
        }
    }
    if (timer.joinable()) {
        timer.join();
    }
}

void Executor::Submit(TaskPtr task) {
    {
        std::lock_guard lock(mutex_);
        if (!started_ || stopping_.load(std::memory_order_acquire)) {
            task->Cancel();
            return;
        }

        if (task->TryFastSubmit()) {
            ready_queue_.push(std::move(task));
            ++in_flight_;
            cv_.notify_one();
            return;
        }
    }

    task->RequestSubmit(shared_from_this());
}

void Executor::WaitIdle() {
    std::unique_lock lock(mutex_);
    idle_cv_.wait(lock, [this] { return in_flight_ == 0; });
}

void Executor::SubmitAfter(TimePoint tp, TaskPtr task) {
    bool need_start_timer = false;

    {
        std::lock_guard lock(mutex_);
        if (!started_ || stopping_.load(std::memory_order_acquire)) {
            task->Cancel();
            return;
        }
        if (!timer_started_) {
            timer_started_ = true;
            need_start_timer = true;
        }
    }

    if (need_start_timer) {
        timer_thread_ = std::thread([this] { TimerLoop(); });
    }

    task->AddTimeTrigger(shared_from_this());

    {
        std::lock_guard lock(timer_mutex_);
        timers_.push(TimerEntry{tp, std::move(task)});
    }
    timer_cv_.notify_one();
}

void Executor::SubmitAfter(TaskPtr trigger, TaskPtr task) {
    {
        std::lock_guard lock(mutex_);
        if (!started_ || stopping_.load(std::memory_order_acquire)) {
            task->Cancel();
            return;
        }
    }
    task->AddTaskTrigger(shared_from_this(), trigger);
}

void Executor::WorkerLoop() {
    for (;;) {
        TaskPtr task;

        {
            std::unique_lock lock(mutex_);
            cv_.wait(lock, [this] {
                return stopping_.load(std::memory_order_acquire) || !ready_queue_.empty();
            });

            if (ready_queue_.empty()) {
                if (stopping_.load(std::memory_order_acquire)) {
                    return;
                }
                continue;
            }

            task = std::move(ready_queue_.front());
            ready_queue_.pop();
        }

        if (!task->TryRun()) {
            task.reset();
            std::lock_guard lock(mutex_);
            if (in_flight_ > 0) {
                --in_flight_;
            }
            if (in_flight_ == 0) {
                idle_cv_.notify_all();
            }
            continue;
        }

        try {
            task->Run();
            task->FinishSuccess();
        } catch (...) {
            task->FinishFailure(std::current_exception());
        }

        task.reset();

        {
            std::lock_guard lock(mutex_);
            if (in_flight_ > 0) {
                --in_flight_;
            }
            if (in_flight_ == 0) {
                idle_cv_.notify_all();
            }
        }
    }
}

void Executor::TimerLoop() {
    std::unique_lock lock(timer_mutex_);

    for (;;) {
        if (stopping_.load(std::memory_order_acquire)) {
            return;
        }

        if (timers_.empty()) {
            timer_cv_.wait(lock, [this] {
                return stopping_.load(std::memory_order_acquire) || !timers_.empty();
            });
            if (stopping_.load(std::memory_order_acquire)) {
                return;
            }
        }

        if (timers_.empty()) {
            continue;
        }

        const auto at = timers_.top().at;
        timer_cv_.wait_until(lock, at, [this, at] {
            return stopping_.load(std::memory_order_acquire) || timers_.empty() ||
                   timers_.top().at != at;
        });

        if (stopping_.load(std::memory_order_acquire)) {
            return;
        }

        if (timers_.empty()) {
            continue;
        }

        const auto now = std::chrono::system_clock::now();
        if (timers_.top().at > now) {
            continue;
        }

        std::vector<TaskPtr> fired;
        while (!timers_.empty() && timers_.top().at <= now) {
            fired.push_back(std::move(timers_.top().task));
            timers_.pop();
        }

        lock.unlock();
        for (auto& task : fired) {
            task->FireTimeTrigger();
        }
        lock.lock();
    }
}

void Executor::OnTaskReady(TaskPtr task) {
    bool cancel = false;

    {
        std::lock_guard lock(mutex_);
        if (stopping_.load(std::memory_order_acquire)) {
            cancel = true;
        } else {
            ready_queue_.push(std::move(task));
            ++in_flight_;
            cv_.notify_one();
        }
    }

    if (cancel) {
        task->Cancel();
    }
}

std::shared_ptr<Executor> MakeThreadPoolExecutor(uint32_t num_threads) {
    return std::make_shared<Executor>(num_threads);
}