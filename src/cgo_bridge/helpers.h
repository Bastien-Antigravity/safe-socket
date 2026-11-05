#ifndef HELPERS_H
#define HELPERS_H

#include <stdlib.h>
#include <string.h>

extern _Thread_local char* last_socket_error;

void set_socket_error(const char* err);

#endif
