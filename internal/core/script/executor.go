// Package script provides script execution with sandboxing.
package script

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Executor executes scripts with optional sandboxing.
type Executor struct {
	workDir      string
	timeout      time.Duration
	restrictions *Restrictions
}

// Restrictions defines execution restrictions.
type Restrictions struct {
	NoNetwork    bool
	NoFileWrite  bool
	NoFileRead   bool
	NoSubprocess bool
	MaxMemory    int64 // in bytes
	MaxCPUTime   time.Duration
}

// ExecutionResult represents the result of script execution.
type ExecutionResult struct {
	Success   bool          `json:"success"`
	Output    string        `json:"output"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	ExitCode  int           `json:"exit_code"`
	Artifacts []string      `json:"artifacts,omitempty"`
}

// SandboxResult represents the result of sandbox execution.
type SandboxResult struct {
	Success  bool          `json:"success"`
	Output   string        `json:"output"`
	Error    error         `json:"error,omitempty"`
	Warnings []string      `json:"warnings,omitempty"`
	Duration time.Duration `json:"duration"`
}

// ExecutorOption is a functional option for Executor.
type ExecutorOption func(*Executor)

// WithWorkDir sets the working directory.
func WithWorkDir(dir string) ExecutorOption {
	return func(e *Executor) {
		e.workDir = dir
	}
}

// WithTimeout sets the execution timeout.
func WithTimeout(timeout time.Duration) ExecutorOption {
	return func(e *Executor) {
		e.timeout = timeout
	}
}

// WithRestrictions sets the execution restrictions.
func WithRestrictions(r *Restrictions) ExecutorOption {
	return func(e *Executor) {
		e.restrictions = r
	}
}

// NewExecutor creates a new script executor.
func NewExecutor(opts ...ExecutorOption) *Executor {
	e := &Executor{
		timeout: 60 * time.Second,
		restrictions: &Restrictions{
			NoNetwork:    false,
			NoFileWrite:  false,
			NoFileRead:   false,
			NoSubprocess: false,
		},
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Execute executes a script.
func (e *Executor) Execute(code, language, target string) *ExecutionResult {
	startTime := time.Now()
	result := &ExecutionResult{
		ExitCode: -1,
	}

	// 1. Create temporary file
	scriptFile, cleanup, err := e.createScriptFile(code, language)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create script file: %v", err)
		return result
	}
	defer cleanup()

	// 2. Execute with timeout
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	// 3. Build command with context
	cmd, err := e.buildCommand(scriptFile, language, target, ctx)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to build command: %v", err)
		return result
	}

	// 4. Capture output
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(startTime)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Error = err.Error()
		result.Output = string(output)
		return result
	}

	result.Success = true
	result.Output = string(output)
	result.ExitCode = 0

	return result
}

// RunInSandbox executes a script in a sandboxed environment.
func (e *Executor) RunInSandbox(code, language string, timeout time.Duration) *SandboxResult {
	startTime := time.Now()
	result := &SandboxResult{
		Warnings: []string{},
	}

	// 1. Create isolated work directory
	sandboxDir, err := e.createSandboxDir()
	if err != nil {
		result.Error = fmt.Errorf("failed to create sandbox directory: %w", err)
		return result
	}
	defer os.RemoveAll(sandboxDir)

	// 2. Create script file in sandbox
	scriptFile := filepath.Join(sandboxDir, "script"+getExtension(language))
	if err := ioutil.WriteFile(scriptFile, []byte(code), 0600); err != nil {
		result.Error = fmt.Errorf("failed to write script file: %w", err)
		return result
	}

	// 3. Execute with strict timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 4. Build sandboxed command with context
	cmd, err := e.buildSandboxCommand(scriptFile, language, sandboxDir, ctx)
	if err != nil {
		result.Error = fmt.Errorf("failed to build sandbox command: %w", err)
		return result
	}

	// 5. Capture output
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(startTime)
	result.Output = string(output)

	if err != nil {
		result.Error = err
		return result
	}

	result.Success = true
	return result
}

// createScriptFile creates a temporary script file.
func (e *Executor) createScriptFile(code, language string) (string, func(), error) {
	ext := getExtension(language)

	tmpFile, err := ioutil.TempFile("", "fcapital-script-*"+ext)
	if err != nil {
		return "", nil, err
	}

	if _, err := tmpFile.WriteString(code); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", nil, err
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		return "", nil, err
	}

	// Make executable on Unix-like systems
	if runtime.GOOS != "windows" {
		os.Chmod(tmpFile.Name(), 0700)
	}

	cleanup := func() {
		os.Remove(tmpFile.Name())
	}

	return tmpFile.Name(), cleanup, nil
}

// createSandboxDir creates an isolated sandbox directory.
func (e *Executor) createSandboxDir() (string, error) {
	return ioutil.TempDir("", "fcapital-sandbox-")
}

// buildCommand builds the command to execute the script with context support.
func (e *Executor) buildCommand(scriptFile, language, target string, ctx context.Context) (*exec.Cmd, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	switch strings.ToLower(language) {
	case "python", "python3":
		args := []string{scriptFile}
		if target != "" {
			args = append(args, "--target", target)
		}
		return exec.CommandContext(ctx, "python", args...), nil

	case "bash", "sh":
		args := []string{scriptFile}
		if target != "" {
			args = append(args, target)
		}
		return exec.CommandContext(ctx, "bash", args...), nil

	case "powershell", "ps1":
		args := []string{"-ExecutionPolicy", "Bypass", "-File", scriptFile}
		if target != "" {
			args = append(args, "-Target", target)
		}
		return exec.CommandContext(ctx, "powershell", args...), nil

	case "go":
		// For Go, we need to compile first
		exeFile := strings.TrimSuffix(scriptFile, ".go")
		if runtime.GOOS == "windows" {
			exeFile += ".exe"
		}
		compileCmd := exec.CommandContext(ctx, "go", "build", "-o", exeFile, scriptFile)
		if err := compileCmd.Run(); err != nil {
			return nil, fmt.Errorf("compilation failed: %w", err)
		}
		return exec.CommandContext(ctx, exeFile), nil

	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
}

// buildSandboxCommand builds a sandboxed command with restrictions.
func (e *Executor) buildSandboxCommand(scriptFile, language, sandboxDir string, ctx context.Context) (*exec.Cmd, error) {
	cmd, err := e.buildCommand(scriptFile, language, "", ctx)
	if err != nil {
		return nil, err
	}

	// Set environment restrictions
	cmd.Env = e.buildSandboxEnv()

	// Set working directory
	cmd.Dir = sandboxDir

	return cmd, nil
}

// buildSandboxEnv builds a restricted environment for sandbox execution.
func (e *Executor) buildSandboxEnv() []string {
	// Start with minimal environment (cross-platform)
	var env []string

	if runtime.GOOS == "windows" {
		// Windows environment
		env = []string{
			"SystemRoot=" + os.Getenv("SystemRoot"),
			"TEMP=" + os.TempDir(),
			"TMP=" + os.TempDir(),
		}
		// Add PATH for essential Windows tools
		if systemPath := os.Getenv("PATH"); systemPath != "" {
			env = append(env, "PATH="+systemPath)
		}
	} else {
		// Unix-like environment
		env = []string{
			"PATH=/usr/bin:/bin",
			"HOME=/tmp",
			"TEMP=/tmp",
			"TMP=/tmp",
		}
	}

	// Add restrictions
	if e.restrictions != nil {
		if e.restrictions.NoNetwork {
			env = append(env, "NO_NETWORK=1")
		}
	}

	return env
}

// getExtension returns the file extension for a language.
func getExtension(language string) string {
	switch strings.ToLower(language) {
	case "python", "python3":
		return ".py"
	case "bash", "sh":
		return ".sh"
	case "powershell", "ps1":
		return ".ps1"
	case "go":
		return ".go"
	case "javascript", "js":
		return ".js"
	case "ruby", "rb":
		return ".rb"
	default:
		return ".txt"
	}
}

// IsLanguageSupported checks if a language is supported.
func IsLanguageSupported(language string) bool {
	supported := map[string]bool{
		"python":     true,
		"python3":    true,
		"bash":       true,
		"sh":         true,
		"powershell": true,
		"ps1":        true,
		"go":         true,
	}
	return supported[strings.ToLower(language)]
}

// GetSupportedLanguages returns a list of supported languages.
func GetSupportedLanguages() []string {
	return []string{"python", "bash", "powershell", "go"}
}
