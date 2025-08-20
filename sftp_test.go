package sftptest

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Example of using the SFTP server in tests
func TestSFTPOperations(t *testing.T) {
	// Setup SFTP server
	server, err := NewSFTPServerLocal("testuser", "testpass", 2222, "./test_sftp_data")
	if err != nil {
		t.Fatalf("Failed to create SFTP server: %v", err)
	}

	// Start the server
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start SFTP server: %v", err)
	}
	defer server.Stop()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Connect to the SFTP server
	client, err := connectSFTP("testuser", "testpass", "localhost:2222")
	if err != nil {
		t.Fatalf("Failed to connect to SFTP server: %v", err)
	}
	defer client.Close()

	// Test file upload
	t.Run("Upload File", func(t *testing.T) {
		testData := []byte("Hello, SFTP Test!")
		remotePath := "test_upload.txt"

		// Create remote file
		remoteFile, err := client.Create(remotePath)
		if err != nil {
			t.Fatalf("Failed to create remote file: %v", err)
		}
		defer remoteFile.Close()

		// Write data
		_, err = remoteFile.Write(testData)
		if err != nil {
			t.Fatalf("Failed to write to remote file: %v", err)
		}

		// Verify the file exists on the server
		localPath := filepath.Join(server.GetRootDir(), remotePath)
		content, err := os.ReadFile(localPath)
		if err != nil {
			t.Fatalf("Failed to read uploaded file from server: %v", err)
		}

		if !bytes.Equal(content, testData) {
			t.Errorf("File content mismatch. Expected: %s, Got: %s", testData, content)
		}
	})

	// Test file download
	t.Run("Download File", func(t *testing.T) {
		// First, create a file on the server
		testData := []byte("Download test content")
		localPath := filepath.Join(server.GetRootDir(), "test_download.txt")
		if err := os.WriteFile(localPath, testData, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Download the file via SFTP
		remoteFile, err := client.Open("test_download.txt")
		if err != nil {
			t.Fatalf("Failed to open remote file: %v", err)
		}
		defer remoteFile.Close()

		// Read the content
		buffer := make([]byte, len(testData))
		n, err := remoteFile.Read(buffer)
		if err != nil && err != io.EOF {
			t.Fatalf("Failed to read remote file: %v", err)
		}

		if !bytes.Equal(buffer[:n], testData) {
			t.Errorf("Downloaded content mismatch. Expected: %s, Got: %s", testData, buffer[:n])
		}
	})

	// Test directory operations
	t.Run("Directory Operations", func(t *testing.T) {
		// Create a directory
		dirName := "test_dir"
		if err := client.Mkdir(dirName); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Verify directory exists on server
		localDirPath := filepath.Join(server.GetRootDir(), dirName)
		info, err := os.Stat(localDirPath)
		if err != nil {
			t.Fatalf("Directory not created on server: %v", err)
		}
		if !info.IsDir() {
			t.Error("Created path is not a directory")
		}

		// List directory contents
		entries, err := client.ReadDir(".")
		if err != nil {
			t.Fatalf("Failed to list directory: %v", err)
		}

		found := false
		for _, entry := range entries {
			if entry.Name() == dirName && entry.IsDir() {
				found = true
				break
			}
		}
		if !found {
			t.Error("Created directory not found in listing")
		}

		// Remove directory
		if err := client.RemoveDirectory(dirName); err != nil {
			t.Fatalf("Failed to remove directory: %v", err)
		}
	})

	// Test file operations in subdirectories
	t.Run("Subdirectory File Operations", func(t *testing.T) {
		// Create nested directory structure
		nestedPath := "level1/level2"
		if err := client.MkdirAll(nestedPath); err != nil {
			t.Fatalf("Failed to create nested directories: %v", err)
		}

		// Upload file to nested directory
		testData := []byte("Nested file content")
		nestedFilePath := filepath.Join(nestedPath, "nested_file.txt")

		remoteFile, err := client.Create(nestedFilePath)
		if err != nil {
			t.Fatalf("Failed to create nested file: %v", err)
		}

		_, err = remoteFile.Write(testData)
		remoteFile.Close()
		if err != nil {
			t.Fatalf("Failed to write nested file: %v", err)
		}

		// Verify on server
		localFilePath := filepath.Join(server.GetRootDir(), nestedFilePath)
		content, err := os.ReadFile(localFilePath)
		if err != nil {
			t.Fatalf("Failed to read nested file from server: %v", err)
		}

		if !bytes.Equal(content, testData) {
			t.Errorf("Nested file content mismatch")
		}
	})
}

// Helper function to connect to SFTP server
func connectSFTP(user, password, addr string) (*sftp.Client, error) {
	// SSH client configuration
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // Only for testing!
		Timeout:         5 * time.Second,
	}

	// Connect to SSH
	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH: %w", err)
	}

	// Create SFTP client
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}

	return sftpClient, nil
}

// Example of using in integration tests
func TestIntegrationWithSFTP(t *testing.T) {
	// Start SFTP server for the entire test suite
	server, err := NewSFTPServerLocal("integrationuser", "integrationpass", 2223, "./integration_test_data")
	if err != nil {
		t.Fatalf("Failed to create SFTP server: %v", err)
	}

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start SFTP server: %v", err)
	}
	defer func() {
		server.Stop()
		// Clean up test data
		os.RemoveAll("./integration_test_data")
	}()

	// Your integration test logic here
	// The SFTP server is available at localhost:2223
	// Files are stored in ./integration_test_data
}

// Benchmark example
func BenchmarkSFTPUpload(b *testing.B) {
	server, _ := NewSFTPServerLocal("benchuser", "benchpass", 2224, "./bench_data")
	server.Start()
	defer server.Stop()
	defer os.RemoveAll("./bench_data")

	time.Sleep(100 * time.Millisecond)

	client, err := connectSFTP("benchuser", "benchpass", "localhost:2224")
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	testData := make([]byte, 1024) // 1KB test data
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	for i := 0; b.Loop(); i++ {
		fileName := fmt.Sprintf("bench_%d.txt", i)
		file, err := client.Create(fileName)
		if err != nil {
			b.Fatalf("Failed to create file: %v", err)
		}
		file.Write(testData)
		file.Close()
	}
}
