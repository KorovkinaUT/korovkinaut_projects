#pragma once

#include "../geometry/vector.h"

#include <math.h>

struct CameraOptions {
  int screen_width;
  int screen_height;
  double fov = M_PI / 2;
  Vector look_from = {0., 0., 0.};
  Vector look_to = {0., 0., -1.};
};
