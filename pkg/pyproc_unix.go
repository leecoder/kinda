//go:build !windows
// +build !windows

package pkg

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// BoundRunPythonScriptFromFile runs a Python script from a file with the given arguments.
// It ensures that the child process is killed if the parent process is killed.
func (env *Environment) BoundRunPythonScriptFromFile(scriptPath string, args ...string) error {
	// Create the command
	// put scriptPath at the front of the args
	args = append([]string{scriptPath}, args...)
	cmd := exec.Command(env.PythonPath, args...)

	// Create a pipe for the output of the script
	stdoutPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return err
	}

	// Create a channel to receive signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Wait for the command to finish or a signal to be received
	go func() {
		<-signalChan
		// Kill the child process when a signal is received
		cmd.Process.Kill()
	}()

	// Read from the command's stdout
	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		fmt.Println("Python script output:", scanner.Text())
	}

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == -1 {
				// The child process was killed
				return errors.New("child process was killed")
			}
		}
		return err
	}

	return nil
}
