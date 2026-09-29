# Netcat Exec: Direct Connection vs `io.Pipe`

This document explains the differences, trade-offs, and underlying mechanics between using `io.Pipe()` and assigning
`net.Conn` directly to `cmd.Stdout` / `cmd.Stdin` when creating a Netcat-style execution shell in Go.

---

## 1. Overview

In Go, when creating a TCP command execution listener (like `netcat -e /bin/sh`), standard input and output streams of
the executed process (`exec.Cmd`) need to be wired to the TCP socket (`net.Conn`).

---

## 2. Approach 1: Using `io.Pipe()`

```go
func handle(conn net.Conn) {
cmd := exec.Command("/bin/sh", "-i")
rp, wp := io.Pipe()

cmd.Stdin = conn
cmd.Stdout = wp

go io.Copy(conn, rp)
cmd.Run()
conn.Close()
}
```

### How it Works

1. `io.Pipe()` returns a synchronous reader (`rp`) and writer (`wp`).
2. `cmd.Stdout` is assigned to `wp`. Output produced by the process is written to `wp`.
3. `go io.Copy(conn, rp)` runs in a separate background goroutine, reading bytes from `rp` and writing them to the
   network connection `conn`.

---

## 3. Approach 2: Direct Assignment (`cmd.Stdout = conn`)

```go
func handle(conn net.Conn) {
cmd := exec.Command("/bin/sh", "-i")
cmd.Stdin = conn
cmd.Stdout = conn
cmd.Stderr = conn // optional: also route stderr to conn
cmd.Run()
conn.Close()
}
```

### How it Works

- `net.Conn` implements the `io.Writer` interface via its `Write([]byte)` method.
- Because `cmd.Stdout` expects an `io.Writer`, `conn` can be directly assigned to `cmd.Stdout` and `cmd.Stderr`.

---

## 4. What Happens Under the Hood?

When you pass a non-file `io.Writer` (such as `net.Conn`) to `cmd.Stdout`, Go's `os/exec` standard library package
internally creates an OS pipe and launches an unexported background goroutine to copy data from the process's stdout
file descriptor to your `io.Writer`.

Therefore, direct assignment `cmd.Stdout = conn` delegates the pipe creation and copying logic directly to `os/exec`.

### Why Did the Book Use io.Pipe()?

1. Educational / Instructional Purposes: Demonstrating how Go's io.Pipe(), io.Reader, and io.Writer primitives work together across goroutines.
2. Stream Interception / Transformation: Using io.Pipe() creates an explicit stream bridge where you can easily insert intermediate processing (e.g., logging, encryption, filtering, or tee-ing data) before sending
   it across the network connection.


---

## 5. Summary & Key Differences

| Feature / Aspect                | `cmd.Stdout = wp` + `io.Pipe()`                                          | Direct `cmd.Stdout = conn`                      |
|:--------------------------------|:-------------------------------------------------------------------------|:------------------------------------------------|
| **Code Complexity**             | Requires manual goroutine & `io.Pipe()` management                       | Cleaner, shorter, and idiomatic                 |
| **Standard Library Delegation** | Manual stream copying via `io.Copy(conn, rp)`                            | `os/exec` handles piping and copying internally |
| **Custom Stream Processing**    | **Ideal**: Allows inspecting, filtering, or modifying streams mid-flight | Direct stream pass-through                      |
| **Educational Value**           | Explicitly demonstrates Go stream piping & concurrency                   | Hides stream piping details behind `os/exec`    |
