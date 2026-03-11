#include "../options/camera_options.h"
#include "../options/render_options.h"
#include "../raytracer.h"
#include "../utils/image.h"
#include "test_cases/commons.h"

#include <cmath>
#include <numbers>
#include <optional>
#include <string_view>

void CheckImage(std::string_view obj_filename, std::string_view result_filename,
                const CameraOptions &camera_options,
                const RenderOptions &render_options,
                const std::optional<std::filesystem::path> &output_path =
                    std::filesystem::current_path() / "output.png") {
  static const auto kTestsDir = std::filesystem::current_path() / "test_case";
  auto image = Render(kTestsDir / obj_filename, camera_options, render_options);
  if (output_path) {
    image.Write(*output_path);
  }
  Compare(image, Image{kTestsDir / result_filename});
}

void run_shading_parts_test() {
  CameraOptions camera_opts{640, 480};
  return CheckImage("shading_parts/scene.obj", "shading_parts/scene.png",
                    camera_opts, {1});
}

void run_triangle_test() {
  CameraOptions camera_opts{.screen_width = 640,
                            .screen_height = 480,
                            .look_from = {0., 2., 0.},
                            .look_to = {0., 0., 0.}};
  return CheckImage("triangle/scene.obj", "triangle/scene.png", camera_opts,
                    {1});
}

void run_triangle2_test() {
  CameraOptions camera_opts{.screen_width = 640,
                            .screen_height = 480,
                            .look_from = {0., -2., 0.},
                            .look_to = {0., 0., 0.}};
  return CheckImage("triangle/scene.obj", "triangle/black.png", camera_opts,
                    {1});
}

void run_box_with_spheres_test() {
  CameraOptions camera_opts{.screen_width = 640,
                            .screen_height = 480,
                            .fov = std::numbers::pi / 3,
                            .look_from = {0., .7, 1.75},
                            .look_to = {0., .7, 0.}};
  return CheckImage("box/cube.obj", "box/cube.png", camera_opts, {4});
}

void run_classic_box_test() {
  CameraOptions camera_opts{.screen_width = 500,
                            .screen_height = 500,
                            .look_from = {-.5, 1.5, .98},
                            .look_to = {0., 1., 0.}};
  CheckImage("classic_box/CornellBox.obj", "classic_box/first.png", camera_opts,
             {4});
  camera_opts.look_from = {-.9, 1.9, -1};
  camera_opts.look_to = {0., 0., 0.};
  CheckImage("classic_box/CornellBox.obj", "classic_box/second.png",
             camera_opts, {4});
}

void run_mirrors_test() {
  CameraOptions camera_opts{.screen_width = 800,
                            .screen_height = 600,
                            .look_from = {2., 1.5, -.1},
                            .look_to = {1., 1.2, -2.8}};
  CheckImage("mirrors/scene.obj", "mirrors/result.png", camera_opts, {9});
}

void run_distored_box_test() {
  CameraOptions camera_opts{.screen_width = 500,
                            .screen_height = 500,
                            .look_from = {-0.5, 1.5, 1.98},
                            .look_to = {0., 1., 0.}};
  CheckImage("distorted_box/CornellBox.obj", "distorted_box/result.png",
             camera_opts, {4});
}

void run_deer_test() {
  CameraOptions camera_opts{.screen_width = 500,
                            .screen_height = 500,
                            .look_from = {100., 200., 150.},
                            .look_to = {0., 100., 0.}};
  CheckImage("deer/CERF_Free.obj", "deer/result.png", camera_opts, {1});
}

int main() {
  run_shading_parts_test();
}