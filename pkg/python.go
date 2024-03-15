package pkg

import (
	"bufio"
	"fmt"
	"os/exec"
)

func RunReadStdout(binPath string, args ...string) (string, error) {
	retv := ""
	cmd := exec.Command(binPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	defer stdout.Close()

	// continue to read the output until there is no more
	// or an error occurs
	if err := cmd.Start(); err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		retv += scanner.Text() + "\n"
	}
	return retv, nil
}

func (env *Environment) RunPythonReadCombined(scriptPath string, args ...string) (string, error) {
	args = append([]string{scriptPath}, args...)
	cmd := exec.Command(env.PythonPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}

func (env *Environment) RunPythonReadStdout(scriptPath string, args ...string) (string, error) {
	// put scriptPath at the front of the args
	retv := ""
	args = append([]string{scriptPath}, args...)
	cmd := exec.Command(env.PythonPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	defer stdout.Close()

	// continue to read the output until there is no more
	// or an error occurs
	if err := cmd.Start(); err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		retv += scanner.Text() + "\n"
	}
	return retv, nil
}

func (env *Environment) RunPythonScriptFromFile(scriptPath string, args ...string) error {
	// put scriptPath at the front of the args
	args = append([]string{scriptPath}, args...)
	cmd := exec.Command(env.PythonPath, args...)

	// Create a pipe for the output of the script
	stdoutPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %v", err)
	}

	// // Create a pipe for the input of the script
	// stdinPipe, err := cmd.StdinPipe()
	// if err != nil {
	// 	return fmt.Errorf("error creating stdin pipe: %v", err)
	// }

	// Start the command
	if err := cmd.Start(); err != nil {
		return err
	}

	// // Write to the command's stdin
	// go func() {
	// 	defer stdinPipe.Close()
	// 	io.WriteString(stdinPipe, "hello\n")
	// 	io.WriteString(stdinPipe, "world\n")
	// }()

	// Read from the command's stdout
	scanner := bufio.NewScanner(stdoutPipe)
	for scanner.Scan() {
		fmt.Println("Python script output:", scanner.Text())
	}

	// Wait for the command to finish
	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}
