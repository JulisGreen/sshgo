package sshgo

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// GetCurrentDirectory ...
func GetCurrentDirectory() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", err
	}
	exPath := filepath.Dir(ex)

	return exPath, nil
}

func publicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		panic(err)
	}
	return ssh.PublicKeys(key)
}

func asyncReceive(reader io.Reader) {
	// To be run async
	scanner := bufio.NewScanner(reader)
	// scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		m := scanner.Text()
		fmt.Println(m)
	}
}
