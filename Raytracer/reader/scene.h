#pragma once

#include "../geometry/vector.h"
#include "light.h"
#include "object.h"

#include <filesystem>
#include <fstream>
#include <iostream>
#include <string>
#include <unordered_map>
#include <vector>

class Scene {
public:
  const std::vector<Object> &GetObjects() const { return objects_; }
  const std::vector<SphereObject> &GetSphereObjects() const {
    return sphere_objects_;
  }
  const std::vector<Light> &GetLights() const { return lights_; }
  const std::unordered_map<std::string, Material> &GetMaterials() const {
    return materials_;
  }

  void AddObject(const Object &obj) { objects_.push_back(obj); }
  void AddSphereObject(const SphereObject &sphere_obj) {
    sphere_objects_.push_back(sphere_obj);
  }
  void AddLight(const Light &light) { lights_.push_back(light); }
  void AddMaterial(const std::string &name, const Material &material) {
    materials_[name] = material;
  }
  void AddVertices(const std::vector<Vector> &verticies) {
    verticies_ = verticies;
  }
  void AddNormals(const std::vector<Vector> &normals) { normals_ = normals; }

private:
  std::vector<Vector> verticies_;
  std::vector<Vector> normals_;
  std::vector<Object> objects_;
  std::vector<SphereObject> sphere_objects_;
  std::vector<Light> lights_;
  std::unordered_map<std::string, Material> materials_;
};

std::unordered_map<std::string, Material>
ReadMaterials(const std::filesystem::path &path) {
  std::ifstream file_stream(path);

  std::unordered_map<std::string, Material> materials;
  Material current_material;
  std::string line;

  while (std::getline(file_stream, line)) {
    std::istringstream line_stream(line);
    std::string command;
    line_stream >> command;

    if (command == "newmtl") {
      if (!current_material.name.empty()) {
        materials[current_material.name] = current_material;
      }

      line_stream >> current_material.name;
      current_material.specular_exponent = 1.0;
      current_material.refraction_index = 1.0;
      current_material.ambient_color = Vector(0, 0, 0);
      current_material.diffuse_color = Vector(0, 0, 0);
      current_material.specular_color = Vector(0, 0, 0);
      current_material.intensity = Vector(0, 0, 0);
      current_material.albedo = Vector(1, 0, 0);

    } else if (command == "Ka") {
      double r, g, b;
      line_stream >> r >> g >> b;
      current_material.ambient_color = Vector(r, g, b);

    } else if (command == "Kd") {
      double r, g, b;
      line_stream >> r >> g >> b;
      current_material.diffuse_color = Vector(r, g, b);

    } else if (command == "Ks") {
      double r, g, b;
      line_stream >> r >> g >> b;
      current_material.specular_color = Vector(r, g, b);

    } else if (command == "Ke") {
      double r, g, b;
      line_stream >> r >> g >> b;
      current_material.intensity = Vector(r, g, b);

    } else if (command == "Ns") {
      line_stream >> current_material.specular_exponent;

    } else if (command == "Ni") {
      line_stream >> current_material.refraction_index;

    } else if (command == "al") {
      double x, y, z;
      line_stream >> x >> y >> z;
      current_material.albedo = Vector(x, y, z);
    }
  }

  if (!current_material.name.empty()) {
    materials[current_material.name] = current_material;
  }

  return materials;
}

Scene ReadScene(const std::filesystem::path &path) {
  std::ifstream file_stream(path);

  Scene scene;

  std::vector<Vector> vertices;
  std::vector<Vector> normals;
  std::string current_material;

  std::string line;
  while (std::getline(file_stream, line)) {
    std::istringstream line_stream(line);
    std::string command;
    line_stream >> command;

    if (command == "v") {
      double x, y, z;
      line_stream >> x >> y >> z;
      vertices.push_back(Vector(x, y, z));

    } else if (command == "vn") {
      double x, y, z;
      line_stream >> x >> y >> z;
      normals.push_back(Vector(x, y, z));

    } else if (command == "f") {
      std::vector<int> vertex_indices;
      std::vector<int> normal_indices;
      std::string token;

      while (line_stream >> token) {
        std::istringstream token_stream(token);
        std::string token_part;
        std::vector<std::string> indices;

        // v или v/vt или v/vt/vn или v//vn
        while (std::getline(token_stream, token_part, '/')) {
          indices.push_back(token_part);
        }

        int first_index = std::stoi(indices[0]);
        vertex_indices.push_back(
            first_index > 0 ? first_index - 1 : vertices.size() + first_index);

        if (indices.size() >= 3) {
          int last_index = std::stoi(indices[2]);
          normal_indices.push_back(
              last_index > 0 ? last_index - 1 : normals.size() + last_index);
        }
      }

      // Триангулируем многоугольник
      for (size_t i = 1; i < vertex_indices.size() - 1; ++i) {
        Triangle triangle(vertices[vertex_indices[0]],
                          vertices[vertex_indices[i]],
                          vertices[vertex_indices[i + 1]]);

        Object obj(triangle);

        if (!normal_indices.empty()) {
          obj.normals.push_back(normals[normal_indices[0]]);
          obj.normals.push_back(normals[normal_indices[i]]);
          obj.normals.push_back(normals[normal_indices[i + 1]]);
        }

        if (!current_material.empty()) {
          const auto &materials = scene.GetMaterials();
          auto it = materials.find(current_material);
          if (it != materials.end()) {
            obj.material = &(it->second);
          }
        }

        scene.AddObject(obj);
      }

    } else if (command == "mtllib") {
      std::string mtl_filename;
      line_stream >> mtl_filename;
      auto mtl_path = path.parent_path() / mtl_filename;
      auto materials = ReadMaterials(mtl_path);

      for (auto &key_value : materials) {
        scene.AddMaterial(key_value.first, key_value.second);
      }

    } else if (command == "usemtl") {
      line_stream >> current_material;

    } else if (command == "S") {
      double x, y, z, r;
      line_stream >> x >> y >> z >> r;

      SphereObject sphere_obj(Sphere(Vector(x, y, z), r));

      if (!current_material.empty()) {
        const auto &materials = scene.GetMaterials();
        auto it = materials.find(current_material);
        if (it != materials.end()) {
          sphere_obj.material = &(it->second);
        }
      }

      scene.AddSphereObject(sphere_obj);

    } else if (command == "P") {
      double x, y, z, r, g, b;
      line_stream >> x >> y >> z >> r >> g >> b;

      Light light;
      light.position = Vector(x, y, z);
      light.intensity = Vector(r, g, b);

      scene.AddLight(light);
    }
  }

  scene.AddVertices(vertices);
  scene.AddNormals(normals);

  return scene;
}
