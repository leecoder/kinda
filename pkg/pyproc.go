package pkg

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
	"text/template"
	"time"
)

// PythonProcess represents a running Python process with its I/O pipes
type PythonProcess struct {
	Cmd    *exec.Cmd
	Stdin  io.WriteCloser
	Stdout io.ReadCloser
	Stderr io.ReadCloser
	script io.WriteCloser // For writing the secondary bootstrap script
}

type Module struct {
	Name   string
	Path   string
	Source string
}

type Package struct {
	Name    string
	Path    string
	Modules []Module
}

type PythonProgram struct {
	Name     string
	Path     string
	Program  Module
	Packages []Package
}

// Data struct to hold the pipe number
type TemplateData struct {
	PipeNumber int
}

// NewModuleFromPath creates a new module from a file path
func NewModuleFromPath(name, path string) (*Module, error) {
	// load the source file from the path
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// base64 encode the source
	encoded := base64.StdEncoding.EncodeToString(source)

	return &Module{
		Name:   name,
		Path:   path,
		Source: encoded,
	}, nil
}

// NewModuleFromString creates a new module from a string
func NewModuleFromString(name, original_path string, source string) *Module {
	// base64 encode the source
	encoded := base64.StdEncoding.EncodeToString([]byte(source))

	return &Module{
		Name:   name,
		Source: encoded,
		Path:   original_path,
	}
}

// New function to create a Package
func NewPackage(name, path string, modules []Module) *Package {
	return &Package{
		Name:    name,
		Path:    path,
		Modules: modules,
	}
}

func procTemplate(templateStr string, data interface{}) string {
	// Parse the template
	tmpl, err := template.New("pythonTemplate").Parse(templateStr)
	if err != nil {
		log.Fatalf("Error parsing template: %v", err)
	}

	// Execute the template with the data
	var result bytes.Buffer
	err = tmpl.Execute(&result, data)
	if err != nil {
		log.Fatalf("Error executing template: %v", err)
	}

	return result.String()
}

func (env *Environment) NewPythonProcessFromProgram(program *PythonProgram, environment_vars map[string]string, extrafiles []*os.File, debug bool, args ...string) (*PythonProcess, error) {
	// Create two pipes
	reader_bootstrap, writer_bootstrap, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	reader_program, writer_program, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	// get the file descriptor for the bootstrap script
	reader_bootstrap_fd := reader_bootstrap.Fd()
	primaryBootstrapScript := procTemplate(primaryBootstrapScriptTemplate, TemplateData{PipeNumber: int(reader_bootstrap_fd)})

	// Create the command with the primary bootstrap script
	// fullArgs := append([]string{"-u", "-c", primaryBootstrapScript}, args...)
	// cmd := exec.Command(env.PythonPath, fullArgs...)
	cmd := exec.Command(env.PythonPath)

	// Pass both file descriptors using ExtraFiles
	// this will return a list of strings with the file descriptors
	extradescriptors := setExtraFiles(cmd, append([]*os.File{reader_bootstrap, reader_program}, extrafiles...))

	// At this point, cmd.Args will contain just the python path.  We can now append the "-c" flag and the primary bootstrap script
	cmd.Args = append(cmd.Args, "-u", "-c", primaryBootstrapScript)

	// append the count of extra files to the command arguments as a string
	cmd.Args = append(cmd.Args, fmt.Sprintf("%d", len(extradescriptors)))

	// append the extra file descriptors to the command arguments
	cmd.Args = append(cmd.Args, extradescriptors...)

	// append the program arguments to the command arguments
	cmd.Args = append(cmd.Args, args...)

	// Set environment variables
	cmd.Env = os.Environ()
	if environment_vars != nil {
		for key, value := range environment_vars {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}

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

	// Prepare the program data
	programData, err := json.Marshal(program)
	if err != nil {
		return nil, err
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	// Write the secondary bootstrap script and program data to separate pipes
	go func() {
		defer writer_bootstrap.Close()
		secondaryBootstrapScript := procTemplate(secondaryBootstrapScriptTemplate, TemplateData{PipeNumber: int(reader_program.Fd())})
		io.WriteString(writer_bootstrap, secondaryBootstrapScript)
	}()

	go func() {
		defer writer_program.Close()
		writer_program.Write(programData)
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

// NewPythonProcessFromString starts a Python script from a string with the given arguments.
// It returns a PythonProcess struct containing the command and I/O pipes.
// It ensures that the child process is killed if the parent process is killed.
func (env *Environment) NewPythonProcessFromString(script string, environment_vars map[string]string, extrafiles []*os.File, debug bool, args ...string) (*PythonProcess, error) {
	// Create a pipe
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	// Create the command with the bootstrap script
	// We want stdin/stdout to unbuffered (-u) and to run the bootstrap script
	// The "-c" flag is used to pass the script as an argument and terminates the python option list
	bootloader := procTemplate(primaryBootstrapScriptTemplate, TemplateData{PipeNumber: int(reader.Fd())})
	fullArgs := append([]string{"-u", "-c", bootloader}, args...)
	cmd := exec.Command(env.PythonPath, fullArgs...)

	// Pass the file descriptor using ExtraFiles
	// prepend our reader to the list of extra files and assign
	extrafiles = append([]*os.File{reader}, extrafiles...)
	setExtraFiles(cmd, extrafiles)

	// set it's environment variables as our environment variables
	cmd.Env = os.Environ()

	// set the environment variables if they are provided
	if environment_vars != nil {
		for key, value := range environment_vars {
			cmd.Env = append(cmd.Env, key+"="+value)
		}
	}

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

	// Write the main script to the pipe
	go func() {
		// Close the writer when the function returns
		// Python will not run the bootstrap script until the writer is closed
		defer writer.Close()
		io.WriteString(writer, script)
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
