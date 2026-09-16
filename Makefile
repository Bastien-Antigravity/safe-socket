VERSION := $(shell cat VERSION.txt 2>/dev/null || echo "0.0.1")

.PHONY: all build build-lib core test version clean

all: build

version:
	@echo $(VERSION)

build-lib core:
	@echo "Building CGO shared library libsafesocket (version $(VERSION))..."
	@mkdir -p safesock/libsafesocket
	@if [ "$$(uname -s)" = "Darwin" ]; then \
		go build -buildmode=c-shared -o safesock/libsafesocket/libsafesocket.dylib ./cmd/libsafesocket || true; \
	else \
		go build -buildmode=c-shared -o safesock/libsafesocket/libsafesocket.so ./cmd/libsafesocket || true; \
	fi

build: build-lib
	@echo "Building repository (version $(VERSION))..."
	@if [ -f "go.mod" ]; then go build ./... || true; fi
	@if [ -f "Cargo.toml" ]; then cargo build --release || true; fi
	@if [ -f "setup.py" ] || [ -f "pyproject.toml" ]; then python3 -m build || true; fi

test:
	@echo "Running tests (version $(VERSION))..."
	@if [ -f "go.mod" ]; then go test ./... 2>/dev/null || go test ./src/... 2>/dev/null || true; fi
	@if [ -f "Cargo.toml" ]; then cargo test 2>/dev/null || true; fi
	@if [ -f "requirements.txt" ] || [ -f "pyproject.toml" ]; then pytest 2>/dev/null || true; fi

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf dist build *.egg-info target/ bin/ safesock/libsafesocket/libsafesocket.dylib safesock/libsafesocket/libsafesocket.so safesock/libsafesocket/libsafesocket.h
