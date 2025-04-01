#include <algorithm>
#include <iostream>
#include <cmath>
#include <vector>

namespace detail {
  bool equal(double first, double second) {
    return std::fabs(first - second) < 0.00001;
  }

  bool vector_equal(std::vector<double>& first, std::vector<double>& second) {
    if (first.size() != second.size()) {
      return false;
    }
    for (size_t i = 0; i < first.size(); ++i) {
      if (not equal(first[i], second[i])) {
        return false;
      }
    }
    return true;
  }
}

struct Point {
  double x;
  double y;
  Point() : x(0), y(0) {};
  Point(double x, double y) : x(x), y(y) {}
  Point(const Point& other) : x(other.x), y(other.y) {}
  ~Point() = default;
  bool operator==(const Point& other) const {
    return detail::equal(x, other.x) and detail::equal(y, other.y);
  }
  bool operator!=(const Point& other) const {
    return not (*this == other);
  }
};

namespace detail {
  double my_distance(const Point& first, const Point& second) {
    return std::pow(std::pow(first.x - second.x, 2.0) +
                    std::pow(first.y - second.y, 2.0), 0.5);
  }

  Point middle_point(const Point& first, const Point& second) {
    return Point((first.x + second.x) / 2.0, (first.y + second.y) / 2.0);
  }

  struct Vector {
    double x;
    double y;
    Vector() : x(0), y(0) {};
    Vector(double x, double y) : x(x), y(y) {};
    Vector(const Point& first, const Point& second) : x(second.x - first.x),
                                                      y(second.y - first.y) {};
    ~Vector() = default;
    double length() const {
      return std::pow(x * x + y * y, 0.5);
    }
    double angle(const Vector& other) const;
    Vector& operator*=(double cofficient);
  };

  Vector& Vector::operator*=(double cofficient) {
    x *= cofficient;
    y *= cofficient;
    return *this;
  }

  Vector operator+(const Vector& first, const Vector& second) {
    return Vector(first.x + second.x, first.y + second.y);
  }

  Vector operator-(const Vector& first, const Vector& second) {
    return Vector(first.x - second.x, first.y - second.y);
  }

  Vector operator*(const Vector& first, double coefficient) {
    return Vector(first.x * coefficient, first.y * coefficient);
  }

  double Vector::angle(const Vector& other) const {
    Vector inner_side = *this + other;
    if (x * inner_side.y - y * inner_side.x > 0) {
      return 2 * M_PI - std::acos(-(x * other.x + y * other.y) /
                        (length() * other.length()));
    }
    return std::acos(-(x * other.x + y * other.y) /
           (length() * other.length()));
  }
}

using namespace detail;

class Line {
 private:
  double slope() const;
  double shift() const;
 public:
  Point l_point;
  Point r_point;
  Line(const Point& a, const Point& b);
  Line(double slope, double shift);
  Line(const Point& point, double slope);
  ~Line() = default;

  bool operator==(const Line& other) const;
  bool operator!=(const Line& other) const {
    return not (*this == other);
  }
};

double Line::slope() const {
  return (r_point.y - l_point.y) / (r_point.x - l_point.x);
}

double Line::shift() const {
  return l_point.y - slope() * l_point.x;
}

Line::Line(const Point& a, const Point& b) : l_point(a), r_point(b) {
  if (a.x > b.x or (equal(a.x, b.x) and a.y > b.y)) {
    std::swap(l_point, r_point);
  }
}

Line::Line(double slope, double shift) : l_point(0, shift) {
  if (equal(slope, 0)) {
    r_point = Point(1, shift);
  } else if (equal(shift, 0)) {
    r_point = Point((1 - shift) / slope, 1);
  } else {
    r_point = Point(shift / slope, 0);
  }
  if (l_point.x > r_point.x or (equal(l_point.x, r_point.x) and
                                l_point.y > r_point.y)) {
    std::swap(l_point, r_point);
  }
}

Line::Line(const Point& point, double slope) {
  *this = Line(slope, point.y - slope * point.x);
}

bool Line::operator==(const Line& other) const {
  if (equal(l_point.x, r_point.x)) {
    return equal(other.l_point.x, other.r_point.x) and
           equal(other.l_point.x, l_point.x);
  }
  return equal(slope(), other.slope()) and equal(shift(), other.shift());
}

class Shape {
 public:
  Shape() = default;
  virtual ~Shape() = default;
  virtual double perimeter() const = 0;
  virtual double area() const = 0;
  virtual bool operator==(const Shape& other) const = 0;
  virtual bool operator!=(const Shape& other) const = 0;
  virtual bool isCongruentTo(const Shape& other) const = 0;
  virtual bool isSimilarTo(const Shape& other) const = 0;
  virtual bool containsPoint(const Point& point) const = 0;
  virtual void rotate(const Point& center, double angle) = 0;
  virtual void reflect(const Point& center) = 0;
  virtual void reflect(const Line& axis) = 0;
  virtual void scale(const Point& center, double coefficient) = 0;
};

class Ellipse: public Shape {
 protected:
  Point l_focus;
  Point r_focus;
  double distances;
  double major_semi_axes() const {
    return distances / 2.0;
  }
  double minor_semi_axes() const;
 public:
  Ellipse(const Ellipse& other) : l_focus(other.l_focus), r_focus(other.r_focus),
                                  distances(other.distances) {};
  Ellipse(const Point& first_focus, const Point& second_focus, double distances);
  ~Ellipse() = default;

  std::pair<Point,Point> focuses() const {
    return std::pair(l_focus, r_focus);
  }
  std::pair<Line, Line> directrices() const;
  double eccentricity() const {
    return std::pow(std::pow(major_semi_axes(), 2.0) -
                    std::pow(minor_semi_axes(), 2.0), 0.5) / major_semi_axes();
  }
  Point center() const {
    return middle_point(l_focus, r_focus);
  }

  double perimeter() const override {
    return 4 * major_semi_axes() * std::comp_ellint_2(eccentricity());
  }
  double area() const override {
    return M_PI * major_semi_axes() * minor_semi_axes();
  }
  bool operator==(const Shape& other) const override;
  bool operator!=(const Shape& other) const override {
    return not(*this == other);
  }
  bool isCongruentTo(const Shape& other) const override;
  bool isSimilarTo(const Shape& other) const override;
  bool containsPoint(const Point& point) const override {
    return (my_distance(l_focus, point) + my_distance(r_focus, point)) <= distances;
  }
  void rotate(const Point& center, double angle) override;
  void reflect(const Point& center) override;
  void reflect(const Line& axis) override;
  void scale(const Point& center, double coefficient) override;
};

Ellipse::Ellipse(const Point& first_focus, const Point& second_focus, double distances)
    : l_focus(first_focus), r_focus(second_focus), distances(distances) {
  if (l_focus.x > r_focus.x) {
    std::swap(l_focus, r_focus);
  }
  if (equal(l_focus.x, r_focus.x) and l_focus.y > r_focus.y) {
    std::swap(l_focus, r_focus);
  }
}

double Ellipse::minor_semi_axes() const {
  double first_turm = std::pow(distances / 2.0, 2.0);
  double second_turm = std::pow(my_distance(l_focus, r_focus) / 2.0 , 2.0);
  return std::pow(first_turm - second_turm, 0.5);
}

std::pair<Line, Line> Ellipse::directrices() const {
  Vector major_axes(l_focus, r_focus);
  major_axes = major_axes * ((major_semi_axes() / eccentricity()) / major_axes.length());
  Point ellipse_center = center();
  Point r_a(ellipse_center.x + major_axes.x, ellipse_center.y + major_axes.y);
  Vector normal(major_axes.y, major_axes.x);
  Point r_b(r_a.x + normal.x, r_a.y + normal.y);
  Line r_line(r_a, r_b);

  major_axes.x = -major_axes.x;
  major_axes.y = -major_axes.y;
  Point l_a(ellipse_center.x + major_axes.x, ellipse_center.y + major_axes.y);
  Point l_b(l_a.x + normal.x, l_a.y + normal.y);
  Line l_line(l_a, l_b);
  return std::pair(l_line, r_line);
}

bool Ellipse::operator==(const Shape& other) const {
  const Ellipse* another = dynamic_cast<const Ellipse*>(&other);
  if (another == nullptr) {
    return false;
  }
  return (l_focus == another->l_focus) and (r_focus == another->r_focus)
         and equal(distances, another->distances); 
}

bool Ellipse::isCongruentTo(const Shape& other) const {
  const Ellipse* another = dynamic_cast<const Ellipse*>(&other);
  if (another == nullptr) {
    return false;
  }
  return equal(my_distance(l_focus, r_focus),
               my_distance(another->l_focus, another->r_focus)) and
         equal(distances, another->distances);
}

bool Ellipse::isSimilarTo(const Shape& other) const {
  const Ellipse* another = dynamic_cast<const Ellipse*>(&other);
  if (another == nullptr) {
    return false;
  }
  double koeff = minor_semi_axes() / another->minor_semi_axes();
  return equal(koeff, major_semi_axes() / another->major_semi_axes());
}

void Ellipse::rotate(const Point& center, double angle) {
  angle = (angle * M_PI) / 180.0;
  double x = l_focus.x - center.x;
  l_focus.x = (l_focus.x - center.x) * std::cos(angle) -
              (l_focus.y - center.y) * std::sin(angle);
  l_focus.y = x * std::sin(angle) + (l_focus.y - center.y) * std::cos(angle);
  x = r_focus.x - center.x;
  r_focus.x = (r_focus.x - center.x) * std::cos(angle) -
              (r_focus.y - center.y) * std::sin(angle);
  r_focus.y = x * std::sin(angle) + (r_focus.y - center.y) * std::cos(angle);
}

void Ellipse::reflect(const Point& center) {
  l_focus.x = l_focus.x + 2 * (center.x - l_focus.x);
  l_focus.y = l_focus.y + 2 * (center.y - l_focus.y);
  r_focus.x = r_focus.x + 2 * (center.x - r_focus.x);
  r_focus.y = r_focus.y + 2 * (center.y - r_focus.y);
}

void Ellipse::reflect(const Line& axis) {
  Vector guide(axis.r_point, axis.l_point);

  Vector not_normal(l_focus, axis.l_point);
  double angle = not_normal.angle(guide);
  Vector normal = not_normal + guide * (std::cos(angle) / guide.length());
  l_focus.x += 2 * normal.length();
  l_focus.y += 2 * normal.length();

  not_normal = Vector(r_focus, axis.l_point);
  angle = not_normal.angle(guide);
  normal = not_normal + guide * (std::cos(angle) / guide.length());
  r_focus.x += 2 * normal.length();
  r_focus.y += 2 * normal.length();
}

void Ellipse::scale(const Point& center, double coefficient) {
  l_focus.x = center.x + coefficient * (l_focus.x - center.x);
  l_focus.y = center.y + coefficient * (l_focus.y - center.y);
  r_focus.x = center.x + coefficient * (r_focus.x - center.x);
  r_focus.y = center.y + coefficient * (r_focus.y - center.y);
  distances *= coefficient;
}

class Circle : public Ellipse {
 public:
  Circle(const Point& center, double radius) : Ellipse(center, center, 2 * radius) {};
  Circle(const Circle& other) : Ellipse(other.l_focus, other.r_focus, other.distances) {};
  ~Circle() = default;
  
  double radius() const {
    return distances / 2.0;
  }
};

class Polygon: public Shape {
 protected:
  std::vector<Point> vertices;
  std::vector<double> sides() const;
  std::vector<double> angles() const;
 public:
  Polygon() : vertices(std::vector<Point>(0)) {};
  Polygon(Polygon& other) : vertices(other.vertices) {};
  Polygon(std::vector<Point> vertices) : vertices(vertices) {};
  template<typename... T>
  Polygon(T&&... args);
  ~Polygon() = default;

  Polygon& operator=(const Polygon& other);
  size_t verticesCount() const {
    return vertices.size();
  }
  const std::vector<Point> getVertices() const {
    return vertices;
  }
  bool isConvex() const;

  double perimeter() const override;
  double area() const override;
  bool operator==(const Shape& other) const override;
  bool operator!=(const Shape& other) const override {
    return not(*this == other);
  }
  bool isCongruentTo(const Shape& other) const override;
  bool isSimilarTo(const Shape& other) const override;
  bool containsPoint(const Point& point) const override;
  void rotate(const Point& center, double angle) override;
  void reflect(const Point& center) override;
  void reflect(const Line& axis) override;
  void scale(const Point& center, double coefficient) override;
};

std::vector<double> Polygon::sides() const {
  std::vector<double> sides(vertices.size());
  for (size_t i = 0; i + 1 < vertices.size(); ++i) {
    sides[i] = std::pow(std::pow(vertices[i].x - vertices[i + 1].x, 2.0) +
                        std::pow(vertices[i].y - vertices[i + 1].y, 2.0), 0.5);
  }
  sides[vertices.size() - 1] = std::pow(std::pow(vertices[vertices.size() - 1].x -
                                                 vertices[0].x, 2.0) +
                                        std::pow(vertices[vertices.size() - 1].y -
                                                 vertices[0].y, 2.0), 0.5);
  return sides;
}

std::vector<double> Polygon::angles() const {
  std::vector<double> angles(vertices.size());
  Vector first_side;
  Vector second_side;
  for (size_t i = 0; i < vertices.size(); ++i) {
    first_side = Vector(vertices[i], vertices[(i + 1) % vertices.size()]);
    second_side = Vector(vertices[(i + 1) % vertices.size()],
                         vertices[(i + 2) % vertices.size()]);
    angles[i] = std::fmin(first_side.angle(second_side),
                          2 * M_PI - first_side.angle(second_side));
  }
  return angles;
}

template<typename... T>
Polygon::Polygon(T&&... args) {
  (vertices.push_back(std::forward<T>(args)), ...);
}

Polygon& Polygon::operator=(const Polygon& other) {
  vertices = other.vertices;
  return *this;
}

bool Polygon::isConvex() const {
  if (vertices.size() <= 3) {
    return true;
  }
  Vector first_side(vertices[0], vertices[1]);
  Vector second_side(vertices[1], vertices[2]);
  int rotate = first_side.x * second_side.y - first_side.y * second_side.x;
  for (size_t i = 1; i < vertices.size(); ++i) {
    first_side = Vector(vertices[i], vertices[(i + 1) % vertices.size()]);
    second_side = Vector(vertices[(i + 1) % vertices.size()],
                         vertices[(i + 2) % vertices.size()]);
    int now_rotate = first_side.x * second_side.y - first_side.y * second_side.x;
    if ((rotate < 0 and now_rotate > 0) or (rotate > 0 and now_rotate < 0)) {
      return false;
    }
  }
  return true;
}

double Polygon::perimeter() const {
  double perimeter = 0;
  std::vector<double> side = sides();
  for(size_t i = 0; i < vertices.size(); ++i) {
    perimeter += side[i];
  }
  return perimeter;
}

double Polygon::area() const {
  double area = 0;
  for (size_t i = 0; i + 1 < vertices.size(); ++i) {
    area += vertices[i].x * vertices[i + 1].y;
    area -= vertices[i + 1].x * vertices[i].y;
  }
  area += vertices.back().x * vertices[0].y;
  area -= vertices.back().y * vertices[0].x;
  return 0.5 * std::fabs(area);
}

bool Polygon::operator==(const Shape& other) const {
  const Polygon* another = dynamic_cast<const Polygon*>(&other);
  if (another == nullptr) {
    return false;
  }
  if (vertices.size() != another->vertices.size()) {
    return false;
  }

  std::vector<Point> copy = vertices;
  for (size_t i = 0; i < vertices.size(); ++i) {
    if (copy == another->vertices) {
      return true;
    }
    std::reverse(copy.begin(), copy.end());
    if (copy == another->vertices) {
      return true;
    }
    std::reverse(copy.begin(), copy.end());
    std::rotate(copy.begin(), copy.begin() + 1, copy.end());
  }
  return false;
}

bool Polygon::isCongruentTo(const Shape& other) const {
  const Polygon* another = dynamic_cast<const Polygon*>(&other);
  if (another == nullptr) {
    return false;
  }
  if (vertices.size() != another->vertices.size()) {
    return false;
  }

  std::vector<double> this_sides = sides();
  std::vector<double> this_angles = angles();
  std::rotate(this_angles.begin(), this_angles.end() - 1,
              this_angles.end());
  std::vector<double> other_sides = another->sides();
  std::vector<double> other_angles = another->angles();
  for (size_t i = 0; i < vertices.size(); ++i) {
    if (vector_equal(this_sides, other_sides) and
        vector_equal(this_angles, other_angles)) {
      return true;
    }
    std::reverse(this_angles.begin(), this_angles.end());
    std::reverse(this_sides.begin(), this_sides.end());
    if (vector_equal(this_sides, other_sides) and
        vector_equal(this_angles, other_angles)) {
      return true;
    }

    std::reverse(this_angles.begin(), this_angles.end());
    std::reverse(this_sides.begin(), this_sides.end());
    std::rotate(this_sides.begin(), this_sides.begin() + 1,
                this_sides.end());
    std::rotate(this_angles.begin(), this_angles.begin() + 1,
                this_angles.end());
  }
  return false;
}

bool Polygon::isSimilarTo(const Shape& other) const {
  const Polygon* another = dynamic_cast<const Polygon*>(&other);
  if (another == nullptr) {
    return false;
  }
  if (vertices.size() != another->vertices.size()) {
    return false;
  }

  std::vector<double> this_sides = sides();
  std::vector<double> this_angles = angles();
  std::rotate(this_angles.begin(), this_angles.end() - 1,
              this_angles.end());
  std::vector<double> other_sides = another->sides();
  std::vector<double> other_angles = another->angles();
  for (size_t i = 0; i < vertices.size(); ++i) {
    if (vector_equal(this_angles, other_angles)) {
      double koeff = this_sides[0] / other_sides[0];
      for (size_t i = 1; i < vertices.size(); ++i) {
        if (not equal(koeff, this_sides[i] / other_sides[i])) {
          break;
        }
      }
      return true;
    }
    std::reverse(this_angles.begin(), this_angles.end());
    if (vector_equal(this_angles, other_angles)) {
      std::reverse(this_sides.begin(), this_sides.end());
      double koeff = this_sides[0] / other_sides[0];
      for (size_t i = 1; i < vertices.size(); ++i) {
        if (not equal(koeff, this_sides[i] / other_sides[i])) {
          std::reverse(this_sides.begin(), this_sides.end());
          break;
        }
      }
      return true;
    }
    std::reverse(this_angles.begin(), this_angles.end());
    std::rotate(this_sides.begin(), this_sides.begin() + 1,
                this_sides.end());
    std::rotate(this_angles.begin(), this_angles.begin() + 1,
                this_angles.end());
  }
  return false;
}

bool Polygon::containsPoint(const Point& point) const {
  double sum_angle = 0;
  Vector first_side;
  Vector second_side;
  for (size_t i = 0; i < vertices.size(); ++i) {
    if (point == vertices[i] or point == vertices[(i + 1) % vertices.size()]) {
      return true;
    }
    first_side = Vector(point, vertices[i]);
    second_side = Vector(point, vertices[(i + 1) % vertices.size()]);
    sum_angle += std::fabs(M_PI - first_side.angle(second_side));
  }
  return equal(sum_angle, 2 * M_PI);
}

void Polygon::rotate(const Point& center, double angle) {
  angle = (angle * M_PI) / 180.0;
  for (size_t i = 0; i < vertices.size(); ++i) {
    double x = vertices[i].x - center.x;
    vertices[i].x = x * std::cos(angle) -
                    (vertices[i].y - center.y) * std::sin(angle);
    vertices[i].x += center.x;
    vertices[i].y = x * std::sin(angle) +
                    (vertices[i].y - center.y) * std::cos(angle);
    vertices[i].y += center.y;
  }
}

void Polygon::reflect(const Point& center) {
  for (size_t i = 0; i < vertices.size(); ++i) {
    vertices[i].x = vertices[i].x + 2 * (center.x - vertices[i].x);
    vertices[i].y = vertices[i].y + 2 * (center.y - vertices[i].y);
  }
}

void Polygon::reflect(const Line& axis) {
  Vector not_normal;
  Vector guide(axis.r_point, axis.l_point);
  Vector normal;
  for (size_t i = 0; i < vertices.size(); ++i) {
    not_normal = Vector(vertices[i], axis.l_point);
    double angle = not_normal.angle(guide);
    normal = not_normal + guide * (std::cos(angle) * not_normal.length() /
                                                     guide.length());
    vertices[i].x += 2 * normal.x;
    vertices[i].y += 2 * normal.y;
  }
}

void Polygon::scale(const Point& center, double coefficient) {
  for (size_t i = 0; i < vertices.size(); ++i) {
    vertices[i].x = center.x + coefficient * (vertices[i].x - center.x);
    vertices[i].y = center.y + coefficient * (vertices[i].y - center.y);
  }
}

class Rectangle: public Polygon {
  public:
    Rectangle(const Point& a, const Point& c, double coefficient);
    Rectangle(const Rectangle& other) : Polygon(other.vertices) {};
    ~Rectangle() = default;
    std::pair<Line, Line> diagonals() const {
      return std::pair(Line(vertices[0], vertices[2]),
                       Line(vertices[1], vertices[3]));
    }
    Point center() const {
      return middle_point(vertices[0], vertices[2]);
    }
};

Rectangle::Rectangle(const Point& a, const Point& c, double coefficient) {
  double short_side_legth = my_distance(a, c) /
                            std::pow(coefficient * coefficient + 1.0, 0.5);
  Point copy_c(c);
  double angle = std::atan(coefficient);
  double x = copy_c.x - a.x;
  copy_c.x = x * std::cos(angle) - (copy_c.y - a.y) * std::sin(angle) + a.x;
  copy_c.y = x * std::sin(angle) + (copy_c.y - a.y) * std::cos(angle) + a.y;
  Vector short_side(a, copy_c);
  short_side *= short_side_legth / short_side.length();

  copy_c = c;
  angle = 2 * M_PI - std::atan(1 / coefficient);
  copy_c.x = x * std::cos(angle) - (copy_c.y - a.y) * std::sin(angle) + a.x;
  copy_c.y = x * std::sin(angle) + (copy_c.y - a.y) * std::cos(angle) + a.y;
  Vector long_side(a, copy_c);
  long_side *= coefficient * short_side_legth / long_side.length();

  vertices.emplace_back(a);
  vertices.emplace_back(Point(a.x + short_side.x, a.y + short_side.y));
  vertices.emplace_back(c);
  vertices.emplace_back(Point (a.x + long_side.x, a.y + long_side.y));
}

class Square: public Rectangle {
 public:
  Square(const Point& a, const Point& c) : Rectangle(a, c, 1) {};
  Square(const Square& other) : Rectangle(other.vertices[0], other.vertices[2], 1) {};
  ~Square() = default;

  Circle circumscribedCircle() const {
    return Circle(middle_point(vertices[0], vertices[2]),
                  my_distance(vertices[0], vertices[2]) / 2.0);
  }
  Circle inscribedCircle() const {
    return Circle(middle_point(vertices[0], vertices[2]),
                  my_distance(vertices[0], vertices[1]) / 2.0);
  }
};

class Triangle: public Polygon {
 public:
  Triangle(Triangle& other) : Polygon(other.vertices) {};
  Triangle(std::vector<Point> vertices) : Polygon(vertices) {};
  template<typename... T>
  Triangle(T&&... args) : Polygon(args...) {};
  ~Triangle() = default;

  Circle circumscribedCircle() const;
  Circle inscribedCircle() const;
  Point centroid() const {
    return Point((vertices[0].x + vertices[1].x + vertices[2].x) / 3.0,
                 (vertices[0].y + vertices[1].y + vertices[2].y) / 3.0);
  }
  Point orthocenter() const;
  Line EulerLine() const {
    return Line(orthocenter(), circumscribedCircle().center());
  }
  Circle ninePointsCircle() const;
};

Circle Triangle::circumscribedCircle() const {
  Vector second_side(vertices[1], vertices[2]);
  Vector third_side(vertices[2], vertices[0]);
  Vector normal;
  double radius = my_distance(vertices[0],
                              vertices[1]) /
                              (2.0 * std::fabs(std::sin(second_side.angle(third_side))));
  if (second_side.angle(third_side) > M_PI) {
    normal = Vector(-second_side.y, second_side.x);
  } else {
    normal = Vector(second_side.y, -second_side.x);
  }
  if (equal(radius, second_side.length() / 2.0)) {
    normal.x = 0;
    normal.y = 0;
  } else {
    normal *= std::pow(radius * radius - std::pow(second_side.length() / 2.0, 2.0), 0.5) /
                                         normal.length();
  }
  Point middle = middle_point(vertices[1], vertices[2]);
  Point center(middle.x + normal.x, middle.y + normal.y);
  return Circle(center, radius);
}

Circle Triangle::inscribedCircle() const {
  double radius = 2.0 * area() / perimeter();
  Vector second_side(vertices[1], vertices[2]);
  Vector third_side(vertices[2], vertices[0]);
  Vector normal;
  if (second_side.angle(third_side) > M_PI) {
    second_side *= radius / std::tan(M_PI - second_side.angle(third_side) / 2.0) /
                             second_side.length();
    normal = Vector(-second_side.y, second_side.x);
  } else {
    second_side *= radius / std::tan(second_side.angle(third_side) / 2.0) /
                             second_side.length();
    normal = Vector(second_side.y, -second_side.x);
  }
  normal *= radius / normal.length();
  Point center(vertices[2].x - second_side.x + normal.x,
               vertices[2].y - second_side.y + normal.y);
  return Circle(center, radius);
}

Point Triangle::orthocenter() const {
  Vector second_side(vertices[1], vertices[2]);
  Vector third_side(vertices[2], vertices[0]);
  Point center;
  if (equal(second_side.x, 0)) {
    center.y = (vertices[0].x * second_side.x + vertices[0].y * second_side.y) /
                second_side.y;
    center.x = (vertices[1].x * third_side.x + vertices[1].y * third_side.y -
                center.y * third_side.y) / third_side.x;
  } else if (equal(second_side.y, 0)) {
    center.x = (vertices[0].x * second_side.x + vertices[0].y * second_side.y) /
                second_side.x;
    center.y = (vertices[1].x * third_side.x + vertices[1].y * third_side.y -
                center.x * third_side.x) / third_side.y;
  } else if (equal(third_side.x, 0)) {
    center.y = (vertices[1].x * third_side.x + vertices[1].y * third_side.y) /
                third_side.y;
    center.x = (vertices[0].x * second_side.x + vertices[0].y * second_side.y -
                center.y * second_side.y) / second_side.x;
  } else if (equal(third_side.y, 0)) {
    center.x = (vertices[1].x * third_side.x + vertices[1].y * third_side.y) /
                third_side.x;
    center.y = (vertices[0].x * second_side.x + vertices[0].y * second_side.y -
                center.x * second_side.x) / second_side.y;
  } else {
    center.y = (vertices[1].x * third_side.x + vertices[1].y * third_side.y -
                vertices[0].x * third_side.x - vertices[0].y * second_side.y *
                third_side.x / second_side.x) / (third_side.y - second_side.y *
                                                 third_side.x / second_side.x);
    center.x = (vertices[0].x * second_side.x + vertices[0].y * second_side.y -
                center.y * second_side.y) / second_side.x;
  }
  return center;
}

Circle Triangle::ninePointsCircle() const {
  Triangle middle(middle_point(vertices[0], vertices[1]),
                  middle_point(vertices[1], vertices[2]),
                  middle_point(vertices[2], vertices[0]));
  return middle.circumscribedCircle();
}