# Research: Low-Latency Cross-Language Execution via Shared Memory

## Goal
Enable a "Provider" process (e.g., Server) to write data to Shared Memory and **instantly trigger** code execution in a "Consumer" process (e.g., Client), minimizing jitter and OS context switching overhead. The focus is on interoperability between languages (Go, C, C++, Python, ASM, Rust).

## Research Avenues

### 1. Function Pointer Exchange (C / Cgo / ASM)
**Concept**: Store the memory address of a callback function (residing in the Consumer execution space) directly in Shared Memory. The Provider reads this address and "calls" it.
**Mechanism**:
- Use `Cgo` (Go) or `ctypes` (Python) to expose a C-compatible function pointer.
- Use Assembly (`JMP` or `CALL`) to jump to that address.
**Critical Challenges**:
- **Process Isolation**: Modern OSs (Windows/Linux) enforce Virtual Memory. Address `0x00F` in Process A does NOT map to Process B. A direct jump will cause an immediate **Segmentation Fault (Access Violation)**.
- **ASLR (Address Space Layout Randomization)**: Memory locations are randomized at startup. Addresses are not deterministic.
- **NX Bit (No-Execute)**: Data segments (like SHM) are often marked as non-executable to prevent malware.

### 2. Header-less "Hot" Polling via ASM
**Concept**: Instead of OS signals (slow), Consumer spin-loops in Assembly on a specific cache-line in SHM.
- **Optimization**: Use `PAUSE` or `MONITOR`/`MWAIT` instructions (x86) to reduce CPU load while waiting for a single bit flip.
- **Benefit**: "Zero-syscall" latency.
- **Downside**: Burns 1 CPU core at 100%.

### 3. Shared Library / Native Interface
**Concept**: Use a specialized C-shared library (`.dll`/`.so`) loaded by both processes to bridge the runtime gap.
- The "Trigger" is a native function exposed by the library.
- **Go Support**: `import "C"` allows Go to pass pointers to C, but passing Go function pointers *back* to C code that runs in another thread/context requires careful `cgo.Handle` management to avoid GC panics.

## Summary of Encountered Limitations
1.  **Safety vs Speed**: True "Direct Execution" (Process A running code in Process B) is functionally malware behavior and blocked by the OS kernel.
2.  **Runtime incompatibility**: A Go function expects the Go Runtime (Garbage Collector, Stack check). Calling it from a raw C pointer without setting up the Go environment will crash the process.
3.  **Synchronization**: Without OS primitives (Mutex/Semaphores), managing race conditions requires precise Atomic CPU instructions.