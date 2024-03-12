package pkg

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type Environment struct {
	Name              string  // Name of the environment
	RootDir           string  // Root directory of the environment
	EnvPath           string  // Path to the environment
	EnvBinPath        string  // Path to the bin directory within the environment
	EnvLibPath        string  // Path to the lib directory within the environment
	PythonVersion     Version // Version of the Python environment
	MicromambaVersion Version // Version of the micromamba executable
	PipVersion        Version // Version of the pip executable
	MicromambaPath    string  // Path to the micromamba executable
	PythonPath        string  // Path to the Python executable within the environment
	PythonLibPath     string  // Path to the Python library within the environment
	PipPath           string  // Path to the pip executable within the environment
	PythonHeadersPath string  // Path to the Python headers within the environment
	SitePackagesPath  string  // Path to the site-packages directory within the environment
}

func CreateEnvironment(envName string, rootDir string, pythonVersion string, channel string) (*Environment, error) {
	requestedVersion, err := ParseVersion(pythonVersion)
	if err != nil {
		return nil, fmt.Errorf("error parsing requested python version: %v", err)
	}

	binDirectory := filepath.Join(rootDir, "bin")
	// Check if the specified root directory exists
	if _, err := os.Stat(binDirectory); os.IsNotExist(err) {
		// Ensure the target bin directory exists
		if err := os.MkdirAll(binDirectory, 0755); err != nil {
			return nil, fmt.Errorf("error creating directory: %v", err)
		}
	}

	// Check if the specified root directory is writable
	if !isDirWritable(rootDir) {
		return nil, fmt.Errorf("root directory is not writable: %s", rootDir)
	}

	// Detect platform and architecture
	platform := runtime.GOOS
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "64"
	case "arm64":
		if platform == "win" {
			// As of now, there is not a separate arm64 download for Windows
			// We'll use the same download as for amd64
			arch = "64"
		}
	default:
		return nil, fmt.Errorf("unsupported architecture: %s", arch)
	}

	// Convert platform and arch to match micromamba naming
	var executableName string = "micromamba"
	if platform == "windows" {
		executableName += ".exe"
	}

	// Create the environment object
	env := &Environment{
		Name:           envName,
		RootDir:        rootDir,
		MicromambaPath: filepath.Join(binDirectory, executableName),
	}

	// Check if binDirectory already has micromamba by getting it's version
	mver, err := RunReadStdout(env.MicromambaPath, "micromamba", "--version")
	if err != nil {
		_, ok := err.(*fs.PathError)
		if ok {
			// download micromamba
			fmt.Println("Downloading micromamba")
			env.MicromambaPath, err = ExpectMicromamba(binDirectory)
			if err != nil {
				return nil, fmt.Errorf("error downloading micromamba: %v", err)
			}
			mver, err = RunReadStdout(env.MicromambaPath, "micromamba", "--version")
			if err != nil {
				return nil, fmt.Errorf("error running micromamba --version: %v", err)
			}
		} else {
			return nil, fmt.Errorf("error running micromamba --version: %v", err)
		}
	}

	env.MicromambaVersion, err = ParseVersion(mver)
	if err != nil {
		return nil, fmt.Errorf("error parsing micromamba version: %v", err)
	}

	// check if the environment exists
	envPath := filepath.Join(env.RootDir, "envs", env.Name)
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		// Create a new Python environment with micromamba
		fmt.Println("Creating Python environment...")
		var createEnvCmd *exec.Cmd = nil
		if channel != "" {
			createEnvCmd = exec.Command(env.MicromambaPath, "--root-prefix", env.RootDir, "create", "-n", env.Name, "python="+pythonVersion, "-c", "conda-forge", "-y")
		} else {
			createEnvCmd = exec.Command(env.MicromambaPath, "create", "-n", env.Name, "python="+pythonVersion, "-y")
		}

		createEnvCmd.Stdout = os.Stdout
		createEnvCmd.Stderr = os.Stderr
		createEnvCmd.Env = append(os.Environ(), "MAMBA_ROOT_PREFIX="+env.RootDir)
		if err := createEnvCmd.Run(); err != nil {
			return nil, err
		}
	}

	// Construct the full paths to the Python and pip executables within the created environment
	env.EnvPath = envPath
	env.EnvBinPath = filepath.Join(env.RootDir, "envs", env.Name, "bin")
	env.PythonPath = filepath.Join(env.EnvBinPath, "python")
	env.PipPath = filepath.Join(env.EnvBinPath, "pip")
	env.SitePackagesPath = filepath.Join(env.RootDir, "envs", env.Name, "lib", "python"+requestedVersion.MinorString(), "site-packages")

	// find the python lib path
	env.EnvLibPath = filepath.Join(env.RootDir, "envs", env.Name, "lib")
	env.PythonLibPath = env.EnvLibPath
	if platform == "windows" {
		env.PythonLibPath = filepath.Join(env.RootDir, "envs", env.Name, "python"+requestedVersion.MinorString()+".dll")
	} else if platform == "darwin" {
		env.PythonLibPath = filepath.Join(env.RootDir, "envs", env.Name, "lib", "libpython"+requestedVersion.MinorString()+".dylib")
	} else {
		env.PythonLibPath = filepath.Join(env.RootDir, "envs", env.Name, "lib", "libpython"+requestedVersion.MinorString()+".so")
	}

	// find the python headers path
	env.PythonHeadersPath = filepath.Join(env.RootDir, "envs", env.Name, "include", "python"+requestedVersion.MinorString())

	// Check if the Python executable exists and get its version
	pver, err := RunReadStdout(env.PythonPath, "--version")
	if err != nil {
		return nil, fmt.Errorf("error running python --version: %v", err)
	}
	env.PythonVersion, err = ParsePythonVersion(pver)
	if err != nil {
		return nil, fmt.Errorf("error parsing Python version: %v", err)
	}
	// Check if the Python lib exists
	if _, err := os.Stat(env.PythonLibPath); os.IsNotExist(err) {
		env.PythonLibPath = ""
	}

	// Check if the pip executable exists and get its version
	pipver, err := RunReadStdout(env.PipPath, "--version")
	if err != nil {
		return nil, fmt.Errorf("error running pip --version: %v", err)
	}
	env.PipVersion, err = ParsePipVersion(pipver)
	if err != nil {
		return nil, fmt.Errorf("error parsing pip version: %v", err)
	}

	// ensure the python version is equal or greater than the requested version
	if env.PythonVersion.Compare(requestedVersion) < 0 {
		return nil, fmt.Errorf("requested python version %s is not available, found %s", requestedVersion.String(), env.PythonVersion.String())
	}

	return env, nil
}

func ExpectMicromamba(binFolder string) (string, error) {
	// Detect platform and architecture
	platform := runtime.GOOS
	arch := runtime.GOARCH

	// Convert platform and arch to match micromamba naming
	var executableName string = "micromamba"
	if platform == "darwin" {
		platform = "osx"
	}

	switch arch {
	case "amd64":
		arch = "64"
	case "arm64":
		if platform == "win" {
			// As of now, there is not a separate arm64 download for Windows
			return "", fmt.Errorf("windows arm 64 not supported: %s", arch)
		}
	default:
		return "", fmt.Errorf("unsupported architecture: %s", arch)
	}

	// Construct the download URL
	var downloadURL string
	version := "" // Use this to specify a version, or leave empty for latest
	// https://github.com/mamba-org/micromamba-releases/releases/download/1.5.7-0/micromamba-osx-arm64
	if version == "" {
		downloadURL = fmt.Sprintf("https://github.com/mamba-org/micromamba-releases/releases/latest/download/%s-%s-%s", executableName, platform, arch)
	} else {
		downloadURL = fmt.Sprintf("https://github.com/mamba-org/micromamba-releases/releases/download/%s/%s-%s-%s", version, executableName, platform, arch)
	}

	// Ensure the target bin directory exists
	if err := os.MkdirAll(binFolder, 0755); err != nil {
		return "", fmt.Errorf("error creating directory: %v", err)
	}

	// Download the binary
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("error downloading file: %v", err)
	}
	defer resp.Body.Close()

	// Create the file
	if platform == "windows" {
		executableName += ".exe"
	}
	binpath := filepath.Join(binFolder, executableName)
	outFile, err := os.Create(binpath)
	if err != nil {
		return "", fmt.Errorf("error creating file: %v", err)
	}
	defer outFile.Close()

	// Write the body to file
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("error writing file: %v", err)
	}

	// Change file permissions to make it executable (not applicable for Windows)
	if platform != "windows" {
		if err := os.Chmod(binpath, 0755); err != nil {
			return "", fmt.Errorf("error setting file permissions: %v", err)
		}
	}

	return binpath, nil
}
