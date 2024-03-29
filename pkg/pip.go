package pkg

import (
	"fmt"
	"os"
	"os/exec"
)

func (env *Environment) PipInstallPackages(packages []string, index_url string, extra_index_url string, no_cache bool) error {
	args := []string{
		"install",
		"--no-warn-script-location",
	}

	if no_cache {
		args = append(args, "--no-cache-dir")
	}

	args = append(args, packages...)
	if index_url != "" {
		args = append(args, "--index-url", index_url)
	}
	if extra_index_url != "" {
		args = append(args, "--extra-index-url", extra_index_url)
	}

	installCmd := exec.Command(env.PipPath, args...)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("error installing package: %v", err)
	}
	return nil
}

func (env *Environment) PipInstallRequirmements(requirementsPath string) error {
	installCmd := exec.Command(env.PipPath, "install", "--no-warn-script-location", "-r", requirementsPath)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("error installing requirements: %v", err)
	}
	return nil
}

func (env *Environment) PipInstallPackage(packageToInstall string, index_url string, extra_index_url string, no_cache bool) error {
	packages := []string{
		packageToInstall,
	}
	return env.PipInstallPackages(packages, index_url, extra_index_url, no_cache)
}
