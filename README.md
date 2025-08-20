# SFTP Test Server

A lightweight, local SFTP server implementation in Go for testing, development, and integration purposes. This package provides an easy way to spin up an SFTP server in your test environment without needing external dependencies or complex setup.

## 🎯 Purpose

This tool is designed for developers who need to test SFTP functionality in their applications without setting up a full SFTP server infrastructure. Perfect for:

- **Unit and integration testing** of SFTP client code
- **Development environments** where you need a quick SFTP server
- **CI/CD pipelines** that require SFTP testing
- **Learning and experimentation** with SFTP protocols
- **Isolated testing** without external dependencies

## ✨ Features

- **Zero-configuration setup** - starts with minimal parameters
- **In-memory SSH key generation** - no need to manage certificates
- **Configurable authentication** - set custom username/password
- **Custom root directory** - sandbox file operations to specific directories
- **Thread-safe operations** - safe for concurrent connections
- **Full SFTP support** - file upload, download, directory operations
- **Easy cleanup** - stop server and clean up resources

## 🚀 Quick Start

### Installation

```bash
go get github.com/JuniorGuerra/sftp_test_server
```

### Basic Usage

```go
package main

import (
    "log"
    sftptest "github.com/JuniorGuerra/sftp_test_server"
)

func main() {
    // Create a new SFTP server
    server, err := sftptest.NewSFTPServerLocal("testuser", "testpass", 2222, "./sftp_root")
    if err != nil {
        log.Fatal(err)
    }

    // Start the server
    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
    defer server.Stop()

    // Server is now running on localhost:2222
    // Username: testuser, Password: testpass
    // Files will be stored in ./sftp_root directory
    
    log.Println("SFTP server is running on port 2222")
    // Your application logic here
}
```

## 📁 Project Structure

```
.
├── sftp_test_server.go    # Main server implementation
├── sftp_test.go           # Comprehensive test examples
├── examples/              # Usage examples
│   ├── create_file/       # File upload example
│   └── get_file/          # File download example
├── test_sftp_data/        # Test data directory
└── go.mod                 # Go module definition
```

## 🔧 API Reference

### Core Functions

#### `NewSFTPServerLocal(user, password string, port int, rootDir string) (*SFTPServer, error)`
Creates a new SFTP server instance.

**Parameters:**
- `user`: Username for authentication
- `password`: Password for authentication  
- `port`: Port number to listen on
- `rootDir`: Root directory for file operations (created if doesn't exist)

#### `Start() error`
Starts the SFTP server on the configured port.

#### `Stop() error`
Stops the SFTP server and cleans up resources.

#### `GetRootDir() string`
Returns the server's root directory path.

#### `GetPort() int`
Returns the server's port number.

#### `IsRunning() bool`
Returns whether the server is currently running.

## 📋 Usage Examples

### 1. Testing File Operations

```go
func TestSFTPFileOperations(t *testing.T) {
    // Setup
    server, _ := sftptest.NewSFTPServerLocal("user", "pass", 2222, "./test_data")
    server.Start()
    defer server.Stop()
    
    // Connect and test
    client := connectToSFTP("user", "pass", "localhost:2222")
    defer client.Close()
    
    // Upload file
    file, _ := client.Create("test.txt")
    file.Write([]byte("Hello World"))
    file.Close()
    
    // Download file
    downloadFile, _ := client.Open("test.txt")
    content, _ := io.ReadAll(downloadFile)
    // Assert content equals "Hello World"
}
```

### 2. Integration Testing

```go
func TestMyAppWithSFTP(t *testing.T) {
    // Start test server
    server, _ := sftptest.NewSFTPServerLocal("testuser", "testpass", 2223, "./integration_data")
    server.Start()
    defer func() {
        server.Stop()
        os.RemoveAll("./integration_data")
    }()
    
    // Test your application that uses SFTP
    myApp := &MyApplication{
        SFTPHost: "localhost:2223",
        Username: "testuser",
        Password: "testpass",
    }
    
    result := myApp.ProcessFiles()
    // Assert expected behavior
}
```

### 3. Concurrent Testing

```go
func TestConcurrentSFTPOperations(t *testing.T) {
    server, _ := sftptest.NewSFTPServerLocal("user", "pass", 2224, "./concurrent_test")
    server.Start()
    defer server.Stop()
    
    var wg sync.WaitGroup
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            // Each goroutine performs SFTP operations
            client := connectToSFTP("user", "pass", "localhost:2224")
            defer client.Close()
            
            filename := fmt.Sprintf("file_%d.txt", id)
            file, _ := client.Create(filename)
            file.Write([]byte(fmt.Sprintf("Content from goroutine %d", id)))
            file.Close()
        }(i)
    }
    wg.Wait()
}
```

## 🏗️ Architecture

The server implements a complete SFTP stack:

1. **SSH Layer**: Handles SSH connections and authentication
2. **SFTP Protocol**: Implements SFTP subsystem for file operations
3. **File System**: Maps SFTP operations to local file system
4. **Security**: Generates temporary RSA keys for secure connections

### Key Components

- **SFTPServer**: Main server struct managing connections and state
- **customHandlers**: Implements SFTP file operations (read, write, list, commands)
- **SSH Configuration**: Handles authentication and encryption
- **Connection Management**: Thread-safe handling of multiple clients

## 🔒 Security Considerations

⚠️ **Important**: This server is designed for testing purposes only.

- Uses `ssh.InsecureIgnoreHostKey()` for host key verification
- Generates temporary RSA keys (not persistent)
- No rate limiting or connection throttling
- Simple password authentication only
- **Do not use in production environments**

## 🧪 Testing

Run the comprehensive test suite:

```bash
# Run all tests
go test -v

# Run with race detection
go test -race -v

# Run benchmarks
go test -bench=. -v
```

The test suite includes:
- File upload/download operations
- Directory creation and listing
- Nested directory structures
- Concurrent access testing
- Performance benchmarks

## 🛠️ Development

### Prerequisites
- Go 1.24.2 or later
- Dependencies managed via Go modules

### Building
```bash
go build ./...
```

### Running Examples
```bash
# File upload example
cd examples/create_file
go run main.go

# File download example  
cd examples/get_file
go run main.go
```

## 📖 Use Cases

### 1. **Unit Testing SFTP Clients**
Test your SFTP client code without external dependencies:
```go
func TestMyClient(t *testing.T) {
    server := setupTestServer()
    defer server.Stop()
    
    client := NewMyClient("localhost:2222", "user", "pass")
    err := client.UploadFile("local.txt", "remote.txt")
    assert.NoError(t, err)
}
```

### 2. **CI/CD Pipeline Testing**
Include in your CI pipeline for automated testing:
```yaml
# .github/workflows/test.yml
- name: Test SFTP Integration
  run: |
    go test ./tests/sftp_integration_test.go -v
```

### 3. **Development Environment**
Quick SFTP server for local development:
```bash
go run -c "server := sftptest.NewSFTPServerLocal(...); server.Start(); select{}"
```

### 4. **API Testing**
Test REST APIs that interact with SFTP:
```go
func TestFileUploadAPI(t *testing.T) {
    sftpServer := setupTestSFTP()
    defer sftpServer.Stop()
    
    response := httptest.NewRequest("POST", "/upload", fileData)
    // Test your API endpoint that uploads to SFTP
}
```

## 🤝 Contributing

Contributions are welcome! This project aims to provide a robust testing tool for the Go community.

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## 📄 License

This project is licensed under the GNU License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built on top of [github.com/pkg/sftp](https://github.com/pkg/sftp)
- Uses [golang.org/x/crypto/ssh](https://golang.org/x/crypto/ssh) for SSH implementation
- Inspired by the need for simple SFTP testing in Go projects

---

**Ready to test your SFTP code?** Get started with `go get github.com/JuniorGuerra/sftp_test_server` and spin up your test server in seconds! 🚀