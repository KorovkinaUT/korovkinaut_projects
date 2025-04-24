#include <stdlib.h>
#include <stdio.h>
#include <stdint.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/stat.h>
#include <sys/mman.h>
#include <assert.h>

enum {
    PAGE_SIZE = 4096,
    PAGE_MASK_SIZE = 512
};

typedef struct {
    int fd;
    void* base_addr;
    uint64_t* page_mask;
    uint64_t curr_page_count;
    uint64_t allowed_page_count;
} file_allocator_t;

size_t CountOnes(uint64_t mask)
{
    size_t count = 0;
    for (size_t i = 0; i < 64; ++i)
    {
        if ((mask & (1ull << i)) != 0)
        {
            ++count;
        }
    }
    return count;
}

void falloc_init(file_allocator_t *allocator, const char *filepath, uint64_t allowed_page_count)
{
    allocator->allowed_page_count = allowed_page_count;
    assert((allocator->fd = open(filepath, O_RDWR | O_CREAT, 0777)) != -1 && "Open failed");
    assert(ftruncate(allocator->fd, (allowed_page_count + 1) * PAGE_SIZE) == 0 &&
           "Ftruncate failed.");
    allocator->base_addr = mmap(NULL, (allowed_page_count + 1) * PAGE_SIZE, PROT_READ | PROT_WRITE,
                                MAP_SHARED, allocator->fd, 0);
    assert(allocator->base_addr != MAP_FAILED && "Mmap failed.");
    allocator->page_mask = (uint64_t *)(allocator->base_addr + allowed_page_count * PAGE_SIZE);
    allocator->curr_page_count = 0;
    for (size_t i = 0; i < PAGE_SIZE / 8; ++i)
    {
        allocator->curr_page_count += CountOnes(allocator->page_mask[i]);
    }
}

void falloc_destroy(file_allocator_t *allocator)
{
    assert(munmap(allocator->base_addr, (allocator->allowed_page_count + 1) * PAGE_SIZE) == 0 &&
           "Mumap failed.");
    allocator->base_addr = NULL;
    allocator->page_mask = NULL;
    assert(close(allocator->fd) == 0 && "Close failed.");
}

int ZeroBit(uint64_t mask)
{
    for (size_t i = 0; i < 64; ++i)
    {
        if ((mask & (1ull << i)) == 0)
        {
            return i;
        }
    }
    return -1;
}

void *falloc_acquire_page(file_allocator_t *allocator)
{
    if (allocator->curr_page_count >= allocator->allowed_page_count)
    {
        return NULL;
    }
    int passed = 0;
    for (size_t i = 0; i < PAGE_SIZE / 8; ++i)
    {
        int bit = ZeroBit(allocator->page_mask[i]);
        if (bit != -1)
        {
            allocator->page_mask[i] |= 1ull << bit;
            ++allocator->curr_page_count;
            return allocator->base_addr + (passed + bit) * PAGE_SIZE;
        }
        passed += 64;
    }
    return NULL;
}

void falloc_release_page(file_allocator_t *allocator, void **addr)
{
    size_t bit = 0;
    while (allocator->base_addr + bit * PAGE_SIZE != *addr)
    {
        ++bit;
    }
    allocator->page_mask[bit / 64] -= 1 << (bit % 64);
    *addr = NULL;
    --allocator->curr_page_count;
}