#pragma once

#include "vector.h"

#include <cstddef>

class Triangle {
public:
  Triangle() = default;

  Triangle(const Vector &a, const Vector &b, const Vector &c)
      : a_(a), b_(b), c_(c) {}

  const Vector &operator[](size_t ind) const {
    switch (ind) {
    case 0:
      return a_;
    case 1:
      return b_;
    default:
      return c_;
    }
  }

  double Area() const {
    Vector ab = b_ - a_;
    Vector ac = c_ - a_;

    Vector cross = CrossProduct(ab, ac);
    return 0.5 * Length(cross);
  }

private:
  Vector a_;
  Vector b_;
  Vector c_;
};
