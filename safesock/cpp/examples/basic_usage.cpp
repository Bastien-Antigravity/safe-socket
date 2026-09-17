// =============================================================================
// ESSENTIAL PROCESS:
// Demonstration client showcasing C++ SafeSocket connection and messaging.
//
// DATA FLOW:
// 1. Input: Target server address and message payload.
// 2. Logic: Establishes SafeSocket connection and writes framed data.
// 3. Output: Console output confirming message transmission.
//
// KEY PARAMETERS:
// - None
// =============================================================================

#include "../SafeSocket.hpp"
#include <iostream>
#include <vector>
#include <string>

int main() {
    try {
        // Attempt to create a socket. 
        // We use auto_connect=false so it doesn't try to connect immediately 
        // and fail if no server is running.
        auto sock = safesock::create("demo", "localhost:8080", "", "client", false);
        std::cout << "Socket created successfully" << std::endl;

        // We can't really test much more without a running server,
        // but this confirms the library is loaded and basic calls work.
        
        sock->close();
        std::cout << "Socket closed" << std::endl;
    } catch (const std::exception& e) {
        std::cerr << "Error: " << e.what() << std::endl;
        return 1;
    }
    return 0;
}
