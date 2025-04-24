#include <assert.h>
#include <pcre.h>
#include <stdio.h>
#include <ftw.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/mman.h>
#include <string.h>

pcre* compiled;

int count_empty_lines(const char* str) {
    int count = 0;
    while (count < strlen(str) && str[count] == '\n') {
        count += 1;
    }
    return count;
}

int minigrep(const char* filepath, const struct stat* file_stat, int flag) {
    if (flag != FTW_F) {
        return 0;
    }
    int fd = open(filepath, O_RDONLY);
    assert(fd != -1 && "Open failed.");
    char* content = (char*)mmap(NULL, file_stat->st_size + 1, PROT_READ | PROT_WRITE,
                                MAP_PRIVATE, fd, 0);
    assert(content != MAP_FAILED && "Mmap failed.");
    assert(close(fd) == 0 && "Close failed.");
    content[file_stat->st_size] = '\0';

    int line = count_empty_lines(content) + 1;
    int index = line;
    char* str = strtok(content, "\n");
    int ovector[strlen(str) * 2];
    while (str != NULL) {
        int count = pcre_exec(compiled, NULL, str, strlen(str), 0, 0, ovector, strlen(str) * 2);
        if (count > 0) {
            printf("%s:%d: %s\n", filepath, line, str);
        }
        index += strlen(str);
        line += count_empty_lines(content + index) + 1;
        str = strtok(NULL, "\n");
    }

    return 0;
}

int main(int argc, char* argv[]) {
    const char* regex = argv[1];
    const char* dir = argv[2];
    const char* error;
    int erroroffset;

    compiled = pcre_compile(regex, 0, &error, &erroroffset, NULL);
    if (compiled == NULL) {
        fprintf(stderr, "%s", error);
        return -1;
    }

    ftw(dir, minigrep, 20);
    pcre_free(compiled);

    return 0;
}