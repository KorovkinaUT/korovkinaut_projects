#include <dirent.h>
#include <string.h>
#include <stdlib.h>
#include <sys/stat.h>
#include <stdio.h>

int arg_is_mode(const char* arg) {
    if (strlen(arg) < 2) {
        return -1;
    }
    int meaning_shift = 0;
    if (arg[0] == '-' && arg[1] == 'm') {
        meaning_shift = 3;
    } else if (strlen(arg) > 6 && arg[0] == '-' && arg[1] == '-') {
        meaning_shift = 7;
    }
    if (meaning_shift > 0) {
        char meaning[4];
        meaning[0] = arg[meaning_shift];
        meaning[1] = arg[meaning_shift + 1];
        meaning[2] = arg[meaning_shift + 2];
        meaning[3] = '\0';
        return strtol(meaning, NULL, 8);
    }
    return -1;
}

int arg_is_p(const char* arg) {
    return (strcmp(arg, "-p") == 0 ? 1 : 0);
}

int next_slash(const char* path) {
    for (int i = 0; i < strlen(path); ++i) {
        if (path[i] == '/') {
            return i;
        }
    }
    return strlen(path);
}

int create_dir(char* current_path, const char* following_path, int create_parents,
                int set_mode) {
    if (strlen(following_path) == 0) {
        return 0;
    }

    DIR* dir = opendir(current_path);
    struct dirent* dir_dirent = readdir(dir);

    int count = next_slash(following_path);
    char next_dir[strlen(following_path) + 1];
    next_dir[0] = '\0';
    strncpy(next_dir, following_path, count);
    next_dir[count] = '\0';
    char next_current_path[strlen(current_path) + strlen(following_path) + 1];
    next_current_path[0] = '\0';
    strcat(next_current_path, current_path);
    if (strcmp(current_path, "/") != 0) {
        strcat(next_current_path, "/");
    }
    strcat(next_current_path, next_dir);
    char next_following_path[strlen(following_path)];
    strcpy(next_following_path, following_path + count + 1);

    while (dir_dirent != NULL) {
        if (strcmp(dir_dirent->d_name, next_dir) == 0) {
            closedir(dir);
            return create_dir(next_current_path, next_following_path, create_parents, set_mode);
        }
        dir_dirent = readdir(dir);
    }

    if (count != strlen(following_path) && !create_parents) {
        return -1;
    }

    mkdir(next_current_path, (set_mode != -1 && strlen(next_following_path) == 0
                              ? set_mode : 0777));
    closedir(dir);
    return create_dir(next_current_path, next_following_path, create_parents, set_mode);
}

int main(int argc, const char* argv[]) {
    if (argc < 2) {
        return -1;
    }
    int current_argument = 1;
    int create_parents = 0;
    int set_mode = -1;
    int return_value;
    while (current_argument < argc) {
        if (create_parents == 0) {
            create_parents = arg_is_p(argv[current_argument]);
            if (create_parents != 0) {
                ++current_argument;
                continue;
            }
        }
        if (set_mode == -1) {
            set_mode = arg_is_mode(argv[current_argument]);
            if (set_mode != -1) {
                ++current_argument;
                continue;
            }
        }

        return_value = create_dir("/", argv[current_argument] + 1, create_parents, set_mode);
        if (return_value != 0) {
            return return_value;
        }
        ++current_argument;
    }
    return 0;
}
