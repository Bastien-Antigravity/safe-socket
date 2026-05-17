GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test

LIB_DIR=safesock/libsafesocket
LIB_NAME=libsafesocket

# Detect OS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
    LIB_EXT = .dylib
    LDFLAGS_SHARED = -ldflags="-extldflags=-Wl,-install_name,@rpath/$(LIB_NAME)$(LIB_EXT)"
else
    LIB_EXT = .so
    LDFLAGS_SHARED = 
endif

.PHONY: all build clean test build-lib

all: build

build: build-lib
	mkdir -p bin
	$(GOBUILD) -o bin/ ./cmd/...

build-lib:
	mkdir -p $(LIB_DIR)
	$(GOBUILD) $(LDFLAGS_SHARED) -buildmode=c-shared -o $(LIB_DIR)/$(LIB_NAME)$(LIB_EXT) ./cmd/libsafesocket

build-dll:
	mkdir -p $(LIB_DIR)
	# Requires mingw-w64 for cross-compilation
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 $(GOBUILD) -buildmode=c-shared -o $(LIB_DIR)/$(LIB_NAME).dll ./cmd/libsafesocket

clean:
	$(GOCLEAN)
	rm -rf bin/
	rm -rf $(LIB_DIR)/*.so $(LIB_DIR)/*.dylib $(LIB_DIR)/*.dll $(LIB_DIR)/*.h

test:
	$(GOTEST) -v ./...
