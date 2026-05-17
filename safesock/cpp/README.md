# SafeSocket C++ SDK

A header-only, object-oriented C++14 wrapper for the `safe-socket` ecosystem.

## Usage

```cpp
#include "SafeSocket.hpp"

int main() {
    try {
        // Factory creation
        auto sock = safesock::create("tcp-hello", "localhost:8080", "", "client", true);
        
        sock->send({ 'H', 'e', 'l', 'l', 'o' });
        auto data = sock->receive(1024);
        
    } catch (const safesock::SafeSocketError& e) {
        std::cerr << "Error: " << e.what() << std::endl;
    }
    return 0;
}
```

## Integration

Simply include `SafeSocket.hpp` in your project and link against `libsafesocket`.
Ensure the `libsafesocket.h` header is in your include path.
