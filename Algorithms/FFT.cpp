#include <cmath>
#include <complex>
#include <iostream>
#include <vector>

void FFT(std::vector<std::complex<long double>>& values, size_t degree,
         long double root_degree, size_t start, size_t finish) {
  if (degree == 1) {
    return;
  }
  FFT(values, degree / 2, root_degree / 2, start, start + (finish - start) / 2);
  FFT(values, degree / 2, root_degree / 2, start + (finish - start) / 2,
      finish);
  std::complex<long double> root(std::cos(2 * M_PI / root_degree),
                                 std::sin(2 * M_PI / root_degree));
  std::complex<long double> multiplier = 1;
  size_t middle = start + (finish - start) / 2;
  for (size_t i = 0; i < degree / 2; ++i) {
    auto first = values[start + i];
    auto second = values[middle + i];
    values[start + i] = first + multiplier * second;
    values[middle + i] = first - multiplier * second;
    multiplier *= root;
  }
}

size_t TwoDegree(size_t number) {
  size_t current = 1;
  while (current <= number) {
    current *= 2;
  }
  return current;
}

size_t Logarithm(size_t number) {
  size_t degree = 0;
  while (number > 1) {
    number /= 2;
    ++degree;
  }
  return degree;
}

size_t ReverseBits(size_t index, size_t degree) {
  for (size_t i = 0; i < degree / 2; ++i) {
    size_t small_bit = ((index & (1ULL << i)) > 0 ? 1 : 0);
    size_t big_bit = ((index & (1ULL << (degree - 1 - i))) > 0 ? 1 : 0);
    if (small_bit == 1) {
      index |= small_bit << (degree - 1 - i);
    } else if (big_bit == 1) {
      index ^= big_bit << (degree - 1 - i);
    }
    if (big_bit == 1) {
      index |= big_bit << i;
    } else if (small_bit == 1) {
      index ^= small_bit << i;
    }
  }
  return index;
}

void RearrangePolynom(std::vector<std::complex<long double>>& polynom,
                      size_t degree) {
  std::vector<bool> swaped(polynom.size(), false);
  for (size_t i = 0; i < polynom.size(); ++i) {
    if (swaped[i]) {
      continue;
    }
    size_t reversed = ReverseBits(i, degree);
    std::swap(polynom[i], polynom[reversed]);
    swaped[i] = true;
    swaped[reversed] = true;
  }
}

void ComplementPolynom(std::vector<std::complex<long double>>& polynom,
                       const std::vector<long double>& original) {
  for (size_t i = 0; i < polynom.size(); ++i) {
    if (i < original.size()) {
      polynom[i] = {original[i], 0};
    } else {
      polynom[i] = 0;
    }
  }
}

std::vector<int> MultiplyPolynoms(
    const std::vector<long double>& first_coefficients,
    const std::vector<long double>& second_coefficients) {
  size_t new_degree =
      TwoDegree(first_coefficients.size() - 1 + second_coefficients.size() - 1);
  size_t degree = Logarithm(new_degree);
  std::vector<std::complex<long double>> first(new_degree);
  std::vector<std::complex<long double>> second(new_degree);
  std::vector<std::complex<long double>> multiplied(new_degree);
  ComplementPolynom(first, first_coefficients);
  ComplementPolynom(second, second_coefficients);
  RearrangePolynom(first, degree);
  RearrangePolynom(second, degree);

  FFT(first, new_degree, new_degree, 0, new_degree);
  FFT(second, new_degree, new_degree, 0, new_degree);
  for (size_t i = 0; i < new_degree; ++i) {
    multiplied[i] = first[i] * second[i];
  }
  RearrangePolynom(multiplied, degree);
  FFT(multiplied, new_degree, -static_cast<long double>(new_degree), 0,
      new_degree);

  std::vector<int> multiplied_coefficients(first_coefficients.size() +
                                           second_coefficients.size() - 1);
  std::complex<long double> cnew_degree(new_degree, 0);
  for (size_t i = 0; i < multiplied_coefficients.size(); ++i) {
    multiplied_coefficients[i] =
        std::round((multiplied[i] / cnew_degree).real());
  }
  return multiplied_coefficients;
}

int main() {
  size_t first_size;
  std::cin >> first_size;
  ++first_size;
  std::vector<long double> first_coefficients(first_size);
  for (size_t i = 0; i < first_size; ++i) {
    std::cin >> first_coefficients[first_size - 1 - i];
  }
  size_t second_size;
  std::cin >> second_size;
  ++second_size;
  std::vector<long double> second_coefficients(second_size);
  for (size_t i = 0; i < second_size; ++i) {
    std::cin >> second_coefficients[second_size - 1 - i];
  }

  std::vector<int> multiplied =
      MultiplyPolynoms(first_coefficients, second_coefficients);
  std::cout << multiplied.size() - 1 << ' ';
  for (size_t i = 0; i < multiplied.size(); ++i) {
    std::cout << multiplied[multiplied.size() - 1 - i] << ' ';
  }
}