#pragma once

#include "geometry/geometry.h"
#include "geometry/intersection.h"
#include "geometry/ray.h"
#include "geometry/vector.h"
#include "utils/image.h"
#include "options/camera_options.h"
#include "options/render_options.h"
#include "reader/object.h"
#include "reader/scene.h"

#include <filesystem>

Ray CameraRay(const CameraOptions &camera_options, int x, int y) {
  const double epsilon = 1e-6;

  double aspect_ratio = camera_options.screen_width /
                        static_cast<double>(camera_options.screen_height);
  double scale = std::tan(camera_options.fov * 0.5);

  double camera_x = (2.0 * (x + 0.5) / camera_options.screen_width - 1) *
                    aspect_ratio * scale;
  double camera_y =
      (1 - 2.0 * (y + 0.5) / camera_options.screen_height) * scale;

  Vector ray_dir_camera(camera_x, camera_y, -1.0);
  ray_dir_camera.Normalize();

  Vector forward = camera_options.look_from - camera_options.look_to;
  forward.Normalize();

  Vector world_up(0.0, 1.0, 0.0);
  if (DotProduct(world_up, forward) > 1.0 - epsilon) {
    world_up = Vector{0.0, 0.0, -1.0};
  } else if (DotProduct(world_up, forward) < -1.0 + epsilon) {
    world_up = Vector{0.0, 0.0, +1.0};
  }

  Vector right = CrossProduct(world_up, forward);
  right.Normalize();

  Vector up = CrossProduct(forward, right);

  Vector ray_dir_world(
      ray_dir_camera[0] * right[0] + ray_dir_camera[1] * up[0] +
          ray_dir_camera[2] * forward[0],
      ray_dir_camera[0] * right[1] + ray_dir_camera[1] * up[1] +
          ray_dir_camera[2] * forward[1],
      ray_dir_camera[0] * right[2] + ray_dir_camera[1] * up[2] +
          ray_dir_camera[2] * forward[2]);
  ray_dir_world.Normalize();

  return Ray(camera_options.look_from, ray_dir_world);
}

struct FullIntersection {
  Vector position;
  Vector normal;
  double distance;
  bool is_inside;
  const Material *material;

  FullIntersection(const Vector &pos, const Vector &norm, double dist,
                   bool inside, const Material *mat)
      : position(pos), normal(norm), distance(dist), is_inside(inside),
        material(mat) {}

  double GetDistance() const { return distance; }
  Vector GetNormal() const { return normal; }
};

std::optional<FullIntersection> ClosestIntersection(const Scene &scene,
                                                    const Ray &ray) {
  std::optional<FullIntersection> closest_intersection = std::nullopt;
  double min_distance = std::numeric_limits<double>::max();

  for (const Object &obj : scene.GetObjects()) {
    auto intersection = GetIntersection(ray, obj.polygon);
    if (intersection.has_value()) {
      Vector position = intersection->GetPosition();
      double distance = intersection->GetDistance();
      Vector geom_normal = intersection->GetNormal();

      bool is_inside = false;
      Vector normal = geom_normal;
      if (DotProduct(ray.GetDirection(), normal) > 0.0) {
        normal = -normal;
      }

      if (obj.normals.size() == 3) {
        Vector bary = GetBarycentricCoords(obj.polygon, position);
        Vector ni = bary[0] * obj.normals[0] + bary[1] * obj.normals[1] +
                    bary[2] * obj.normals[2];
        ni.Normalize();

        if (DotProduct(ray.GetDirection(), ni) > 0.0) {
          ni = -ni;
        }
        normal = ni;
      }

      if (distance < min_distance) {
        min_distance = distance;
        closest_intersection = FullIntersection(position, normal, distance,
                                                is_inside, obj.material);
      }
    }
  }

  for (const SphereObject &sphere_obj : scene.GetSphereObjects()) {
    auto intersection = GetIntersection(ray, sphere_obj.sphere);
    if (intersection.has_value()) {
      Vector position = intersection->GetPosition();
      Vector normal = intersection->GetNormal();
      double distance = intersection->GetDistance();

      bool is_inside = false;
      if (DotProduct(ray.GetDirection(), normal) > 0) {
        is_inside = true;
        normal = -normal;
      }

      if (distance < min_distance) {
        min_distance = distance;
        closest_intersection = FullIntersection(position, normal, distance,
                                                is_inside, sphere_obj.material);
      }
    }
  }

  return closest_intersection;
}

Vector OffsetPoint(const Vector &p, const Vector &n, const Vector &dir) {
  const double epsilon = 1e-4;
  return p + n * (DotProduct(dir, n) > 0.0 ? epsilon : -epsilon);
}

Vector TraceRay(const Scene &scene, const Ray &ray, int depth) {
  const double epsilon = 1e-4;

  if (depth <= 0) {
    return Vector(0.0, 0.0, 0.0);
  }

  auto intersection = ClosestIntersection(scene, ray);
  if (!intersection.has_value()) {
    return Vector(0.0, 0.0, 0.0);
  }

  Vector point = intersection->position;
  Vector normal = intersection->normal;
  bool is_inside = intersection->is_inside;
  const Material *material = intersection->material;

  Vector color = material->ambient_color + material->intensity;
  Vector total_diffuse(0.0, 0.0, 0.0);
  Vector total_specular(0.0, 0.0, 0.0);

  for (const Light &light : scene.GetLights()) {
    Vector light_dir = light.position - point;
    double light_distance = light_dir.Length();
    light_dir.Normalize();

    Ray shadow_ray(OffsetPoint(point, normal, light_dir), light_dir);
    auto shadow_hit = ClosestIntersection(scene, shadow_ray);
    if (shadow_hit.has_value() &&
        shadow_hit->distance < light_distance - epsilon) {
      continue;
    }

    double diff = std::max(0.0, DotProduct(light_dir, normal));
    total_diffuse += material->diffuse_color * diff * light.intensity;

    Vector view_dir = (-ray.GetDirection()).Normalized();
    Vector reflect_dir = Reflect(-light_dir, normal).Normalized();
    double spec_base = std::max(0.0, DotProduct(view_dir, reflect_dir));
    double spec = (material->specular_exponent > 0.0)
                      ? std::pow(spec_base, material->specular_exponent)
                      : 0.0;
    total_specular += material->specular_color * spec * light.intensity;
  }

  color += material->albedo[0] * (total_diffuse + total_specular);

  if (material->albedo[1] > 0.0 && !is_inside) {
    Vector reflect_dir = Reflect(ray.GetDirection(), normal).Normalized();
    Ray reflect_ray(OffsetPoint(point, normal, reflect_dir), reflect_dir);
    Vector reflect_color = TraceRay(scene, reflect_ray, depth - 1);
    color += material->albedo[1] * reflect_color;
  }

  if (material->albedo[2] > 0.0) {
    double eta = is_inside ? material->refraction_index
                           : (1.0 / material->refraction_index);

    auto refract_dir_opt = Refract(ray.GetDirection(), normal, eta);
    if (refract_dir_opt.has_value()) {
      Vector refract_dir = refract_dir_opt->Normalized();
      Ray refract_ray(OffsetPoint(point, normal, refract_dir), refract_dir);
      Vector refract_color = TraceRay(scene, refract_ray, depth - 1);

      double tr = is_inside ? 1.0 : material->albedo[2];
      color += tr * refract_color;
    }
  }

  return color;
}

Vector PixelColorDepth(const Scene &scene, const Ray &ray, double &max_depth) {
  auto closest_intersection = ClosestIntersection(scene, ray);

  if (closest_intersection.has_value()) {
    max_depth = std::max(max_depth, closest_intersection->GetDistance());
    return Vector(closest_intersection->GetDistance(),
                  closest_intersection->GetDistance(),
                  closest_intersection->GetDistance());
  }

  return Vector(1.0, 1.0, 1.0);
}

Vector PixelColorNormal(const Scene &scene, const Ray &ray) {
  auto closest_intersection = ClosestIntersection(scene, ray);

  if (closest_intersection.has_value()) {
    return 0.5 * closest_intersection->GetNormal() + 0.5;
  }

  return Vector(0.0, 0.0, 0.0);
}

void GammaCorrection(Vector &color) {
  double gamma = 1.0 / 2.2;
  color[0] = std::pow(color[0], gamma);
  color[1] = std::pow(color[1], gamma);
  color[2] = std::pow(color[2], gamma);
}

void ToneMapping(std::vector<std::vector<Vector>> &colors, double max_color) {
  double c = max_color;
  double c_sq = c * c;

  for (auto &row : colors) {
    for (Vector &color : row) {
      Vector v_in = color;
      Vector v_out = (v_in * (1.0 + 1.0 / c_sq * v_in)) / (1.0 + v_in);
      color = v_out;
    }
  }
}

Image Render(const std::filesystem::path &path,
             const CameraOptions &camera_options,
             const RenderOptions &render_options) {
  const double epsilon = 1e-6;

  Scene scene = ReadScene(path);
  Image image(camera_options.screen_width, camera_options.screen_height);

  double max_depth = 0.0;
  double max_color = 0.0;
  std::vector<std::vector<Vector>> colors(
      camera_options.screen_height,
      std::vector<Vector>(camera_options.screen_width));

  for (int y = 0; y < camera_options.screen_height; ++y) {
    for (int x = 0; x < camera_options.screen_width; ++x) {
      Ray ray = CameraRay(camera_options, x, y);
      Vector color;

      switch (render_options.mode) {
      case RenderMode::kFull:
        color = TraceRay(scene, ray, render_options.depth);

        max_color = std::max(max_color, color[0]);
        max_color = std::max(max_color, color[1]);
        max_color = std::max(max_color, color[2]);
        break;

      case RenderMode::kDepth:
        color = PixelColorDepth(scene, ray, max_depth);
        break;

      case RenderMode::kNormal:
        color = PixelColorNormal(scene, ray);
        break;
      }

      colors[y][x] = color;
    }
  }

  if (render_options.mode == RenderMode::kFull && max_color > 0.0) {
    ToneMapping(colors, max_color);
  }

  for (int y = 0; y < camera_options.screen_height; ++y) {
    for (int x = 0; x < camera_options.screen_width; ++x) {
      Vector &color = colors[y][x];

      switch (render_options.mode) {
      case RenderMode::kFull:
        GammaCorrection(color);
        color *= 255.0;
        break;

      case RenderMode::kDepth:
        if (!(fabs(color[0] - 1.0) < epsilon &&
              fabs(color[1] - 1.0) < epsilon &&
              fabs(color[2] - 1.0) < epsilon)) {
          color *= 1.0 / max_depth;
        }
        color *= 255.0;
        break;

      case RenderMode::kNormal:
        color *= 255.0;
        break;
      }

      image.SetPixel(RGB{static_cast<int>(color[0]), static_cast<int>(color[1]),
                         static_cast<int>(color[2])},
                     y, x);
    }
  }

  return image;
}
