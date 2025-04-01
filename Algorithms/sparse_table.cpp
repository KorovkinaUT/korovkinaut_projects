#include <algorithm>
#include <iostream>
#include <limits>
#include <utility>
#include <vector>

class SparceTable {
 public:
  SparceTable(size_t length, const std::vector<int>& array)
      : exponent_(std::vector<size_t>(length + 1)) {
    FillExponent(length);
    sparce_table_.resize(
        exponent_[length] + 1,
        std::vector<std::pair<std::pair<int, size_t>, std::pair<int, size_t>>>(
            length));
    FillSparceTable(array);
  }
  int SecondStatistic(size_t left, size_t right) const;

 private:
  std::vector<size_t> exponent_;
  std::vector<
      std::vector<std::pair<std::pair<int, size_t>, std::pair<int, size_t>>>>
      sparce_table_;
  void FillExponent(size_t length);
  void FillSparceTable(const std::vector<int>& array);
};

void SparceTable::FillExponent(size_t length) {
  for (size_t i = 2; i <= length; ++i) {
    exponent_[i] = exponent_[i - 1];
    if ((i & (1 << exponent_[i])) == 0) {
      ++exponent_[i];
    }
  }
}

void SparceTable::FillSparceTable(const std::vector<int>& array) {
  size_t length = array.size();
  for (size_t i = 0; i < length - 1; ++i) {
    sparce_table_[1][i] = std::make_pair(std::make_pair(array[i], i),
                                         std::make_pair(array[i + 1], i + 1));
  }
  for (size_t j = 2; j <= exponent_[length]; ++j) {
    for (size_t i = 0; i < length - (1 << j) + 1; ++i) {
      auto first_half = sparce_table_[j - 1][i];
      auto second_half = sparce_table_[j - 1][i + (1 << (j - 1))];
      std::vector<std::pair<int, size_t>> to_sort = {
          first_half.first, first_half.second, second_half.first,
          second_half.second};
      sort(to_sort.begin(), to_sort.end());
      sparce_table_[j][i] =
          std::pair<std::pair<int, size_t>, std::pair<int, size_t>>(to_sort[0],
                                                                    to_sort[1]);
    }
  }
}

int SparceTable::SecondStatistic(size_t left, size_t right) const {
  const int kVeryBig = std::numeric_limits<int>::max();
  size_t power = exponent_[right - left + 1];
  auto first_half = sparce_table_[power][left];
  auto second_half = sparce_table_[power][right - (1 << power) + 1];
  std::vector<std::pair<int, size_t>> to_sort = {
      first_half.first, first_half.second, second_half.first,
      second_half.second};
  sort(to_sort.begin(), to_sort.end());
  for (size_t ind = 1; ind < 4; ++ind) {
    if (to_sort[ind].second == to_sort[ind - 1].second) {
      to_sort[ind].first = kVeryBig;
    }
  }
  sort(to_sort.begin(), to_sort.end());
  return to_sort[1].first;
}

int main() {
  size_t length;
  size_t requests;
  std::cin >> length >> requests;
  std::vector<int> array(length);
  for (size_t i = 0; i < length; ++i) {
    std::cin >> array[i];
  }
  SparceTable sparce_table(length, array);

  for (size_t i = 0; i < requests; ++i) {
    size_t left;
    size_t right;
    std::cin >> left >> right;
    std::cout << sparce_table.SecondStatistic(--left, --right) << '\n';
  }
  return 0;
}