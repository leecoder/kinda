package pkg

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// PythonProcess represents a running Python process with its I/O pipes
type PythonProcess struct {
	Cmd    *exec.Cmd
	Stdin  io.WriteCloser
	Stdout io.ReadCloser
	Stderr io.ReadCloser
}

// NewPythonProcessFromString starts a Python script from a string with the given arguments.
// It returns a PythonProcess struct containing the command and I/O pipes.
// It ensures that the child process is killed if the parent process is killed.
func (env *Environment) NewPythonProcessFromString(script string, args ...string) (*PythonProcess, error) {
	// Create the command
	fullArgs := append([]string{"-"}, args...)
	cmd := exec.Command(env.PythonPath, fullArgs...)

	// Create pipes for the input, output, and error of the script
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// Write the script to stdin
	go func() {
		defer stdinPipe.Close()
		io.WriteString(stdinPipe, script)
	}()

	pyProcess := &PythonProcess{
		Cmd:    cmd,
		Stdin:  stdinPipe,
		Stdout: stdoutPipe,
		Stderr: stderrPipe,
	}

	// Set up signal handling
	setupSignalHandler(pyProcess)

	return pyProcess, nil
}

// Wait waits for the Python process to exit and returns an error if it was killed
func (pp *PythonProcess) Wait() error {
	err := pp.Cmd.Wait()
	if err != nil {
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

// Terminate gracefully stops the Python process
func (pp *PythonProcess) Terminate() error {
	if pp.Cmd.Process == nil {
		return nil // Process hasn't started or has already finished
	}

	// Try to terminate gracefully first
	err := pp.Cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		return err
	}

	// Wait for the process to exit
	done := make(chan error, 1)
	go func() {
		done <- pp.Cmd.Wait()
	}()

	// Wait for the process to exit or force kill after timeout
	select {
	case <-time.After(5 * time.Second):
		// Force kill if it doesn't exit within 5 seconds
		err = pp.Cmd.Process.Kill()
		if err != nil {
			return err
		}
		<-done // Wait for the process to be killed
	case err = <-done:
		// Process exited before timeout
	}

	return err
}

func setupSignalHandler(pp *PythonProcess) {
	signalChan := make(chan os.Signal, 1)
	setSignalsForChannel(signalChan)

	go func() {
		<-signalChan
		// Terminate the child process when a signal is received
		pp.Terminate()
	}()
}
