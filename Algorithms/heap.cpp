#include <iostream>
#include <string>
#include <vector>

class MinimumHeap {
 private:
  size_t size_;
  std::vector<std::pair<long long, size_t>> data_;
  std::vector<size_t> add_request_;
  void SiftUp(size_t index);
  void SiftDown(size_t index);

 public:
  MinimumHeap(size_t request_number) : size_(0) {
    for (size_t i = 0; i < request_number; ++i) {
      add_request_.push_back(INT32_MAX);
    }
    data_.push_back(std::pair(-1, 0));
  }
  void GetMin() { std::cout << data_[1].first << '\n'; }
  void Insert(long long element, size_t add_time);
  void ExtractMin();

  void DecreaseKey(size_t add_time, long long delta) {
    data_[add_request_[add_time]].first -= delta;
    SiftUp(add_request_[add_time]);
  }
};

void MinimumHeap::SiftUp(size_t index) {
  if (index <= 1) {
    return;
  }
  if (data_[index].first < data_[index / 2].first) {
    size_t parent_addtime = data_[index / 2].second;
    add_request_[parent_addtime] = index;
    add_request_[data_[index].second] = index / 2;
    std::swap(data_[index], data_[index / 2]);
    SiftUp(index / 2);
  }
}

void MinimumHeap::SiftDown(size_t index) {
  if (2 * index > size_) {
    return;
  }
  size_t min_child = 2 * index;
  if (2 * index + 1 <= size_ and
      data_[2 * index + 1].first < data_[min_child].first) {
    ++min_child;
  }
  if (data_[min_child] < data_[index]) {
    size_t child_addtime = data_[min_child].second;
    add_request_[child_addtime] = index;
    add_request_[data_[index].second] = min_child;
    std::swap(data_[index], data_[min_child]);
    SiftDown(min_child);
  }
}

void MinimumHeap::Insert(long long element, size_t add_time) {
  data_.push_back(std::pair(element, add_time));
  ++size_;
  data_[size_] = std::pair(element, add_time);
  add_request_[add_time] = size_;
  SiftUp(size_);
}

void MinimumHeap::ExtractMin() {
  std::swap(data_[1], data_[size_]);
  size_t add_time = data_[1].second;
  add_request_[add_time] = 1;
  --size_;
  SiftDown(1);
}

int main() {
  std::ios_base::sync_with_stdio(false);
  std::cin.tie(0);
  std::cout.tie(0);
  size_t request_number;
  std::cin >> request_number;
  MinimumHeap heap(request_number);
  std::string request;
  for (size_t i = 0; i < request_number; ++i) {
    std::cin >> request;
    if (request == "getMin") {
      heap.GetMin();
    } else if (request == "extractMin") {
      heap.ExtractMin();
    } else if (request == "insert") {
      long long element;
      std::cin >> element;
      heap.Insert(element, i + 1);
    } else if (request == "decreaseKey") {
      size_t add_time;
      long long delta;
      std::cin >> add_time >> delta;
      heap.DecreaseKey(add_time, delta);
    }
  }
  return 0;
}