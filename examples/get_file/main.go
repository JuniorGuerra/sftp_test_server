package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	// import the sftp_test_server module
	sftp_test_server "github.com/JuniorGuerra/sftp_test_server"
)

// this guide is getting from https://docs.couchdrop.io/walkthroughs/using-sftp-clients/using-sftp-with-golang
func main() {
	// SFTP connection parameters
	host := "localhost"
	port := 2222
	user := "fake_username"
	password := "fake_password"

	server, err := sftp_test_server.NewSFTPServerLocal(user, password, port, "./temp")

	if err != nil {
		fmt.Println("Failed to create local SFTP server: ", err)
		return
	}

	if err := server.Start(); err != nil {
		fmt.Println("Failed to start SFTP server: ", err)
		return
	}
	defer server.Stop()

	// Create SSH client configuration
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	// Connect to the SSH server
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		fmt.Println("Failed to connect to SSH server:", err)
		return
	}
	defer conn.Close()

	// Open SFTP session
	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		fmt.Println("Failed to open SFTP session:", err)
		return
	}

	createFile(sftpClient)

	defer sftpClient.Close()

	// Now you can perform SFTP operations using the sftpClient
}

func createFile(client *sftp.Client) {
	remoteFile, err := client.Open("file_to_download.txt")
	if err != nil {
		fmt.Println("Failed to open remote file:", err)
		return
	}
	defer remoteFile.Close()

	localFile, err := os.Create("./file.txt")
	if err != nil {
		fmt.Println("Failed to create local file:", err)
		return
	}
	defer localFile.Close()

	_, err = io.Copy(localFile, remoteFile)
	if err != nil {
		fmt.Println("Failed to download file:", err)
		return
	}

}
