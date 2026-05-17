CXX = g++
CXXFLAGS = -std=c++20 -Wall -I..
LDFLAGS = -L../../libsafesocket -lsafesocket

# Detect OS for library extension and rpath
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
    LIB_EXT = .dylib
    LDFLAGS += -Wl,-rpath,@loader_path/../../libsafesocket
else
    LIB_EXT = .so
    LDFLAGS += -Wl,-rpath,$$ORIGIN/../../libsafesocket
endif

TARGETS = basic_usage matrix_client
LIB_PATH = ../../libsafesocket/libsafesocket$(LIB_EXT)

all: $(TARGETS)

basic_usage: basic_usage.cpp $(LIB_PATH)
	$(CXX) $(CXXFLAGS) basic_usage.cpp -o basic_usage $(LDFLAGS)

matrix_client: matrix_client.cpp $(LIB_PATH)
	$(CXX) $(CXXFLAGS) matrix_client.cpp -o matrix_client $(LDFLAGS)

clean:
	rm -f $(TARGETS)
