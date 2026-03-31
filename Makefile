GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

CAPI_SRC=python/capi/main.go
PYTHON_LIB_DIR=python/safesocket
LIB_NAME=libsafe_socket

# Detect OS
ifeq ($(OS),Windows_NT)
    LIB_EXT=.dll
else
    UNAME_S := $(shell uname -s)
    ifeq ($(UNAME_S),Linux)
        LIB_EXT=.so
    endif
    ifeq ($(UNAME_S),Darwin)
        LIB_EXT=.dylib
    endif
endif

LIB_OUT=$(PYTHON_LIB_DIR)/$(LIB_NAME)$(LIB_EXT)

.PHONY: all build clean test python-build

all: build

build:
	$(GOBUILD) -o $(LIB_OUT) -buildmode=c-shared $(CAPI_SRC)

python-build: build
	cd python && python3 -m build

clean:
	rm -f $(PYTHON_LIB_DIR)/$(LIB_NAME).*
	rm -f $(PYTHON_LIB_DIR)/*.h

test:
	$(GOTEST) -v ./...
