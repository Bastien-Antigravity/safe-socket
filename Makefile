GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test

LIB_DIR=safesock/libsafesocket
LIB_NAME=libsafesocket

.PHONY: all build clean test build-lib

all: build


build: build-lib
	mkdir -p bin
	$(GOBUILD) -o bin/ ./cmd/...

build-lib:
	mkdir -p $(LIB_DIR)
	# Build for Linux/Unix
	$(GOBUILD) -buildmode=c-shared -o $(LIB_DIR)/$(LIB_NAME).so ./cmd/libsafesocket
	# macOS support: copy to .dylib if on Darwin or for clarity
	cp $(LIB_DIR)/$(LIB_NAME).so $(LIB_DIR)/$(LIB_NAME).dylib || true

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
