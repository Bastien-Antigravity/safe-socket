// =============================================================================
// ESSENTIAL PROCESS:
// Manages thread-local C error string storage for the CGO FFI bridge, providing
// safe getter, setter, and deallocation primitives across Darwin and Linux ABIs.
//
// DATA FLOW:
// 1. Input: Error strings from Go FFI exports via set_socket_error.
// 2. Logic: Duplicates error strings into _Thread_local storage and frees prior allocations.
// 3. Output: Thread-local error pointer retrieved via get_socket_error and cleared via free_socket_error.
//
// KEY PARAMETERS:
// - last_socket_error: Thread-local pointer holding the latest C error string.
// =============================================================================

#include "helpers.h"

_Thread_local char* last_socket_error = NULL;

// -----------------------------------------------------------------------------

char* get_socket_error(void) {
    return last_socket_error;
}

// -----------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------

void free_socket_error(char* ptr) {
    if (ptr == NULL) {
        return;
    }
    if (ptr == last_socket_error) {
        last_socket_error = NULL;
    }
    free(ptr);
}
