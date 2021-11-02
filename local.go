package sshgo

import (
	"fmt"
	"os/exec"
	"runtime"
)

// LocalClient ...
type LocalClient struct{}

// ExecCommand ...
func (c *LocalClient) ExecCommand(cmd string) error {
	var commande *exec.Cmd
	if runtime.GOOS == "windows" {
		commande = exec.Command("cmd", "/C", cmd)
	} else {
		commande = exec.Command("bash", "-c", cmd)
	}

	// ret, err := commande.CombinedOutput()
	// return string(ret), err

	return c.runCommand(commande)
}

// ExecBashFile ...
func (c *LocalClient) ExecBashFile(path string) error {
	commande := exec.Command("bash", path)

	return c.runCommand(commande)
}

// RunCommand = local
func (c *LocalClient) runCommand(cmd *exec.Cmd) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	err = cmd.Start()
	if err != nil {
		fmt.Println("error starting cmd:", err.Error())
		return err
	}

	go asyncReceive(stdout)
	go asyncReceive(stderr)

	err = cmd.Wait()
	if err != nil {
		fmt.Println("error waiting cmd:", err.Error())
		return err
	}

	return nil
}
