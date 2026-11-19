VERSION := $(strip $(shell cat VERSION.txt))
$(if $(VERSION),,$(error VERSION.txt is missing or empty))

.PHONY: all build build-lib core test version clean

all: build

version:
	@echo $(VERSION)

build-lib core:
	@echo "Building CGO shared library libsafesocket (version $(VERSION))..."
	@mkdir -p safesock/libsafesocket
	@OS=$$(uname -s); \
	case "$$OS" in \
		Darwin) \
			go build -buildmode=c-shared -o safesock/libsafesocket/libsafesocket.dylib ./cmd/libsafesocket; \
			install_name_tool -id @rpath/libsafesocket.dylib safesock/libsafesocket/libsafesocket.dylib 2>/dev/null || true ;; \
		MINGW*|MSYS*|CYGWIN*|Windows_NT) \
			go build -buildmode=c-shared -o safesock/libsafesocket/libsafesocket.dll ./cmd/libsafesocket ;; \
		*) \
			go build -buildmode=c-shared -o safesock/libsafesocket/libsafesocket.so ./cmd/libsafesocket ;; \
	esac

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
