VERSION := $(shell cat VERSION.txt 2>/dev/null || echo "0.0.1")

.PHONY: all build build-lib core test version clean

all: build

version:
	@echo $(VERSION)

build-lib core:
	@echo "Building CGO shared library libsafesocket (version $(VERSION))..."
	@mkdir -p safesock/libsafesocket
	@if [ "$$(uname -s)" = "Darwin" ]; then \
		go build -buildmode=c-shared -o safesock/libsafesocket/libsafesocket.dylib ./cmd/libsafesocket; \
	else \
		go build -buildmode=c-shared -o safesock/libsafesocket/libsafesocket.so ./cmd/libsafesocket; \
	fi

build: build-lib
	@echo "Building repository (version $(VERSION))..."
	@if [ -f "go.mod" ]; then go build ./...; fi
	@if [ -f "Cargo.toml" ]; then cargo build --release; fi
	@if [ -f "setup.py" ] || [ -f "pyproject.toml" ]; then python3 -m build; fi

test:
	@echo "Running tests (version $(VERSION))..."
	@if [ -f "go.mod" ]; then go test -v ./...; fi
	@if [ -f "Cargo.toml" ]; then cargo test; fi
	@if [ -f "requirements.txt" ] || [ -f "pyproject.toml" ]; then pytest; fi

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf dist build *.egg-info target/ bin/ safesock/libsafesocket/libsafesocket.dylib safesock/libsafesocket/libsafesocket.so safesock/libsafesocket/libsafesocket.h
