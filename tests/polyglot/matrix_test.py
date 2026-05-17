import subprocess
import time
import os
import signal
import sys

# Determine project root (2 levels up from this script)
ROOT_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))

def run_command(cmd, cwd=None, background=False, env=None):
    # Use ROOT_DIR if no cwd is provided
    target_cwd = cwd if cwd else ROOT_DIR
    print(f"Executing: {cmd} in {target_cwd}")
    
    full_env = os.environ.copy()
    if env:
        full_env.update(env)

    if background:
        return subprocess.Popen(cmd, shell=True, cwd=target_cwd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, preexec_fn=os.setsid, env=full_env)
    else:
        result = subprocess.run(cmd, shell=True, cwd=target_cwd, capture_output=True, text=True, env=full_env)
        return result

def main():
    # 1. Build everything
    print("--- Phase 1: Building ---")
    run_command("make build-lib")
    run_command("go build -o matrix_server cmd/test/matrix_server/main.go")
    
    has_cargo = run_command("cargo --version").returncode == 0
    has_gpp = run_command("g++ --version").returncode == 0

    if has_cargo:
        run_command("cargo build --example matrix_client", cwd=os.path.join(ROOT_DIR, "safesock/rust"))
    
    if has_gpp:
        run_command("make matrix_client", cwd=os.path.join(ROOT_DIR, "safesock/cpp/examples"))

    # 2. Start Matrix Server
    print("\n--- Phase 2: Starting Server ---")
    server_process = run_command("./matrix_server 127.0.0.1:9999", background=True)
    time.sleep(2)  # Wait for server to start

    results = []

    # 3. Run Python Client
    print("\n--- Phase 3: Python Client ---")
    py_result = run_command("python3 safesock/python/tests/matrix_client.py 127.0.0.1:9999")
    print(py_result.stdout)
    print(py_result.stderr)
    results.append(("Python", py_result.returncode == 0))

    # 4. Run Rust Client
    if has_cargo:
        print("\n--- Phase 4: Rust Client ---")
        rust_result = run_command("cargo run --example matrix_client -- 127.0.0.1:9999", cwd=os.path.join(ROOT_DIR, "safesock/rust"))
        print(rust_result.stdout)
        print(rust_result.stderr)
        results.append(("Rust", rust_result.returncode == 0))

    # 5. Run C++ Client
    if has_gpp:
        print("\n--- Phase 5: C++ Client ---")
        lib_dir = os.path.abspath(os.path.join(ROOT_DIR, "safesock/libsafesocket"))
        cpp_env = {
            "DYLD_LIBRARY_PATH": lib_dir,
            "LD_LIBRARY_PATH": lib_dir
        }
        cpp_result = run_command("./matrix_client 127.0.0.1:9999", cwd=os.path.join(ROOT_DIR, "safesock/cpp/examples"), env=cpp_env)
        print(cpp_result.stdout)
        print(cpp_result.stderr)
        results.append(("C++", cpp_result.returncode == 0))

    # 6. Cleanup
    print("\n--- Phase 6: Cleanup ---")
    try:
        os.killpg(os.getpgid(server_process.pid), signal.SIGTERM)
        run_command("rm matrix_server")
    except:
        pass
    
    # Summary
    print("\n--- INTEROPERABILITY MATRIX SUMMARY ---")
    all_success = True
    for lang, success in results:
        status = "PASSED" if success else "FAILED"
        print(f"{lang:10}: {status}")
        if not success:
            all_success = False
    
    if all_success:
        print("\nOVERALL STATUS: SUCCESS")
        sys.exit(0)
    else:
        print("\nOVERALL STATUS: FAILURE")
        sys.exit(1)

if __name__ == "__main__":
    main()
