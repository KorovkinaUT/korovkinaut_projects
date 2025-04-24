#include <linux/limits.h>

#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/wait.h>
#include <sys/ptrace.h>
#include <sys/user.h>

typedef struct Counter{
    char filename[PATH_MAX];
    int counter;
    struct Counter* next;
} Counter;

typedef struct Counters{
    struct Counter* head;
} Counters;

void increment(Counters* counters, char* filename, int value) {
    Counter* current = counters->head;
    while (current != NULL) {
        if (strncmp(current->filename, filename, PATH_MAX) == 0) {
            current->counter += value;
            return;
        }
        current = current->next;
    }
    Counter* new_head = malloc(sizeof(Counter));
    new_head->next = counters->head;
    new_head->counter = value;
    strncpy(new_head->filename, filename, PATH_MAX - 1);
    counters->head = new_head;
}

void print(Counters* counters) {
    Counter* current = counters->head;
    while (current != NULL) {
        printf("%s:%d\n", current->filename, current->counter);
        current = current->next;
    }
}

int main(int argc, char *argv[]) {
    Counters* counters = malloc(sizeof(Counter));
    counters->head = NULL;
    
    pid_t pid = fork();
    assert(pid != -1 && "Fork failed.");
    if (pid == 0) {
        assert(ptrace(PTRACE_TRACEME, 0, NULL, NULL) != -1 && "Ptrace failed.");
        execvp(argv[1], argv + 1);
        assert(0 && "Exec failed.");
    }

    int status;
    struct user_regs_struct regs;
    int started = 1;
    char fdpath[200];
    char filepath[200];
    ssize_t length;
    while (1) {
        wait(&status);
        if (WIFEXITED(status)) {
            break;
        }
        assert(ptrace(PTRACE_GETREGS, pid, NULL, &regs) != -1 && "Ptrace failed.");
        if (regs.orig_rax == 1) {
            if (started) {
                sprintf(fdpath, "/proc/%d/fd/%lld", pid, regs.rdi);
                assert((length = readlink(fdpath, filepath, 200)) != -1 && "Readlink failed.");
                filepath[length] = '\0';
                started = 0;
            } else {
                increment(counters, filepath, regs.rax);
                started = 1;
            }
        }
        assert(ptrace(PTRACE_SYSCALL, pid, NULL, NULL) != -1 && "Ptrace failed.");
    }

    print(counters);
}
