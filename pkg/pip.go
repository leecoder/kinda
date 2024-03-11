package pkg

import (
	"fmt"
	"os"
	"os/exec"
)

func (env *Environment) PipInstallRequirmements(requirementsPath string) error {
	installCmd := exec.Command(env.PipPath, "install", "-r", requirementsPath)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("error installing requirements: %v", err)
	}
	return nil
}

func (env *Environment) PipInstallPackage(packageToInstall string) error {
	installCmd := exec.Command(env.PipPath, "install", packageToInstall)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("error installing package: %v", err)
	}
	return nil
}
