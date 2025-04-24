#include <stdint.h>
#include <unistd.h>
#include <errno.h>

typedef struct {
  int fd;
} utf8_file_t;

uint64_t count_leading_one(uint8_t byte) {
    uint64_t count = 0;
    while ((byte & 1 << (7 - count)) > 0) {
        ++count;
    }
    return count;
}

uint64_t count_meaning_size(uint32_t symbol) {
    uint64_t count = 31;
    while ((symbol & 1 << count) == 0) {
        --count;
    }
    return count + 1;
}

int utf8_write(utf8_file_t* f, const uint32_t* str, size_t count) {
    uint64_t symbols = 0;
    uint8_t* symbol = (uint8_t*)malloc(6);
    int write_result = 1;
    if (symbol == NULL) {
        write_result = -2;
    }
    for (uint64_t i = 0; i < 6; ++i) {
        symbol[i] = 0;
    }
    while (symbols < count && write_result > 0) {
        uint64_t meaning_size = count_meaning_size(str[symbols]);
        uint64_t symbol_size;
        if (meaning_size > 7) {
            symbol_size = meaning_size / 6 + (meaning_size % 6 > 0);
        } else {
            symbol_size = 1;
        }

        size_t position = 0;
        for (uint64_t i = symbol_size - 1; i > 0; --i) {
            for (uint64_t j = 0; j < 6; ++j) {
                uint32_t bit = ((str[symbols] & 1 << position) > 0 ? 1 : 0);
                symbol[i] |= bit << j;
                ++position;
            }
            symbol[i] |= 1 << 7;
        }
        for (uint64_t j = 0; j < (symbol_size == 1 ? 7 : 8 - symbol_size); ++j) {
            uint32_t bit = ((str[symbols] & 1 << position) > 0 ? 1 : 0);
            symbol[0] |= bit << j;
            ++position;
        }
        if (symbol_size > 1) {
            for (uint64_t j = 8 - symbol_size; j < 8; ++j) {
                symbol[0] |= 1 << j;
            }
        }

        write_result = write(f->fd, symbol, symbol_size);
        for (uint64_t i = 0; i < 6; ++i) {
            symbol[i] = 0;
        }
        ++symbols;
    }
    
    free(symbol);
    if (write_result < 0) {
        errno = write_result;
        return write_result;
    }
    return symbols;
}

int utf8_read(utf8_file_t* f, uint32_t* res, size_t count) {
    uint64_t symbols = 0;
    uint8_t* bytes = (uint8_t*)malloc(6);
    int read_result = read(f->fd, bytes, 1);
    if (bytes == NULL) {
        read_result = -2;
    }

    while (symbols < count && read_result > 0) {
        uint64_t count1 = count_leading_one(bytes[0]);
        uint64_t symbol_size = (count1 == 0 ? 1 : count1);
        if (symbol_size > 6) {
            read_result = -1;
            break;
        }

        read_result = read(f->fd, bytes + 1, symbol_size - 1);
        if (read_result < 0) {
            break;
        }

        uint64_t position = 0;
        res[symbols] = 0;
        for (uint64_t i = symbol_size - 1; i > 0; --i) {
            for (uint64_t j = 0; j < 6; ++j) {
                uint32_t bit = ((bytes[i] & 1 << j) > 0 ? 1 : 0);
                res[symbols] |= bit << position;
                ++position;
            }
        }
        for (size_t j = 0; j < 8 - symbol_size; ++j) {
            uint32_t bit = ((bytes[0] & 1 << j) > 0 ? 1 : 0);
            res[symbols] |= bit << position;
            ++position;
        }

        ++symbols;
        if (symbols >= count) {
            break;
        }
        read_result = read(f->fd, bytes, 1);
    }

    free(bytes);
    if (read_result < 0) {
        errno = read_result;
        return read_result;
    }
    return symbols;
}

utf8_file_t* utf8_fromfd(int fd) {
    utf8_file_t* file = (utf8_file_t*)malloc(sizeof(utf8_file_t));
    if (file == NULL) {
        return file;
    }
    file->fd = fd;
    return file;
}
