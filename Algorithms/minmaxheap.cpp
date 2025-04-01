#include <iostream>
#include <string>
#include <vector>

class MinMaxHeap {
 private:
  size_t size_ = 0;
  std::vector<long long> min_heap_ = {-1};
  std::vector<long long> max_heap_ = {-1};
  std::vector<size_t> max_indexes_ = {0};
  std::vector<size_t> min_indexes_ = {0};
  static bool Comparison(long long first, long long second, int type);
  void ChangeSecond(int type, size_t index1, size_t index2);
  void SiftDown(size_t index, int type, std::vector<long long>& data);
  void SiftUp(size_t index, int type, std::vector<long long>& data);

 public:
  void GetMin() const { std::cout << min_heap_[1] << '\n'; }
  void GetMax() const { std::cout << max_heap_[1] << '\n'; }
  void Size() const { std::cout << size_ << '\n'; }
  void Clear();
  void Insert(long long element);
  void ExtractMaxMin();
  void ExtractMinMax();
  void ExtractMin();
  void ExtractMax();

  bool IsEmpty() const {
    if (size_ == 0) {
      std::cout << "error" << '\n';
      return false;
    }
    return true;
  }
};

bool MinMaxHeap::Comparison(long long first, long long second, int type) {
  if (type == 1) {
    return first < second;
  }
  return first > second;
}

void MinMaxHeap::ChangeSecond(int type, size_t index1, size_t index2) {
  if (type == -1) {
    min_indexes_[max_indexes_[index1]] = index2;
    min_indexes_[max_indexes_[index2]] = index1;
    std::swap(max_indexes_[index1], max_indexes_[index2]);
  } else {
    max_indexes_[min_indexes_[index1]] = index2;
    max_indexes_[min_indexes_[index2]] = index1;
    std::swap(min_indexes_[index1], min_indexes_[index2]);
  }
}

void MinMaxHeap::SiftDown(size_t index, int type,
                          std::vector<long long>& data) {
  if (2 * index > size_) {
    return;
  }
  size_t child = 2 * index;
  if (2 * index + 1 <= size_ and
      Comparison(data[2 * index], data[2 * index + 1], type)) {
    child = 2 * index + 1;
  }
  if (Comparison(data[index], data[child], type)) {
    ChangeSecond(type, index, child);
    std::swap(data[index], data[child]);
    SiftDown(child, type, data);
  }
}

void MinMaxHeap::SiftUp(size_t index, int type, std::vector<long long>& data) {
  if (index == 1) {
    return;
  }
  if (Comparison(data[index / 2], data[index], type)) {
    ChangeSecond(type, index, index / 2);
    std::swap(data[index / 2], data[index]);
    SiftUp(index / 2, type, data);
  }
}

void MinMaxHeap::Clear() {
  size_ = 0;
  min_heap_.clear();
  max_heap_.clear();
  min_indexes_.clear();
  max_indexes_.clear();
  min_heap_.push_back(-1);
  max_heap_.push_back(-1);
  min_indexes_.push_back(0);
  max_indexes_.push_back(0);
  std::cout << "ok" << '\n';
}

void MinMaxHeap::Insert(long long element) {
  ++size_;
  min_heap_.push_back(element);
  max_heap_.push_back(element);
  min_heap_[size_] = element;
  max_heap_[size_] = element;
  max_indexes_.push_back(size_);
  min_indexes_.push_back(size_);
  max_indexes_[size_] = size_;
  min_indexes_[size_] = size_;
  SiftUp(size_, 1, max_heap_);
  SiftUp(size_, -1, min_heap_);
  std::cout << "ok" << '\n';
}

void MinMaxHeap::ExtractMaxMin() {
  max_indexes_[min_indexes_[size_]] = 1;
  max_heap_[1] = max_heap_[size_];
  min_indexes_[1] = min_indexes_[size_];
  SiftDown(1, 1, max_heap_);
}

void MinMaxHeap::ExtractMinMax() {
  min_indexes_[max_indexes_[size_]] = 1;
  min_heap_[1] = min_heap_[size_];
  max_indexes_[1] = max_indexes_[size_];
  SiftDown(1, -1, min_heap_);
}

void MinMaxHeap::ExtractMin() {
  std::cout << min_heap_[1] << '\n';
  max_heap_[max_indexes_[1]] = INT32_MAX;
  SiftUp(max_indexes_[1], 1, max_heap_);
  ExtractMaxMin();
  min_indexes_[max_indexes_[size_]] = 1;
  min_heap_[1] = min_heap_[size_];
  max_indexes_[1] = max_indexes_[size_];
  --size_;
  if (size_ > 1) {
    SiftDown(1, -1, min_heap_);
  }
}

void MinMaxHeap::ExtractMax() {
  std::cout << max_heap_[1] << '\n';
  min_heap_[min_indexes_[1]] = INT32_MIN;
  SiftUp(min_indexes_[1], -1, min_heap_);
  ExtractMinMax();
  max_indexes_[min_indexes_[size_]] = 1;
  max_heap_[1] = max_heap_[size_];
  min_indexes_[1] = min_indexes_[size_];
  --size_;
  if (size_ > 1) {
    SiftDown(1, 1, max_heap_);
  }
}

int main() {
  std::ios_base::sync_with_stdio(false);
  std::cin.tie(0);
  std::cout.tie(0);
  size_t number;
  std::cin >> number;
  std::string request;
  MinMaxHeap data;
  for (size_t i = 0; i < number; ++i) {
    std::cin >> request;
    if (request == "extract_min") {
      if (data.IsEmpty()) {
        data.ExtractMin();
      }
    } else if (request == "extract_max") {
      if (data.IsEmpty()) {
        data.ExtractMax();
      }
    } else if (request == "get_min") {
      if (data.IsEmpty()) {
        data.GetMin();
      }
    } else if (request == "get_max") {
      if (data.IsEmpty()) {
        data.GetMax();
      }
    } else if (request == "insert") {
      long long element;
      std::cin >> element;
      data.Insert(element);
    } else if (request == "size") {
      data.Size();
    } else if (request == "clear") {
      data.Clear();
    }
  }
  return 0;
}