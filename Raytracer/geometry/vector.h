#pragma once

#include <array>
#include <cmath>
#include <cstddef>

class Vector {
public:
  Vector() : data_{0, 0, 0} {}

  Vector(double x, double y, double z) : data_{x, y, z} {}

  double &operator[](size_t ind) { return data_[ind]; }

  double operator[](size_t ind) const { return data_[ind]; }

  double Length() const {
    return std::sqrt(data_[0] * data_[0] + data_[1] * data_[1] +
                     data_[2] * data_[2]);
  }

  double LengthSq() const {
    return data_[0] * data_[0] + data_[1] * data_[1] + data_[2] * data_[2];
  }

  void Normalize() {
    double len = Length();
    if (len != 0) {
      data_[0] /= len;
      data_[1] /= len;
      data_[2] /= len;
    }
  }

  Vector Normalized() const {
    double len = Length();
    if (len != 0) {
      return Vector(data_[0] / len, data_[1] / len, data_[2] / len);
    }
    return *this;
  }

  Vector &operator+=(const Vector &other) {
    data_[0] += other[0];
    data_[1] += other[1];
    data_[2] += other[2];
    return *this;
  }

  Vector &operator-=(const Vector &other) {
    data_[0] -= other[0];
    data_[1] -= other[1];
    data_[2] -= other[2];
    return *this;
  }

  Vector &operator*=(double d) {
    data_[0] *= d;
    data_[1] *= d;
    data_[2] *= d;
    return *this;
  }

  Vector &operator/=(double d) {
    if (d != 0) {
      data_[0] /= d;
      data_[1] /= d;
      data_[2] /= d;
    }
    return *this;
  }

  Vector operator-() const { return Vector(-data_[0], -data_[1], -data_[2]); }

private:
  std::array<double, 3> data_;
};

// Внешние операторы и функции
double DotProduct(const Vector &a, const Vector &b) {
  return a[0] * b[0] + a[1] * b[1] + a[2] * b[2];
}

Vector CrossProduct(const Vector &a, const Vector &b) {
  return Vector(a[1] * b[2] - a[2] * b[1], a[2] * b[0] - a[0] * b[2],
                a[0] * b[1] - a[1] * b[0]);
}

double Length(const Vector &v) { return v.Length(); }

Vector operator+(const Vector &lhs, const Vector &rhs) {
  return Vector(lhs[0] + rhs[0], lhs[1] + rhs[1], lhs[2] + rhs[2]);
}

Vector operator-(const Vector &lhs, const Vector &rhs) {
  return Vector(lhs[0] - rhs[0], lhs[1] - rhs[1], lhs[2] - rhs[2]);
}

Vector operator*(const Vector &lhs, double rhs) {
  return Vector(lhs[0] * rhs, lhs[1] * rhs, lhs[2] * rhs);
}

Vector operator*(double lhs, const Vector &rhs) {
  return Vector(lhs * rhs[0], lhs * rhs[1], lhs * rhs[2]);
}

Vector operator/(const Vector &lhs, double rhs) {
  if (rhs != 0) {
    return Vector(lhs[0] / rhs, lhs[1] / rhs, lhs[2] / rhs);
  }
  return lhs;
}

Vector operator+(const Vector &lhs, double rhs) {
  return Vector(lhs[0] + rhs, lhs[1] + rhs, lhs[2] + rhs);
}

Vector operator+(double lhs, const Vector &rhs) {
  return Vector(lhs + rhs[0], lhs + rhs[1], lhs + rhs[2]);
}

bool operator==(const Vector &lhs, const Vector &rhs) {
  const double epsilon = std::numeric_limits<double>::epsilon();
  return std::abs(lhs[0] - rhs[0]) < epsilon &&
         std::abs(lhs[1] - rhs[1]) < epsilon &&
         std::abs(lhs[2] - rhs[2]) < epsilon;
}

bool operator!=(const Vector &lhs, const Vector &rhs) { return !(lhs == rhs); }

Vector operator*(const Vector &lhs, const Vector &rhs) {
  return Vector(lhs[0] * rhs[0], lhs[1] * rhs[1], lhs[2] * rhs[2]);
}

Vector operator/(const Vector &lhs, const Vector &rhs) {
  return Vector(lhs[0] / rhs[0], lhs[1] / rhs[1], lhs[2] / rhs[2]);
}
