// Package sftptest provides a local SFTP server for testing purposes
package sftptest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SFTPServer represents a local SFTP server instance
type SFTPServer struct {
	user       string
	password   string
	port       int
	rootDir    string
	listener   net.Listener
	config     *ssh.ServerConfig
	running    bool
	mu         sync.Mutex
	privateKey ssh.Signer
	stopChan   chan struct{}
}

// NewSFTPServerLocal creates a new local SFTP server instance
func NewSFTPServerLocal(user, password string, port int, rootDir string) (*SFTPServer, error) {
	// Create root directory if it doesn't exist
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create root directory: %w", err)
	}

	server := &SFTPServer{
		user:     user,
		password: password,
		port:     port,
		rootDir:  rootDir,
		stopChan: make(chan struct{}),
	}

	// Generate a temporary private key for the server
	privateKey, err := generatePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}
	server.privateKey = privateKey

	// Configure SSH server
	server.config = &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == server.user && string(pass) == server.password {
				return nil, nil
			}
			return nil, fmt.Errorf("authentication failed")
		},
	}
	server.config.AddHostKey(server.privateKey)

	return server, nil
}

// Start starts the SFTP server
func (s *SFTPServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server is already running")
	}

	// Start listening on the specified port
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}
	s.listener = listener
	s.running = true

	// Handle connections in a goroutine
	go s.acceptConnections()

	log.Printf("SFTP server started on port %d", s.port)
	return nil
}

// Stop stops the SFTP server
func (s *SFTPServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("server is not running")
	}

	close(s.stopChan)
	s.running = false

	if s.listener != nil {
		return s.listener.Close()
	}

	log.Println("SFTP server stopped")
	return nil
}

// GetRootDir returns the root directory path
func (s *SFTPServer) GetRootDir() string {
	return s.rootDir
}

// GetPort returns the server port
func (s *SFTPServer) GetPort() int {
	return s.port
}

// IsRunning returns whether the server is running
func (s *SFTPServer) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// acceptConnections handles incoming connections
func (s *SFTPServer) acceptConnections() {
	for {
		select {
		case <-s.stopChan:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				if !s.IsRunning() {
					return
				}
				log.Printf("Failed to accept connection: %v", err)
				continue
			}

			go s.handleConnection(conn)
		}
	}
}

// handleConnection handles a single SSH/SFTP connection
func (s *SFTPServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, s.config)
	if err != nil {
		log.Printf("Failed to establish SSH connection: %v", err)
		return
	}
	defer sshConn.Close()

	log.Printf("SSH connection established with %s", sshConn.RemoteAddr())

	// Discard all global requests
	go ssh.DiscardRequests(reqs)

	// Handle channels
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Failed to accept channel: %v", err)
			continue
		}

		go s.handleChannel(channel, requests)
	}
}

// handleChannel handles an SSH channel for SFTP
func (s *SFTPServer) handleChannel(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()

	// Handle channel requests
	go func() {
		for req := range requests {
			switch req.Type {
			case "subsystem":
				if string(req.Payload[4:]) == "sftp" {
					req.Reply(true, nil)
				} else {
					req.Reply(false, nil)
				}
			default:
				req.Reply(false, nil)
			}
		}
	}()

	// Create SFTP server handler with custom root
	handlers := &customHandlers{rootDir: s.rootDir}
	server := sftp.NewRequestServer(channel, sftp.Handlers{
		FileGet:  handlers,
		FilePut:  handlers,
		FileCmd:  handlers,
		FileList: handlers,
	})
	if err := server.Serve(); err != nil && err != io.EOF {
		log.Printf("SFTP server error: %v", err)
	}
}

// customHandlers implements sftp.Handlers with a custom root directory
type customHandlers struct {
	rootDir string
}

func (h *customHandlers) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	path := filepath.Join(h.rootDir, r.Filepath)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (h *customHandlers) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	path := filepath.Join(h.rootDir, r.Filepath)

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (h *customHandlers) Filecmd(r *sftp.Request) error {
	path := filepath.Join(h.rootDir, r.Filepath)

	switch r.Method {
	case "Remove":
		return os.Remove(path)
	case "Rename":
		newPath := filepath.Join(h.rootDir, r.Target)
		return os.Rename(path, newPath)
	case "Mkdir":
		return os.Mkdir(path, 0755)
	case "Rmdir":
		return os.Remove(path)
	case "Setstat":
		return nil // Simplified: ignore attribute changes
	default:
		return fmt.Errorf("unsupported file command: %s", r.Method)
	}
}

func (h *customHandlers) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	path := filepath.Join(h.rootDir, r.Filepath)

	switch r.Method {
	case "List":
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}

		var fileInfos []os.FileInfo
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			fileInfos = append(fileInfos, info)
		}
		return listerat(fileInfos), nil

	case "Stat":
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		return listerat([]os.FileInfo{info}), nil

	default:
		return nil, fmt.Errorf("unsupported list command: %s", r.Method)
	}
}

// listerat implements sftp.ListerAt for a slice of os.FileInfo
type listerat []os.FileInfo

func (l listerat) ListAt(ls []os.FileInfo, offset int64) (int, error) {
	if offset >= int64(len(l)) {
		return 0, io.EOF
	}

	n := copy(ls, l[offset:])
	if n < len(ls) {
		return n, io.EOF
	}
	return n, nil
}

// generatePrivateKey generates a temporary RSA private key for testing
func generatePrivateKey() (ssh.Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	privatePEM := pem.EncodeToMemory(privateKeyPEM)
	return ssh.ParsePrivateKey(privatePEM)
}
