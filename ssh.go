package sshgo

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// SSHClient ...
type SSHClient struct {
	Host    string
	User    string
	KeyPath string
	Client  *ssh.Client
}

// NewSSHClient ..
func NewSSHClient(host, user, keyPath string) *SSHClient {
	return &SSHClient{
		Host:    host + ":22",
		User:    user,
		KeyPath: keyPath,
	}
}

// Connect ...
func (c *SSHClient) Connect() error {
	sshConfig := &ssh.ClientConfig{
		User: c.User,
		Auth: []ssh.AuthMethod{publicKeyFile(c.KeyPath)},
	}
	sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	client, err := ssh.Dial("tcp", c.Host, sshConfig)
	if err != nil {
		return err
	}

	c.Client = client

	return nil
}

// ExecCommand ...
func (c *SSHClient) ExecCommand(cmd string) error {
	session, err := c.Client.NewSession() // Les sessions ne s'utilisent qu'une seule fois => recréer à chaque fois
	if err != nil {
		return err
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}
	err = session.Start(cmd)
	if err != nil {
		fmt.Println("error starting cmd:", err.Error())
		return err
	}

	go asyncReceive(stdout)
	go asyncReceive(stderr)

	err = session.Wait()
	if err != nil {
		fmt.Println("error waiting cmd:", err.Error())
		return err
	}

	return nil
}

// ExecBashFile ...
func (c *SSHClient) ExecBashFile(path string) error {
	// Faire attention au "" et '' dans le fichier. Erreur possible
	dat, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	return c.ExecCommand(string(dat))
}

// CopyFile ...
// localPath = chemin jusqu'au fichier source
// remotePath = chemin jusqu'au dossier de destination
func (c *SSHClient) CopyFile(localPath, remotePath string) error {
	fileName := filepath.Base(localPath)

	file, err := os.Open(filepath.FromSlash(localPath)) // fromSlash nécessaire pour windows
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	return c.WriteFile(reader, filepath.Clean(remotePath)+"/"+fileName)
}

// CopyFolder ...
func (c *SSHClient) CopyFolder(localPath, remotePath string) error {
	files, err := ioutil.ReadDir(localPath)
	if err != nil {
		return err
	}

	for _, f := range files {
		c.CopyFile(filepath.Clean(localPath)+"/"+f.Name(), remotePath)
	}

	return nil
}

// -----------------------------------
// DEPUIS mosolovsa/go_cat_sshfilerw
// -----------------------------------

// WriteFile : Write file to the remote server
//rfrom io.Reader - interface of an instance with content, that should be written
//rpath string - path to the file on the server
func (c *SSHClient) WriteFile(rfrom io.Reader, rpath string) error {
	//perform cat stdin to the file, after perform return value session will be closed,
	// so the cat will be performed
	err := c.perform(func(s *ssh.Session) error {
		if rfrom == nil {
			return errors.New("Reader to read file content is not provided")
		}
		sshstdinPipe, err := s.StdinPipe()
		if err != nil {
			return err
		}

		done := make(chan error)
		go func(done chan error) {
			err = s.Start(fmt.Sprintf("cat > %s", rpath))
			done <- err
		}(done)
		err = <-done
		if err != nil {
			return err
		}

		_, err = io.Copy(sshstdinPipe, rfrom)
		if err != nil {
			return err
		}
		//cat waits for the newline symbol from stdin to perform writing
		_, err = sshstdinPipe.Write([]byte("\n"))
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	//cat will perform write on receiving of '\n' byte, and write it to the file
	//here we cut off that last byte
	err = c.perform(func(s *ssh.Session) error {
		return s.Run(fmt.Sprintf("truncate --size=-1 %s", rpath))
	})
	return err

}

type operation func(s *ssh.Session) error

//Helper function, responsible for openning and closing ssh session. Session performs single command.
//As an argument must be passed a function, that should be performed
func (c *SSHClient) perform(op operation) error {
	s, err := c.Client.NewSession()
	if err != nil {
		return err
	}
	defer s.Close()

	err = op(s)

	return err
}
