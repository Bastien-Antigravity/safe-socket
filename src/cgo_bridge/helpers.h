#ifndef HELPERS_H
#define HELPERS_H

#include <stdlib.h>
#include <string.h>

char* get_socket_error(void);
void set_socket_error(const char* err);
void free_socket_error(char* ptr);

#endif
