#pragma once

#include "declaration.h"
#include "future.h"

#include <atomic>
#include <condition_variable>
#include <cstdint>
#include <functional>
#include <memory>
#include <queue>
#include <thread>
#include <vector>

class Executor : public std::enable_shared_from_this<Executor> {
    friend class Task;

public:
    Executor();
    explicit Executor(uint32_t num_threads);
    ~Executor();

    void Start();
    void Stop();
    void WaitStop();

    void Submit(TaskPtr task);
    void WaitIdle();

    void SubmitAfter(TimePoint tp, TaskPtr task);
    void SubmitAfter(TaskPtr trigger, TaskPtr task);

    template <class T>
    TaskWithResultPtr<T> Invoke(std::function<T()> fn);

    template <class Y, class T>
    TaskWithResultPtr<Y> Then(TaskWithResultPtr<T> input, std::function<Y()> fn);

    template <class T>
    TaskWithResultPtr<std::vector<T>> WhenAll(std::vector<TaskWithResultPtr<T>> all);

    template <class T>
    TaskWithResultPtr<T> WhenAny(std::vector<TaskWithResultPtr<T>> all);

    template <class T>
    TaskWithResultPtr<std::vector<T>> WhenAllBeforeDeadline(std::vector<TaskWithResultPtr<T>> all,
                                                            TimePoint deadline);

private:
    struct TimerEntry {
        TimePoint at;
        TaskPtr task;

        bool operator<(const TimerEntry& other) const {
            return at > other.at;
        }
    };

    void WorkerLoop();
    void TimerLoop();
    void OnTaskReady(TaskPtr task);

private:
    uint32_t num_threads_ = 0;
    std::vector<std::thread> threads_;
    std::thread timer_thread_;

    std::mutex mutex_;
    std::condition_variable cv_;
    std::condition_variable idle_cv_;
    std::queue<TaskPtr> ready_queue_;
    size_t in_flight_ = 0;

    std::mutex timer_mutex_;
    std::condition_variable timer_cv_;
    std::priority_queue<TimerEntry> timers_;

    bool started_ = false;
    bool timer_started_ = false;
    bool joined_ = false;
    std::atomic<bool> stopping_{false};
};

std::shared_ptr<Executor> MakeThreadPoolExecutor(uint32_t num_threads);

template <class T>
TaskWithResultPtr<T> Executor::Invoke(std::function<T()> fn) {
    auto task = std::make_shared<TaskWithResult<T>>(std::move(fn));
    Submit(task);
    return task;
}

template <class Y, class T>
TaskWithResultPtr<Y> Executor::Then(TaskWithResultPtr<T> input, std::function<Y()> fn) {
    auto task = std::make_shared<TaskWithResult<Y>>(std::move(fn));
    task->AddDependency(input);
    Submit(task);
    return task;
}

template <class T>
TaskWithResultPtr<std::vector<T>> Executor::WhenAll(std::vector<TaskWithResultPtr<T>> all) {
    auto task = std::make_shared<TaskWithResult<std::vector<T>>>([all]() mutable {
        std::vector<T> result;
        result.reserve(all.size());
        for (auto& future : all) {
            result.push_back(future->Get());
        }
        return result;
    });

    for (auto& future : all) {
        task->AddDependency(future);
    }

    Submit(task);
    return task;
}

template <class T>
TaskWithResultPtr<T> Executor::WhenAny(std::vector<TaskWithResultPtr<T>> all) {
    auto task = std::make_shared<TaskWithResult<T>>([all]() mutable {
        for (;;) {
            for (auto& future : all) {
                if (future->Ready()) {
                    return future->Get();
                }
            }
            std::this_thread::yield();
        }
    });

    for (auto& future : all) {
        SubmitAfter(future, task);
    }

    return task;
}

template <class T>
TaskWithResultPtr<std::vector<T>> Executor::WhenAllBeforeDeadline(
    std::vector<TaskWithResultPtr<T>> all, TimePoint deadline) {
    auto task = std::make_shared<TaskWithResult<std::vector<T>>>(
        [all = std::move(all), deadline]() mutable {
            while (std::chrono::system_clock::now() < deadline) {
                bool all_ready = true;
                for (auto& future : all) {
                    if (!future->Ready()) {
                        all_ready = false;
                        break;
                    }
                }
                if (all_ready) {
                    break;
                }
                std::this_thread::yield();
            }

            std::vector<T> result;
            result.reserve(all.size());
            for (auto& future : all) {
                if (future->Ready()) {
                    result.push_back(future->Get());
                }
            }
            return result;
        });

    Submit(task);
    return task;
}