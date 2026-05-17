#ifndef SAFESOCKET_HPP
#define SAFESOCKET_HPP

#include <string>
#include <vector>
#include <stdexcept>
#include <memory>
#include <iostream>
#include <cstdint>

// Include the generated C header
#include "../libsafesocket/libsafesocket.h"

namespace safesock {

class SafeSocketError : public std::runtime_error {
public:
    explicit SafeSocketError(const std::string& message) : std::runtime_error(message) {}
};

struct SocketConfig {
    std::string public_ip;
    int deadline_ms = 0;
    int heartbeat_interval_ms = 0;
    int handshake_timeout_ms = 0;
};

class SafeSocketConnection {
public:
    explicit SafeSocketConnection(int32_t handle) : handle_(handle), closed_(false) {}
    
    ~SafeSocketConnection() {
        try {
            close();
        } catch (...) {}
    }

    // Disable copy
    SafeSocketConnection(const SafeSocketConnection&) = delete;
    SafeSocketConnection& operator=(const SafeSocketConnection&) = delete;

    void send(const std::vector<uint8_t>& data) {
        if (SafeSocket_Send(handle_, (unsigned char*)data.data(), static_cast<int>(data.size())) == -1) {
            throw SafeSocketError(getLastError());
        }
    }

    std::vector<uint8_t> receive(int max_length = 65535) {
        std::vector<uint8_t> buffer(max_length);
        int n = SafeSocket_Receive(handle_, (unsigned char*)buffer.data(), max_length);
        if (n == -1) {
            throw SafeSocketError(getLastError());
        }
        buffer.resize(n);
        return buffer;
    }

    void close() {
        if (!closed_) {
            if (SafeSocket_Close(handle_) == -1) {
                closed_ = true;
                throw SafeSocketError(getLastError());
            }
            closed_ = true;
        }
    }

    void set_deadline(double seconds) {
        if (SafeSocket_SetDeadline(handle_, seconds) == -1) {
            throw SafeSocketError(getLastError());
        }
    }

    void set_idle_timeout(double seconds) {
        if (SafeSocket_SetIdleTimeout(handle_, seconds) == -1) {
            throw SafeSocketError(getLastError());
        }
    }

private:
    std::string getLastError() const {
        char* err = SafeSocket_GetSocketError();
        return err ? std::string(err) : "Unknown error";
    }

    int32_t handle_;
    bool closed_;
};

class SafeSocket {
public:
    SafeSocket(const std::string& profile_name, const std::string& address, 
               const SocketConfig& config = {}, const std::string& socket_type = "client", 
               bool auto_connect = false) 
        : closed_(false) {
        
        handle_ = SafeSocket_CreateExtended(
            (char*)profile_name.c_str(),
            (char*)address.c_str(),
            (char*)config.public_ip.c_str(),
            (char*)socket_type.c_str(),
            config.handshake_timeout_ms,
            config.deadline_ms,
            config.heartbeat_interval_ms,
            auto_connect ? 1 : 0
        );
        
        if (handle_ == -1) {
            throw SafeSocketError(getLastError());
        }
    }

    ~SafeSocket() {
        try {
            close();
        } catch (...) {}
    }

    // Disable copy
    SafeSocket(const SafeSocket&) = delete;
    SafeSocket& operator=(const SafeSocket&) = delete;

    void open() {
        if (SafeSocket_Open(handle_) == -1) {
            throw SafeSocketError(getLastError());
        }
    }

    void close() {
        if (!closed_) {
            if (SafeSocket_Close(handle_) == -1) {
                closed_ = true;
                throw SafeSocketError(getLastError());
            }
            closed_ = true;
        }
    }

    void send(const std::vector<uint8_t>& data) {
        if (SafeSocket_Send(handle_, (unsigned char*)data.data(), static_cast<int>(data.size())) == -1) {
            throw SafeSocketError(getLastError());
        }
    }

    std::vector<uint8_t> receive(int max_length = 65535) {
        std::vector<uint8_t> buffer(max_length);
        int n = SafeSocket_Receive(handle_, (unsigned char*)buffer.data(), max_length);
        if (n == -1) {
            throw SafeSocketError(getLastError());
        }
        buffer.resize(n);
        return buffer;
    }

    void listen() {
        if (SafeSocket_Listen(handle_) == -1) {
            throw SafeSocketError(getLastError());
        }
    }

    std::unique_ptr<SafeSocketConnection> accept() {
        int32_t conn_handle = SafeSocket_Accept(handle_);
        if (conn_handle == -1) {
            throw SafeSocketError(getLastError());
        }
        return std::make_unique<SafeSocketConnection>(conn_handle);
    }

    void set_deadline(double seconds) {
        if (SafeSocket_SetDeadline(handle_, seconds) == -1) {
            throw SafeSocketError(getLastError());
        }
    }

    void set_idle_timeout(double seconds) {
        if (SafeSocket_SetIdleTimeout(handle_, seconds) == -1) {
            throw SafeSocketError(getLastError());
        }
    }

private:
    std::string getLastError() const {
        char* err = SafeSocket_GetSocketError();
        return err ? std::string(err) : "Unknown error";
    }

    int32_t handle_;
    bool closed_;
};

// Factory functions
inline std::unique_ptr<SafeSocket> create(const std::string& profile_name, const std::string& address, 
                                          const std::string& public_ip = "", const std::string& socket_type = "client", 
                                          bool auto_connect = false) {
    SocketConfig config;
    config.public_ip = public_ip;
    return std::make_unique<SafeSocket>(profile_name, address, config, socket_type, auto_connect);
}

inline std::unique_ptr<SafeSocket> create_with_config(const std::string& profile_name, const std::string& address, 
                                                       const SocketConfig& config, const std::string& socket_type = "client", 
                                                       bool auto_connect = false) {
    return std::make_unique<SafeSocket>(profile_name, address, config, socket_type, auto_connect);
}

} // namespace safesock

#endif // SAFESOCKET_HPP
