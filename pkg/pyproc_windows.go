//go:build windows
// +build windows

package pkg

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func setSignalsForChannel(c chan os.Signal) {
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
}

func waitForExit(cmd *exec.Cmd) error {
	err := cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ProcessState.ExitCode() == -1 {
				// The child process was killed
				return errors.New("child process was killed")
			}
		}
		return err
	}
	return nil
}
