#pragma once

#include "intersection.h"
#include "ray.h"
#include "sphere.h"
#include "triangle.h"
#include "vector.h"

#include <cmath>
#include <optional>

std::optional<Intersection> GetIntersection(const Ray &ray,
                                            const Sphere &sphere) {
  const double epsilon = 1e-6;

  Vector oc = ray.GetOrigin() - sphere.GetCenter();

  double a = DotProduct(ray.GetDirection(), ray.GetDirection());
  double b = 2.0 * DotProduct(oc, ray.GetDirection());
  double c = DotProduct(oc, oc) - sphere.GetRadius() * sphere.GetRadius();

  double discriminant = b * b - 4 * a * c;

  if (fabs(discriminant) < epsilon) {
    discriminant = 0.0;
  }

  if (discriminant < epsilon) {
    return std::nullopt;
  }

  double sqrt_d = std::sqrt(discriminant);
  double t1 = (-b - sqrt_d) / (2 * a);
  double t2 = (-b + sqrt_d) / (2 * a);

  double t = (t1 > epsilon) ? t1 : ((t2 > epsilon) ? t2 : -1.0);

  if (t <= epsilon) {
    return std::nullopt;
  }

  Vector position = ray.GetOrigin() + t * ray.GetDirection();
  Vector normal = position - sphere.GetCenter();
  normal.Normalize();

  return Intersection(position, normal, t);
}

// Алгоритм Мёллера-Трумбора
// (https://registry.khronos.org/OpenGL-Refpages/gl4/html/reflect.xhtml)
std::optional<Intersection> GetIntersection(const Ray &ray,
                                            const Triangle &triangle) {
  const double epsilon = 1e-6;

  Vector edge1 = triangle[1] - triangle[0];
  Vector edge2 = triangle[2] - triangle[0];
  Vector ray_cross_edge2 = CrossProduct(ray.GetDirection(), edge2);
  double determinant = DotProduct(edge1, ray_cross_edge2);

  if (fabs(determinant) < epsilon) {
    return std::nullopt;
  }

  double inv_determinant = 1.0 / determinant;
  Vector s = ray.GetOrigin() - triangle[0];
  double u = inv_determinant * DotProduct(s, ray_cross_edge2);

  if (u < 0.0 || u > 1.0) {
    return std::nullopt;
  }

  Vector s_cross_edge1 = CrossProduct(s, edge1);
  double v = inv_determinant * DotProduct(ray.GetDirection(), s_cross_edge1);

  if (v < 0.0 || (u + v) > 1.0) {
    return std::nullopt;
  }

  double t = inv_determinant * DotProduct(edge2, s_cross_edge1);

  if (t <= epsilon) {
    return std::nullopt;
  }

  Vector position = ray.GetOrigin() + t * ray.GetDirection();
  Vector normal = CrossProduct(edge1, edge2);
  normal.Normalize();

  return Intersection(position, normal, t);
}

Vector Reflect(const Vector &ray, const Vector &normal) {
  return ray - 2.0 * DotProduct(ray, normal) * normal;
}

std::optional<Vector> Refract(const Vector &ray, const Vector &normal,
                              double eta) {
  const double epsilon = 1e-6;

  double dot_ray_normal = DotProduct(ray, normal);
  double k = 1.0 - eta * eta * (1.0 - dot_ray_normal * dot_ray_normal);

  if (k < -epsilon) {
    return std::nullopt;
  }

  return eta * ray - (eta * dot_ray_normal + std::sqrt(k)) * normal;
}

Vector GetBarycentricCoords(const Triangle &triangle, const Vector &point) {

  Triangle subtriangle1(point, triangle[1], triangle[2]);
  Triangle subtriangle2(triangle[0], point, triangle[2]);
  Triangle subtriangle3(triangle[0], triangle[1], point);

  double total_area = triangle.Area();

  double u = subtriangle1.Area() / total_area;
  double v = subtriangle2.Area() / total_area;
  double w = subtriangle3.Area() / total_area;

  return Vector(u, v, w);
}
