#include <iostream>
#include <string>
#include <vector>

enum class Sign: int {
  positive = 1,
  negative = -1
};

class BigInteger {
 private:
  std::vector<long> digits = {0};
  Sign sign = Sign::positive;
  size_t length = 0;
  size_t capacity = 1;
  static const int Base = 1000000000;
  static const int DigitLength = 9;

  void ChangeLength();
  BigInteger& PositivePlus(const BigInteger& other, size_t shift);
  BigInteger& PositiveMinus(const BigInteger& other, size_t shift);
  BigInteger PositiveMultiply(int other);
  BigInteger& PositiveMultiply(const BigInteger& other);
  BigInteger PositiveDivide(std::vector<long>& quotient, const BigInteger& other, size_t prev);
  BigInteger& PositiveModule(const BigInteger& other);

 public:
  BigInteger()
      : digits(std::vector<long>(1)),
        sign(Sign::positive),
        length(0),
        capacity(1) {};
  BigInteger(std::string number);
  BigInteger(long long number) 
      : BigInteger(std::to_string(number)) {};
  ~BigInteger() = default;

  bool operator==(const BigInteger& other) const;
  bool operator!=(const BigInteger& other) const {
    return !(*this == other);
  }
  bool operator<(const BigInteger& other) const;
  bool operator<=(const BigInteger& other) const {
    return (*this < other) or (*this == other);
  }
  bool operator>(const BigInteger& other) const {
    return !(*this <= other);
  }
  bool operator>=(const BigInteger& other) const {
    return !(*this < other);
  }
  BigInteger operator-();
  BigInteger& operator++();
  BigInteger operator++(int);
  BigInteger& operator--();
  BigInteger operator--(int);
  BigInteger& operator+=(const BigInteger& other);
  BigInteger& operator-=(const BigInteger& other);
  BigInteger& operator*=(const BigInteger& other);
  BigInteger& operator/=(const BigInteger& other);
  BigInteger& operator%=(const BigInteger& other);
  std::string toString() const;
  explicit operator bool () const;
  Sign BigIntegerSign() const;
};

namespace detail {
  BigInteger BigIntegerAbs(BigInteger big_integer) {
    if (big_integer.BigIntegerSign() == Sign::negative) {
      return -big_integer;
    }
    return big_integer;
  }
}

using namespace detail;

void BigInteger::ChangeLength() {
  while (digits[length] == 0 and length > 0) {
    --length;
  }
}

BigInteger::BigInteger(std::string number)
    : digits(std::vector<long>(number.size() / DigitLength + 1)),
      sign(Sign::positive),
      length(0),
      capacity(number.size() / DigitLength + 1) {
  if (number[0] == '-') {
    sign = Sign::negative;
  }
  int i = number.size();
  for (; i - DigitLength >= 0; i -= DigitLength) {
    digits[length] = std::stoi(number.substr(i - DigitLength, DigitLength));
    ++length;
  }
  if (i > 0) {
    digits[length] = std::stoi(number.substr(0, i));
  }
  ChangeLength();
  if (digits[length] < 0) {
    digits[length] = -digits[length];
  }
}

bool BigInteger::operator==(const BigInteger& other) const {
  if (length != other.length) {
    return false;
  }
  if (length == 0 and digits[0] == 0 and other.digits[0] == 0) {
    return true;
  }
  if (sign != other.sign) {
    return false;
  }
  for (size_t i = 0; i <= length; ++i) {
    if (digits[i] != other.digits[i]) {
      return false;
    }
  }
  return true;
}

bool BigInteger::operator<(const BigInteger& other) const {
  if (sign < other.sign) {
    return true;
  }
  if (other.sign < sign) {
    return false;
  }
  if (length < other.length) {
    return sign == Sign::positive;
  }
  if (length > other.length) {
    return sign == Sign::negative;
  }
  for (int i = length; i >= 0; --i) {
    if (digits[i] != other.digits[i] and sign == Sign::positive) {
      return digits[i] < other.digits[i];
    }
    if (digits[i] != other.digits[i] and sign == Sign::negative) {
      return digits[i] > other.digits[i];
    }
  }
  return !(*this == other);
}

BigInteger& BigInteger::operator++() {
  *this += 1;
  return *this;
}

BigInteger BigInteger::operator++(int) {
  BigInteger new_big_integer = *this;
  ++*this;
  return new_big_integer;
}

BigInteger& BigInteger::operator--() {
  *this -= 1;
  return *this;
}

BigInteger BigInteger::operator--(int) {
  BigInteger new_big_integer = *this;
  --*this;
  return new_big_integer;
}

BigInteger& BigInteger::PositivePlus(const BigInteger& other, size_t shift) {
  long long to_next = 0;
  capacity = std::max(capacity, other.length + 2 + shift);
  digits.resize(capacity);
  size_t i = shift;
  for (; i - shift <= other.length; ++i) {
    to_next += digits[i] + other.digits[i - shift];
    digits[i] = to_next % Base;
    to_next /= Base;
  }
  while (i < capacity) {
    to_next += digits[i];
    digits[i] = to_next % Base;
    to_next /= Base;
    ++i;
  }
  length = i - 1;
  ChangeLength();
  return *this;
}

BigInteger& BigInteger::PositiveMinus(const BigInteger& other, size_t shift) {
  size_t i = shift;
  for (; i - shift <= std::min(length, other.length); ++i) {
    if (digits[i] < 0) {
      digits[i] = Base - 1;
      --digits[i + 1];
    }
    if (digits[i] < other.digits[i - shift]) {
      digits[i] += Base;
      --digits[i + 1];
    }
    digits[i] -= other.digits[i - shift];
  }
  while (i < length) {
    if (digits[i] < 0) {
      digits[i] = Base - 1;
      --digits[i + 1];
    }
    ++i;
  }
  ChangeLength();
  return *this;
}

BigInteger& BigInteger::operator+=(const BigInteger& other) {
  if (sign == other.sign) {
    return this->PositivePlus(other, 0);
  }
  if (sign == Sign::positive and other.sign == Sign::negative) {
    if (*this >= BigIntegerAbs(other)) {
      return this->PositiveMinus(other, 0);
    }
    BigInteger this_big_integer = *this;
    *this = other;
    return this->PositiveMinus(this_big_integer, 0);
  }
  if (BigIntegerAbs(*this) > other) {
    return this->PositiveMinus(other, 0);
  }
  BigInteger this_big_integer = *this;
  *this = other;
  return this->PositiveMinus(this_big_integer, 0);
}

BigInteger operator+(const BigInteger& first, const BigInteger& second) {
  BigInteger sum = first;
  sum += second;
  return sum;
}

BigInteger& BigInteger::operator-=(const BigInteger& other) {
  if (sign != other.sign) {
    return this->PositivePlus(other, 0);
  }
  if (sign == Sign::positive and other.sign == Sign::positive) {
    if (*this >= other) {
      return this->PositiveMinus(other, 0);
    }
    BigInteger this_big_integer = *this;
    *this = other;
    *this = this->PositiveMinus(this_big_integer, 0);
    sign = Sign::negative;
    return *this;
  }
  if (*this < other) {
    return this->PositiveMinus(other, 0);
  }
  BigInteger this_big_integer = *this;
  *this = other;
  *this = this->PositiveMinus(this_big_integer, 0);
  sign = Sign::positive;
  return *this;
}

BigInteger operator-(const BigInteger& first, const BigInteger& second) {
  BigInteger difference = first;
  difference -= second;
  return difference;
}
 
BigInteger BigInteger::PositiveMultiply(int other) {
  BigInteger multiplied = *this;
  unsigned long long to_next = 0;
  for (size_t i = 0; i <= multiplied.length; ++i) {
    to_next += multiplied.digits[i] * other;
    multiplied.digits[i] = to_next % Base;
    to_next /= Base;
  }
  if (multiplied.length + 1 >= multiplied.capacity) {
    multiplied.capacity *= 2;
    multiplied.digits.resize(multiplied.capacity);
  }
  ++multiplied.length;
  multiplied.digits[multiplied.length] = to_next;
  return multiplied;
}

BigInteger& BigInteger::PositiveMultiply(const BigInteger& other) {
  BigInteger product;
  for (size_t i = 0; i <= other.length; ++i) {
    product.PositivePlus(this->PositiveMultiply(other.digits[i]), i);
  }
  product.sign = sign;
  *this = product;
  return *this;
}

BigInteger& BigInteger::operator*=(const BigInteger& other) {
  sign = sign == other.sign ? Sign::positive : Sign::negative;
  *this = this->PositiveMultiply(other);
  return *this;
}

BigInteger operator*(const BigInteger& first, const BigInteger& second) {
  BigInteger product = first;
  product *= second;
  return product;
}

BigInteger BigInteger::PositiveDivide(std::vector<long>& quotient, const BigInteger& other, size_t prev) {
  BigInteger slice;
  slice.digits = {};
  slice.capacity = std::min(other.length + 2, length + 1);
  if (not quotient.empty() and prev == 0) {
    ++slice.capacity;
  }
  slice.length = slice.capacity - 1;
  for (size_t i = length - slice.capacity + 1; i <= length; ++i) {
    slice.digits.push_back(digits[i]);
  }
  size_t left = 0;
  size_t right = Base;
  right *= Base;
  while (left < right - 1) {
    size_t middle = (left + right) / 2;
    if (middle * BigIntegerAbs(other) <= slice) {
      left = middle;
    } else {
      right = middle;
    }
  }
  size_t count = left;
  while (left >= Base) {
    quotient.push_back(left / Base);
    left %= Base;
  }
  quotient.push_back(left);
  BigInteger rest = slice - (count * BigIntegerAbs(other));
  if (slice.capacity - 1 == length) {
    return rest;
  }
  if (rest == 0) {
    for (int i = length - slice.capacity; i >= 0; --i) {
      quotient.push_back(digits[i]);
    }
    return rest;
  }
  *this = this->PositiveMinus(count * BigIntegerAbs(other), length - slice.capacity + 1);
  return PositiveDivide(quotient, other, count);
}

BigInteger& BigInteger::operator/=(const BigInteger& other) {
  if (*this == other) {
    *this = 1;
    return *this;
  }
  sign = sign == other.sign ? Sign::positive : Sign::negative;
  std::vector<long> quotient;
  BigInteger rest = this->PositiveDivide(quotient, other, -1);
  capacity = quotient.size();
  length = capacity - 1;
  digits.resize(capacity);
  for (size_t i = 0; i <= length; ++i) {
    digits[i] = quotient[length - i];
  }
  ChangeLength();
  return *this;
}

BigInteger operator/(const BigInteger& first, const BigInteger& second) {
  BigInteger quotient = first;
  quotient /= second;
  return quotient;
}

BigInteger& BigInteger::PositiveModule(const BigInteger& other) {
  if (*this == other) {
    *this = 0;
    return *this;
  }
  std::vector<long> remainder;
  BigInteger rest = this->PositiveDivide(remainder, other, -1);
  if (sign == Sign::negative) {
    rest.sign = Sign::negative;
  }
  *this = rest;
  ChangeLength();
  return *this;
}

BigInteger& BigInteger::operator%=(const BigInteger& other) {
  *this = this->PositiveModule(other);
  return *this;
}

BigInteger operator%(const BigInteger& first, const BigInteger& second) {
  BigInteger remainder = first;
  remainder %= second;
  return remainder;
}

BigInteger BigInteger::operator-() {
  BigInteger opposite = *this;
  opposite -= 2 * (*this);
  return opposite;
}

std::string BigInteger::toString() const {
  std::string big_integer_string;
  if (sign == Sign::negative and *this != 0) {
    big_integer_string += '-';
  }
  big_integer_string += std::to_string(digits[length]);
  for (int i = length - 1; i >= 0; --i) {
    std::string digit = std::to_string(digits[i]);
    for (size_t j = 0; j < DigitLength - digit.size(); ++j) {
      big_integer_string += '0';
    }
    big_integer_string += digit;
  }
  return big_integer_string;
}

std::ostream& operator<<(std::ostream& out, const BigInteger& big_integer) {
  out << big_integer.toString();
  return out;
}

std::istream& operator>>(std::istream& in, BigInteger& big_integer) {
  std::string number;
  in >> number;
  big_integer = BigInteger(number);
  return in;
}

BigInteger::operator bool() const {
  if (length == 0 and digits[0] == 0) {
    return false;
  }
  return true;
}

Sign BigInteger::BigIntegerSign() const { 
  if (*this == 0) {
    return Sign::positive;
  }
  return sign;
}

BigInteger operator""_bi(const char* number) {
  std::string number_string = number;
  return BigInteger(number_string);
}

class Rational {
 private:
  mutable BigInteger numerator;
  mutable BigInteger denominator;
  static const int Base = 1000000000;
  static const int DigitLength = 9;
  BigInteger GCD(BigInteger numerator, BigInteger denominator) const;
  void Reduce() const;

 public:
  Rational()
      : numerator(BigInteger (0)),
        denominator(BigInteger (1)) {};
  Rational (BigInteger number)
      : numerator(number),
        denominator(1) {};
  Rational (long long number)
      : numerator(BigInteger(number)),
        denominator(1) {};
  ~Rational() = default;

  Rational& operator+=(const Rational& other);
  Rational& operator-=(const Rational& other);
  Rational& operator*=(const Rational& other);
  Rational& operator/=(const Rational& other);
  Rational operator-();
  bool operator==(const Rational& other) const;
  bool operator!=(const Rational& other) const {
    return !(*this == other);
  }
  bool operator<(const Rational& other) const;
  bool operator<=(const Rational& other) const;
  bool operator>(const Rational& other) const {
    return !(*this <= other);
  }
  bool operator>=(const Rational& other) const {
    return !(*this < other);
  }
  std::string toString() const;
  std::string asDecimal(size_t precision = 0) const;
  explicit operator double() const;
};

BigInteger Rational::GCD(BigInteger numerator, BigInteger denominator) const {
  numerator = BigIntegerAbs(numerator);
  while (denominator != 0) {
    numerator %= denominator;
    if (numerator == 0) {
      return denominator;
    }
    denominator %= numerator;
  }
  return numerator;
}

void Rational::Reduce() const {
  BigInteger gcd = GCD(numerator, denominator);
  if (gcd > 1) {
    numerator /= gcd;
    denominator /= gcd;
  }
}

Rational& Rational::operator+=(const Rational& other) {
  numerator = numerator * other.denominator + other.numerator * denominator;
  denominator *= other.denominator;
  return *this;
}

Rational& Rational::operator-=(const Rational& other) {
  numerator = numerator * other.denominator - other.numerator * denominator;
  denominator *= other.denominator;
  return *this;
}

Rational& Rational::operator*=(const Rational& other) {
  numerator *= other.numerator;
  denominator *= other.denominator;
  return *this;
}

Rational& Rational::operator/=(const Rational& other) {
  if (*this == other) {
    *this = 1;
    return *this;
  }
  numerator *= other.denominator;
  denominator *= other.numerator;
  if (denominator < 0) {
    denominator *= -1;
    numerator *= -1;
  }
  return *this;
}

Rational Rational::operator-() {
  Rational opposite = *this;
  opposite.numerator = -numerator;
  return opposite;
}

Rational operator+(const Rational& first, const Rational& second) {
  Rational sum = first;
  sum += second;
  return sum;
}

Rational operator-(const Rational& first, const Rational& second) {
  Rational difference = first;
  difference -= second;
  return difference;
}

Rational operator*(const Rational& first, const Rational& second) {
  Rational product = first;
  product *= second;
  return product;
}

Rational operator/(const Rational& first, const Rational& second) {
  Rational quotient = first;
  quotient /= second;
  return quotient;
}

bool Rational::operator==(const Rational& other) const {
  if (numerator.BigIntegerSign() != other.numerator.BigIntegerSign()) {
    return false;
  }
  this->Reduce();
  other.Reduce();
  if (other.numerator == numerator and other.denominator == denominator) {
    return true;
  }
  return false;
}

bool Rational::operator<(const Rational& other) const {
  return (numerator * other.denominator < other.numerator * denominator);
}

bool Rational::operator<=(const Rational& other) const {
  return (numerator * other.denominator <= other.numerator * denominator);
}

std::string Rational::toString() const {
  std::string rational_string;
  Reduce();
  rational_string += numerator.toString();
  if (denominator != 1) {
    rational_string += '/' + denominator.toString();
  }
  return rational_string;
}

std::string Rational::asDecimal(size_t precision) const {
  Reduce();
  std::string decimal_string;
  if (numerator.BigIntegerSign() == Sign::negative) {
    decimal_string += '-';
  }
  BigInteger integer = BigIntegerAbs(numerator) / denominator;
  if (precision == 0 and denominator != 1) {
    ++integer;
  }
  decimal_string += integer.toString();
  if (precision == 0) {
    return decimal_string;
  }
  decimal_string += '.';
  BigInteger previous;
  BigInteger current;
  BigInteger num = BigIntegerAbs(numerator) % denominator;
  std::string next_add;
  for (size_t i = 0; i < precision / DigitLength + 1; ++i) {
    current = (num * Base) / denominator;
    num = (num * Base) % denominator;
    next_add = current.toString();
    for (size_t i = 0; i < DigitLength - next_add.size(); ++i) {
      decimal_string += '0';
    }
    decimal_string += next_add;
  }
  size_t last_position = integer.toString().size() + precision + (numerator < 0);
  if (decimal_string[last_position + 1] > '4') {
    while(decimal_string[last_position] == '9') {
      decimal_string[last_position] = '0';
      --last_position;
    }
    ++decimal_string[last_position];
  }
  return decimal_string.substr(0, last_position + 1);
}

Rational::operator double() const {
  return std::stod(asDecimal(17));
}