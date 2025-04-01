#include <algorithm>
#include <cmath>
#include <iostream>
#include <vector>

const long double cAccuracy = 1e-8;
const long long cVeryBig = 1e9;

bool LowerEqual(long double first, long double second) {
  return first < second || std::abs(first - second) < cAccuracy;
}

bool Equal(long double first, long double second) {
  return std::abs(first - second) < cAccuracy;
}

struct Vector {
  long long x;
  long long y;

  Vector(long long x, long long y) : x(x), y(y) {}
  long double Length() const { return std::sqrt(x * x + y * y); }
};

Vector operator+(const Vector& first, const Vector& second) {
  return {first.x + second.x, first.y + second.y};
}

Vector operator-(const Vector& first, const Vector& second) {
  return {first.x - second.x, first.y - second.y};
}

long long ScalarProduct(const Vector& first, const Vector& second) {
  return first.x * second.x + first.y * second.y;
}

long long VectorProduct(const Vector& first, const Vector& second) {
  return first.x * second.y - second.x * first.y;
}

long double Distance(const Vector& first, const Vector& second) {
  Vector third = first - second;
  return third.Length();
}

long long Sign(long long number) { return (number >= 0 ? 1 : -1); }

struct Comparator {
  Vector control_point;

  bool operator()(const Vector& first, const Vector& second) const {
    if (VectorProduct(first - control_point, second - control_point) > 0) {
      return true;
    }
    if (VectorProduct(first - control_point, second - control_point) < 0) {
      return false;
    }
    return Distance(first, control_point) < Distance(second, control_point);
  }
};

std::vector<Vector> ConvexHull(std::vector<Vector>& points, Vector min_point) {
  std::sort(points.begin(), points.end(), Comparator{min_point});
  std::vector<Vector> convex_hull = {points[0], points[1]};
  size_t size = 2;
  for (size_t i = 2; i < points.size(); ++i) {
    while (
        size >= 2 &&
        LowerEqual(VectorProduct(convex_hull[size - 1] - convex_hull[size - 2],
                                 points[i] - convex_hull[size - 1]),
                   0)) {
      convex_hull.pop_back();
      --size;
    }
    convex_hull.push_back(points[i]);
    ++size;
  }
  return convex_hull;
}

Vector FindMinPoint(const std::vector<Vector>& points) {
  Vector min_point = {cVeryBig, cVeryBig};
  for (size_t i = 0; i < points.size(); ++i) {
    if (points[i].x < min_point.x ||
        (points[i].x == min_point.x && points[i].y < min_point.y)) {
      min_point = points[i];
    }
  }
  return min_point;
}

std::vector<Vector> MinkowskiSum(const std::vector<Vector>& first,
                                 const std::vector<Vector>& second) {
  std::vector<Vector> sum;
  sum.push_back({first[0].x + second[0].x, first[0].y + second[0].y});
  size_t first_pointer = 0;
  size_t second_pointer = 0;
  while (first_pointer < first.size() || second_pointer < second.size()) {
    if (first_pointer >= first.size()) {
      sum.push_back(sum[sum.size() - 1] +
                    second[(second_pointer + 1) % second.size()] -
                    second[second_pointer]);
      ++second_pointer;
      continue;
    }
    if (second_pointer >= second.size()) {
      sum.push_back(sum[sum.size() - 1] +
                    first[(first_pointer + 1) % first.size()] -
                    first[first_pointer]);
      ++first_pointer;
      continue;
    }
    long long angle = VectorProduct(
        second[(second_pointer + 1) % second.size()] - second[second_pointer],
        first[(first_pointer + 1) % first.size()] - first[first_pointer]);
    if (angle > 0) {
      sum.push_back(sum[sum.size() - 1] +
                    second[(second_pointer + 1) % second.size()] -
                    second[second_pointer]);
      ++second_pointer;
    } else if (angle == 0) {
      sum.push_back(
          sum[sum.size() - 1] + second[(second_pointer + 1) % second.size()] -
          second[second_pointer] + first[(first_pointer + 1) % first.size()] -
          first[first_pointer]);
      ++first_pointer;
      ++second_pointer;
    } else {
      sum.push_back(sum[sum.size() - 1] +
                    first[(first_pointer + 1) % first.size()] -
                    first[first_pointer]);
      ++first_pointer;
    }
  }
  sum.pop_back();
  return sum;
}

bool IsPointInCorner(const Vector& first, const Vector& second,
                     const Vector& point) {
  return (
      Sign(VectorProduct(first, point)) == Sign(VectorProduct(first, second)) &&
      Sign(VectorProduct(first, second)) == Sign(VectorProduct(point, second)));
}

std::pair<size_t, size_t> FindCorner(const std::vector<Vector>& polygon,
                                     const Vector& point) {
  size_t right = 1;
  size_t left = polygon.size() - 1;
  while (left > right + 1) {
    size_t middle = (right + left) / 2;
    if (IsPointInCorner(polygon[middle] - polygon[0],
                        polygon[left] - polygon[0], point)) {
      right = middle;
    } else {
      left = middle;
    }
  }
  return {right, left};
}

std::vector<Vector> FindSum(std::vector<std::vector<Vector>>& points) {
  std::vector<std::vector<Vector>> polygons(points.size());
  for (size_t i = 0; i < points.size(); ++i) {
    polygons[i] = ConvexHull(points[i], FindMinPoint(points[i]));
  }
  std::vector<Vector> sum = polygons[0];
  for (size_t i = 1; i < polygons.size(); ++i) {
    sum = MinkowskiSum(sum, polygons[i]);
  }
  return sum;
}

bool IsCenterMass(const std::vector<Vector>& sum, const Vector& request) {
  if (!IsPointInCorner(sum[1] - sum[0], sum[sum.size() - 1] - sum[0],
                       request - sum[0])) {
    return false;
  }
  auto corner = FindCorner(sum, request - sum[0]);
  return IsPointInCorner(sum[corner.second] - sum[corner.first],
                         sum[0] - sum[corner.first],
                         request - sum[corner.first]) &&
         IsPointInCorner(sum[0] - sum[corner.second],
                         sum[corner.first] - sum[corner.second],
                         request - sum[corner.second]);
}

int main() {
  size_t number;
  long long x;
  long long y;
  std::vector<std::vector<Vector>> points(3);
  for (size_t i = 0; i < 3; ++i) {
    std::cin >> number;
    for (size_t j = 0; j < number; ++j) {
      std::cin >> x >> y;
      points[i].push_back({x, y});
    }
  }

  std::vector<Vector> sum = FindSum(points);

  size_t requests_number;
  std::cin >> requests_number;
  for (size_t i = 0; i < requests_number; ++i) {
    std::cin >> x >> y;
    Vector request(x * 3, y * 3);
    if (IsCenterMass(sum, request)) {
      std::cout << "YES" << '\n';
    } else {
      std::cout << "NO" << '\n';
    }
  }
}