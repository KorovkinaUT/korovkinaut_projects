#include <unistd.h>
#include <errno.h>
#include <stdio.h>
#include <string.h>
#include <dirent.h>
#include <stdlib.h>
#include <sys/stat.h>

int remove_path(const char* path, int remove_directories) {
    struct stat* path_stat = (struct stat*)malloc(sizeof(struct stat));
    int return_value = lstat(path, path_stat);
    if (return_value != 0) {
        return return_value;
    }

    if (!S_ISDIR(path_stat->st_mode)) {
        return_value = unlink(path);
        return return_value;
    }

    free(path_stat);
    if (!remove_directories) {
        return -1;
    }
    DIR* dir = opendir(path);
    struct dirent* dir_dirent = readdir(dir);
    char* next_path = (char *)malloc(sizeof(char) * strlen(path) * 2);
    while (dir_dirent != NULL) {
        if (strcmp(dir_dirent->d_name, ".") == 0 || strcmp(dir_dirent->d_name, "..") == 0) {
            dir_dirent = readdir(dir); 
            continue;
        }
        next_path[0] = '\0';
        strcpy(next_path, path);
        strcat(next_path, "/");
        strcat(next_path, dir_dirent->d_name);
        return_value = remove_path(next_path, remove_directories);
    
        if (return_value != 0) {
            free(next_path);
            closedir(dir);
            return -2;
        }

        dir_dirent = readdir(dir);
    }

    free(next_path);
    closedir(dir);
    return_value = rmdir(path);
    return return_value;
}

int main(int argc, const char* argv[]) {
    if (argc < 2) {
        return -1;
    }
    int remove_directories = 0;
    int current_argument = 1;
    if (strcmp(argv[1], "-r") == 0) {
        remove_directories = 1;
        ++current_argument;
    }

    int return_value;
    while (current_argument < argc) {
        return_value = remove_path(argv[current_argument], remove_directories);
        if (return_value != 0) {
            return return_value;
        }
        ++current_argument;
    }
    return 0;
}
