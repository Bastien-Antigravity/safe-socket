CXX = g++
CXXFLAGS = -std=c++14 -Wall -I..
LDFLAGS = -L../../libsafesocket -lsafesocket

# Detect OS for library extension
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
    LIB_EXT = .dylib
else
    LIB_EXT = .so
endif

TARGET = basic_usage
LIB_PATH = ../../libsafesocket/libsafesocket$(LIB_EXT)

all: $(TARGET)

$(TARGET): basic_usage.cpp $(LIB_PATH)
	$(CXX) $(CXXFLAGS) basic_usage.cpp -o $(TARGET) $(LDFLAGS)

clean:
	rm -f $(TARGET)
