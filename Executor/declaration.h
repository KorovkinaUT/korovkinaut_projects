#pragma once

#include <chrono>
#include <memory>

class Task;
class Executor;
using TaskPtr = std::shared_ptr<Task>;
using TimePoint = std::chrono::system_clock::time_point;

template <class T>
class TaskWithResult;
template <class T>
using TaskWithResultPtr = std::shared_ptr<TaskWithResult<T>>;