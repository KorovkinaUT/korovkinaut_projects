#pragma once

#include "dist.h"

#include <random>
#include <vector>
#include <algorithm>
#include <numeric>
#include <filesystem>
#include <cerrno>
#include <system_error>
#include <chrono>
#include <fstream>
#include <array>
#include <string_view>
#include <cstddef>

#ifdef __linux__
#include <sys/time.h>
#include <sys/resource.h>
#include <unistd.h>
#endif

class RandomGenerator {
public:
    explicit RandomGenerator(uint32_t seed = 738'547'485u) : gen_{seed} {
    }

    template <class T>
    std::vector<T> GenIntegralVector(size_t count, T from, T to) {
        UniformIntDistribution dist{from, to};
        std::vector<T> result(count);
        for (auto& cur : result) {
            cur = dist(gen_);
        }
        return result;
    }

    std::string GenString(size_t count, char from, char to) {
        UniformIntDistribution<int> dist{from, to};
        std::string result(count, from);
        for (auto& x : result) {
            x = dist(gen_);
        }
        return result;
    }

    std::string GenString(size_t count) {
        return GenString(count, 'a', 'z');
    }

    template <size_t N>
    std::array<double, N> GenRealArray(double from, double to) {
        UniformRealDistribution dist{from, to};
        std::array<double, N> result;
        for (auto& x : result) {
            x = dist(gen_);
        }
        return result;
    }

    std::vector<int> GenPermutation(size_t count) {
        std::vector<int> result(count);
        std::iota(result.begin(), result.end(), 0);
        std::ranges::shuffle(result, gen_);
        return result;
    }

    template <class T>
    T GenInt(T from, T to) {
        UniformIntDistribution dist{from, to};
        return dist(gen_);
    }

    template <class T>
    T GenInt() {
        return GenInt(std::numeric_limits<T>::min(), std::numeric_limits<T>::max());
    }

    char GenChar(char from, char to) {
        UniformIntDistribution<int> dist{from, to};
        return dist(gen_);
    }

    char GenChar() {
        return GenChar('a', 'z');
    }

    template <class Iterator>
    void Shuffle(Iterator first, Iterator last) {
        std::shuffle(first, last, gen_);
    }

private:
    std::mt19937 gen_;
};

inline std::filesystem::path GetFileDir(std::string file) {
    std::filesystem::path path{std::move(file)};
    if (path.is_absolute() && std::filesystem::is_regular_file(path)) {
        return path.parent_path();
    } else {
        throw std::runtime_error{"Bad file path"};
    }
}

inline std::filesystem::path GetRelativeDir(std::string_view file_path,
                                            std::string_view relative_path) {
    auto path = std::filesystem::path{file_path}.parent_path() / relative_path;
    if (path.is_absolute() && std::filesystem::is_directory(path)) {
        return path;
    } else {
        throw std::runtime_error{"Bad dir path"};
    }
}

class Timer {
    using Clock = std::chrono::steady_clock;
    struct Times {
        Clock::duration wall_time;
        std::chrono::microseconds cpu_time;
    };

public:
    Times GetTimes() const {
        return {Clock::now() - wall_start_, GetCPUTime() - cpu_start_};
    }

private:
    static std::chrono::microseconds GetCPUTime() {
#ifdef __linux__
        if (rusage usage; ::getrusage(RUSAGE_SELF, &usage)) {
            throw std::system_error{errno, std::generic_category()};
        } else {
            auto time = usage.ru_utime;
            return std::chrono::microseconds{1'000'000ll * time.tv_sec + time.tv_usec};
        }
#else
        auto time = Clock::now().time_since_epoch();
        return std::chrono::duration_cast<std::chrono::microseconds>(time);
#endif
    }

    const std::chrono::time_point<Clock> wall_start_ = Clock::now();
    const std::chrono::microseconds cpu_start_ = GetCPUTime();
};

#ifdef __linux__

inline int64_t GetMemoryUsage() {
    if (rusage usage; ::getrusage(RUSAGE_SELF, &usage)) {
        throw std::system_error{errno, std::generic_category()};
    } else {
        return usage.ru_maxrss;
    }
}

class MemoryGuard {
public:
    explicit MemoryGuard(size_t bytes) {
        if (is_active) {
            throw std::runtime_error{"There is an active memory guard"};
        }
        is_active = true;
        bytes += GetDataMemoryUsage();
        if (rlimit limit{bytes, RLIM_INFINITY}; ::setrlimit(RLIMIT_DATA, &limit)) {
            throw std::system_error{errno, std::generic_category()};
        }
    }

    ~MemoryGuard() {
        is_active = false;
        rlimit limit{RLIM_INFINITY, RLIM_INFINITY};
        ::setrlimit(RLIMIT_DATA, &limit);
    }

    MemoryGuard(const MemoryGuard&) = delete;
    MemoryGuard& operator=(const MemoryGuard&) = delete;

private:
    static size_t GetDataMemoryUsage() {
        size_t pages;
        std::ifstream in{"/proc/self/statm"};
        for (auto i = 0; i < 6; ++i) {
            in >> pages;
        }
        if (!in) {
            throw std::runtime_error{"Failed to get number of pages"};
        }
        return pages * kPageSize;
    }

    static inline const auto kPageSize = ::getpagesize();
    static inline auto is_active = false;
};

template <class T = std::byte>
MemoryGuard MakeMemoryGuard(size_t n) {
    return MemoryGuard{n * sizeof(T)};
}

#else

struct DummyGuard {
    ~DummyGuard() {
    }
};

template <class T = std::byte>
DummyGuard MakeMemoryGuard(size_t) {
    return {};
}

#endif
