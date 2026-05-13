#include "helpers.h"

char* last_socket_error = NULL;

void set_socket_error(const char* err) {
    if (last_socket_error != NULL) {
        free(last_socket_error);
    }
    if (err == NULL) {
        last_socket_error = NULL;
    } else {
        last_socket_error = strdup(err);
    }
}
