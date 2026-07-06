#pragma once

#include "task.h"

#include <chrono>
#include <functional>
#include <future>
#include <stdexcept>
#include <utility>

struct Unit {};

template <class T>
class TaskWithResult : public Task {
    friend class Executor;

public:
    explicit TaskWithResult(std::function<T()> fn)
        : fn_(std::move(fn)), future_(promise_.get_future().share()) {
    }

    void Run() override {
        promise_.set_value(fn_());
    }

    bool Ready() {
        if (!IsFinished() || IsCancelled() || GetError() != nullptr) {
            return false;
        }
        return future_.wait_for(std::chrono::seconds(0)) == std::future_status::ready;
    }

    T Get() {
        Wait();
        if (auto error = GetError()) {
            std::rethrow_exception(error);
        }
        if (IsCancelled()) {
            throw std::runtime_error("task cancelled");
        }
        return future_.get();
    }

protected:
    void Discard() override {
        fn_ = {};
    }

    bool NeedsDetachFromSources() const override {
        return true;
    }

private:
    std::function<T()> fn_;
    std::promise<T> promise_;
    std::shared_future<T> future_;
};