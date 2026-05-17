#include "../SafeSocket.hpp"
#include <iostream>
#include <string>
#include <vector>
#include <chrono>

int main(int argc, char* argv[]) {
    std::string address = "127.0.0.1:9999";
    if (argc > 1) {
        address = argv[1];
    }

    std::cout << "C++ Client: Connecting to " << address << std::endl;

    try {
        std::string public_ip = "1.2.3.4-cpp";
        safesock::SocketConfig config;
        config.public_ip = public_ip;

        // Create client with tcp-hello profile and custom identity
        auto client = safesock::create_with_config("tcp-hello:cpp-matrix", address, config, "client", true);
        std::cout << "C++ Client: Connected" << std::endl;

        // --- TEST 1: Basic Ping ---
        std::string payload = "ping-cpp";
        std::cout << "C++ Client: Sending " << payload << std::endl;
        std::vector<uint8_t> data(payload.begin(), payload.end());
        client->send(data);

        auto response = client->receive();
        std::string decoded(response.begin(), response.end());
        if (decoded != "echo:" + payload) {
             std::cerr << "C++ Client: FAILURE - Basic ping failed. Got: " << decoded << std::endl;
             return 1;
        }

        // --- TEST 2: Metadata Verification ---
        std::cout << "C++ Client: Verifying metadata..." << std::endl;
        std::string meta_req = "meta_request";
        std::vector<uint8_t> meta_data(meta_req.begin(), meta_req.end());
        client->send(meta_data);
        
        auto meta_resp_vec = client->receive();
        std::string meta_resp(meta_resp_vec.begin(), meta_resp_vec.end());
        
        if (meta_resp.find("meta:cpp-matrix") == std::string::npos || meta_resp.find(public_ip) == std::string::npos) {
            std::cerr << "C++ Client: FAILURE - Metadata mismatch. Got: " << meta_resp << std::endl;
            return 1;
        }
        std::cout << "C++ Client: Metadata verified" << std::endl;

        // --- TEST 3: Large Payload (1MB) ---
        std::cout << "C++ Client: Testing 1MB payload..." << std::endl;
        std::vector<uint8_t> large_payload(1024 * 1024, 'C');
        auto start = std::chrono::high_resolution_clock::now();
        client->send(large_payload);
        
        auto large_resp = client->receive(1024 * 1024 + 10);
        auto end = std::chrono::high_resolution_clock::now();
        auto duration = std::chrono::duration_cast<std::chrono::milliseconds>(end - start).count();

        if (large_resp.size() != large_payload.size() + 5) {
            std::cerr << "C++ Client: FAILURE - Large payload length mismatch. Expected " << large_payload.size() + 5 << " but got " << large_resp.size() << std::endl;
            return 1;
        }
        
        std::cout << "C++ Client: 1MB payload verified in " << duration << "ms" << std::endl;

        std::cout << "C++ Client: ALL TESTS SUCCESS" << std::endl;
        return 0;

    } catch (const safesock::SafeSocketError& e) {
        std::cerr << "C++ Client: ERROR - " << e.what() << std::endl;
        return 1;
    } catch (const std::exception& e) {
        std::cerr << "C++ Client: Unexpected error - " << e.what() << std::endl;
        return 1;
    }
}
