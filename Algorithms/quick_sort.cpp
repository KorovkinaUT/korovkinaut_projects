#include <iostream>
#include <vector>

size_t SecondPartition(std::vector<int>& array, size_t start, size_t finish) {
  int pivot = array[finish];
  size_t left = start - 1;
  for (size_t i = start; i <= finish; ++i) {
    if (array[i] <= pivot) {
      ++left;
      std::swap(array[left], array[i]);
    }
  }
  return left;
}

size_t FirstPartition(std::vector<int>& array, size_t start, size_t finish) {
  int pivot = array[finish];
  size_t left = start - 1;
  for (size_t i = start; i <= finish; ++i) {
    if (array[i] < pivot) {
      ++left;
      std::swap(array[left], array[i]);
    }
  }
  return left;
}

void QuickSort(std::vector<int>& array, size_t start, size_t finish) {
  if (start >= finish) {
    return;
  }
  size_t left = FirstPartition(array, start, finish);
  size_t right = SecondPartition(array, left + 1, finish);
  QuickSort(array, start, left);
  QuickSort(array, right + 1, finish);
}

int main() {
  size_t number = 0;
  std::cin >> number;
  std::vector<int> array(number + 1);
  for (size_t i = 1; i <= number; ++i) {
    std::cin >> array[i];
  }
  QuickSort(array, 1, number);
  for (size_t i = 1; i <= number; ++i) {
    std::cout << array[i] << ' ';
  }
}