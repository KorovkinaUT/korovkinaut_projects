#include <assert.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/wait.h>

#include <errno.h>

enum {
    MAX_ARGS_COUNT = 256,
    MAX_CHAIN_LINKS_COUNT = 256
};

typedef struct {
    char command[100];
    uint64_t argc;
    char* argv[MAX_ARGS_COUNT];
} chain_link_t;

typedef struct {
    uint64_t chain_links_count;
    chain_link_t* chain_links[MAX_CHAIN_LINKS_COUNT];
} chain_t;

void create_chain(int argc, char* argv[], chain_t* chain) {
    chain_link_t* current = (chain_link_t*)malloc(sizeof(chain_link_t));
    assert(current != NULL && "Malloc failed.");
    for (int i = 1; i < argc; ++i) {
        if (strcmp(argv[i], "|") == 0) {
            continue;
        }
        if (i == 1) {
            strcpy(current->command, argv[i]);
            current->argv[0] = current->command;
            current->argc = 1;
        } else if (strcmp(argv[i - 1], "|") == 0) {
            chain->chain_links[chain->chain_links_count] = current;
            chain->chain_links_count += 1;
            assert((current = (chain_link_t*)malloc(sizeof(chain_link_t))) != NULL &&
                    "Malloc failed.");
            strcpy(current->command, argv[i]);
            current->argv[0] = current->command;
            current->argc = 1;
        } else {
            current->argv[current->argc] = argv[i];
            current->argc += 1;
        }
    }
    chain->chain_links[chain->chain_links_count] = current;
    chain->chain_links_count += 1;
}

void run_chain(chain_t* chain) {
    int pipe_fd[2];
    int pid;
    int out_fd;
    for (int i = 0; i < chain->chain_links_count; ++i) {
        if (i < chain->chain_links_count - 1) {
            assert(pipe(pipe_fd) != -1 && "Pipe failed.");
        }

        assert((pid = fork()) != -1 && "Fork failed.");
        if (pid == 0) {
            if (i > 0) {
                dup2(out_fd, STDIN_FILENO);
            }
            if (i < chain->chain_links_count - 1) {
                close(pipe_fd[0]);
                dup2(pipe_fd[1], STDOUT_FILENO);
            }
            execvp(chain->chain_links[i]->command, chain->chain_links[i]->argv);
            assert(0 && "Exec failed.");
        }

        if (i > 0) {
            assert(close(out_fd) == 0 && "Close failed.");
        }
        if (i < chain->chain_links_count - 1) {
            assert(close(pipe_fd[1]) == 0 && "Close failed.");
            out_fd = pipe_fd[0];
        }
    }

    for(int i = 0; i < chain->chain_links_count; ++i) {
        free(chain->chain_links[i]);
        assert(wait(NULL) != -1 && "Child terminate with error.");
    }
}

int main(int argc, char* argv[]) {
    chain_t chain;

    int new_argc = 1;
    char* new_argv[MAX_CHAIN_LINKS_COUNT + MAX_ARGS_COUNT];
    new_argv[0] = argv[0];
    char* next = strtok(argv[1], " ");
    while(next != NULL) {
        new_argv[new_argc] = next;
        new_argc += 1;
        next = strtok(NULL, " ");
    }

    create_chain(new_argc, new_argv, &chain);
    run_chain(&chain);
    return 0;
}