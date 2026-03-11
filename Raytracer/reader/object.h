#pragma once

#include "../geometry/sphere.h"
#include "../geometry/triangle.h"
#include "../geometry/vector.h"
#include "material.h"

#include <vector>

struct Object {
  const Material *material = nullptr;
  Triangle polygon;
  std::vector<Vector> normals;

  Object() = default;
  Object(const Triangle &polygon) : polygon(polygon) {}
  const Vector *GetNormal(size_t index) const {
    return normals.empty() ? nullptr : &normals[index];
  }
};

struct SphereObject {
  const Material *material = nullptr;
  Sphere sphere;

  SphereObject() = default;
  SphereObject(const Sphere &sphere) : sphere(sphere) {}
};
