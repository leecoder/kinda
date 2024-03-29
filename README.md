# Kinda: A Golang Python Environment Manager
Kinda is a Go package that provides functionality to create and manage Python environments, similar to conda. It is designed to streamline the process of setting up Python environments, installing packages, and running Python scripts within those environments.

## Features
Create isolated Python environments with specified Python versions
Automatically download and install micromamba, a lightweight conda alternative
Install packages into the Python environment using pip or micromamba
Run Python scripts within the created environment
Clone Git repositories and install project dependencies
## Installation
To use Kinda in your Go project, you can install it using go get:

```bash
go get github.com/richinsley/kinda
```

## Usage
Creating an Environment
To create a new Python environment, use the CreateEnvironment function:
```go
env, err := kinda.CreateEnvironment("myenv", "/path/to/root", "3.10", "conda-forge")
if err != nil {
    // Handle error
}
```

This will create a new Python environment named "myenv" with Python 3.10 installed, using the "conda-forge" channel.

Installing Packages
To install packages into the Python environment using pip, use the PipInstallPackages or PipInstallRequirements methods:

```bash
err := env.PipInstallPackages([]string{"numpy", "pandas"}, "", "", false)
if err != nil {
    // Handle error
}

err := env.PipInstallRequirements("requirements.txt")
if err != nil {
    // Handle error
}
```
To install packages using micromamba, use the MicromambaInstallPackage method:

```go
err := env.MicromambaInstallPackage("numpy", "conda-forge")
if err != nil {
    // Handle error
}
```

## Running Python Scripts
To run a Python script within the created environment, use the RunPythonScriptFromFile method:

```go
err := env.RunPythonScriptFromFile("script.py", "arg1", "arg2")
if err != nil {
    // Handle error
}
```
## Cloning Repositories
You can integrate with [go-git](https://github.com/go-git/go-git) to retrieve git python projects and and install its dependencies using Kinda:



```go
repo, err := git.PlainClone("/path/to/repo", false, &git.CloneOptions{
    URL:      "https://github.com/example/repo.git",
    Progress: os.Stdout,
})
if err != nil {
    // Handle error
}

err := env.PipInstallRequirements("/path/to/repo/requirements.txt")
if err != nil {
    // Handle error
}
```
## Why Kinda?
Kinda is a lightweight and easy-to-use alternative to conda, designed specifically for Go projects that need to manage Python environments. It leverages micromamba, a minimal implementation of conda, to provide fast and efficient environment management.

By using Kinda, you can easily create isolated Python environments, install packages, and run Python scripts without the overhead of a full conda installation. Kinda simplifies the process of setting up and managing Python dependencies in your Go projects.

## License
Kinda is released under the MIT License.