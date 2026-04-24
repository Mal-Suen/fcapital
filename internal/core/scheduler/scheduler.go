// Package scheduler provides intelligent tool scheduling and execution.
// It handles tool discovery, installation, and execution based on capabilities.
package scheduler

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ToolDefinition represents a tool's definition.
type ToolDefinition struct {
	Name          string          `json:"name" yaml:"name"`
	Description   string          `json:"description" yaml:"description"`
	Category      string          `json:"category" yaml:"category"`
	Capabilities  []string        `json:"capabilities" yaml:"capabilities"`
	InstallMethods []InstallMethod `json:"install_methods" yaml:"install_methods"`
	Fallbacks     []string        `json:"fallbacks" yaml:"fallbacks"`
	VerifyCmd     string          `json:"verify_cmd" yaml:"verify_cmd"`
}

// InstallMethod represents a method to install a tool.
type InstallMethod struct {
	Type        string `json:"type" yaml:"type"`               // winget, apt, brew, go, pip, etc.
	Package     string `json:"package" yaml:"package"`         // Package name or URL
	PostInstall string `json:"post_install" yaml:"post_install"` // Command to run after install
	VerifyCmd   string `json:"verify_cmd" yaml:"verify_cmd"`   // Override verify command
}

// ToolStatus represents the status of a tool.
type ToolStatus struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Path      string `json:"path"`
	Status    string `json:"status"` // ready, missing, error
	Error     string `json:"error,omitempty"`
}

// ExecutionResult represents the result of tool execution.
type ExecutionResult struct {
	ToolName  string        `json:"tool_name"`
	Command   string        `json:"command"`
	Args      []string      `json:"args"`
	Output    string        `json:"output"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	Success   bool          `json:"success"`
}

// ScheduleRequest represents a request to schedule a tool.
type ScheduleRequest struct {
	Capability string                 `json:"capability"` // Required capability
	Target     string                 `json:"target"`
	Args       []string               `json:"args"`
	Options    map[string]interface{} `json:"options"`
}

// Scheduler manages tool scheduling and execution.
type Scheduler struct {
	tools          map[string]*ToolDefinition
	capabilityMap  map[string]string // capability -> primary tool
	toolStatus     map[string]*ToolStatus
	runner         *Runner
	installer      *Installer
	mu             sync.RWMutex
	logger         Logger
}

// Logger interface for logging.
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// Option is a functional option for Scheduler.
type Option func(*Scheduler)

// WithLogger sets the logger.
func WithLogger(l Logger) Option {
	return func(s *Scheduler) {
		s.logger = l
	}
}

// New creates a new Scheduler.
func New(opts ...Option) *Scheduler {
	s := &Scheduler{
		tools:         make(map[string]*ToolDefinition),
		capabilityMap: make(map[string]string),
		toolStatus:    make(map[string]*ToolStatus),
		logger:        &noopLogger{},
	}
	for _, opt := range opts {
		opt(s)
	}
	s.runner = NewRunner(WithRunnerLogger(s.logger))
	s.installer = NewInstaller(WithInstallerLogger(s.logger))
	return s
}

// RegisterTool registers a tool definition.
func (s *Scheduler) RegisterTool(def *ToolDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tools[def.Name]; exists {
		return fmt.Errorf("tool %s already registered", def.Name)
	}

	s.tools[def.Name] = def

	// Update capability map
	for _, cap := range def.Capabilities {
		if _, exists := s.capabilityMap[cap]; !exists {
			s.capabilityMap[cap] = def.Name
		}
	}

	return nil
}

// RegisterTools registers multiple tool definitions.
func (s *Scheduler) RegisterTools(defs []*ToolDefinition) error {
	for _, def := range defs {
		if err := s.RegisterTool(def); err != nil {
			return err
		}
	}
	return nil
}

// CheckAvailability checks if a tool is available.
func (s *Scheduler) CheckAvailability(name string) (*ToolStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check cache
	if status, exists := s.toolStatus[name]; exists {
		return status, nil
	}

	// Check if tool is registered
	def, exists := s.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not registered", name)
	}

	// Check if tool exists in PATH
	status := &ToolStatus{
		Name:   name,
		Status: "missing",
	}

	path, err := exec.LookPath(name)
	if err == nil {
		status.Path = path
		status.Status = "ready"

		// Get version if verify command exists
		if def.VerifyCmd != "" {
			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				cmd = exec.Command("cmd", "/c", def.VerifyCmd)
			} else {
				cmd = exec.Command("sh", "-c", def.VerifyCmd)
			}
			out, err := cmd.CombinedOutput()
			if err == nil {
				status.Version = strings.TrimSpace(string(out))
			}
		}
	}

	s.toolStatus[name] = status
	return status, nil
}

// FindToolByCapability finds a tool that provides the given capability.
func (s *Scheduler) FindToolByCapability(capability string) (*ToolDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	name, exists := s.capabilityMap[capability]
	if !exists {
		return nil, fmt.Errorf("no tool provides capability: %s", capability)
	}

	return s.tools[name], nil
}

// Schedule schedules a tool for execution based on capability.
func (s *Scheduler) Schedule(ctx context.Context, req *ScheduleRequest) (*ExecutionResult, error) {
	// Find tool by capability
	def, err := s.FindToolByCapability(req.Capability)
	if err != nil {
		return nil, err
	}

	// Check availability
	status, err := s.CheckAvailability(def.Name)
	if err != nil {
		return nil, err
	}

	// If missing, try to install
	if status.Status == "missing" {
		s.logger.Info("Tool %s is missing, attempting installation...", def.Name)
		if err := s.InstallTool(ctx, def.Name); err != nil {
			// Try fallbacks
			for _, fallback := range def.Fallbacks {
				s.logger.Info("Trying fallback tool: %s", fallback)
				fbStatus, _ := s.CheckAvailability(fallback)
				if fbStatus.Status == "ready" {
					return s.Execute(ctx, fallback, req.Args)
				}
			}
			return nil, fmt.Errorf("failed to install %s and no fallbacks available: %w", def.Name, err)
		}
	}

	return s.Execute(ctx, def.Name, req.Args)
}

// InstallTool installs a tool.
func (s *Scheduler) InstallTool(ctx context.Context, name string) error {
	s.mu.RLock()
	def, exists := s.tools[name]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("tool %s not registered", name)
	}

	// Detect available package managers
	managers := s.installer.DetectPackageManagers()

	// Try each install method
	for _, method := range def.InstallMethods {
		if !s.installer.IsManagerAvailable(method.Type, managers) {
			continue
		}

		s.logger.Info("Installing %s using %s...", name, method.Type)
		if err := s.installer.Install(ctx, &method); err != nil {
			s.logger.Warn("Installation with %s failed: %v", method.Type, err)
			continue
		}

		// Verify installation
		time.Sleep(1 * time.Second) // Wait for installation to complete
		status, _ := s.CheckAvailability(name)
		if status.Status == "ready" {
			s.logger.Info("Successfully installed %s", name)
			return nil
		}
	}

	return fmt.Errorf("failed to install %s with all available methods", name)
}

// Execute executes a tool with the given arguments.
func (s *Scheduler) Execute(ctx context.Context, name string, args []string) (*ExecutionResult, error) {
	s.mu.RLock()
	_, exists := s.tools[name]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tool %s not registered", name)
	}

	status, err := s.CheckAvailability(name)
	if err != nil {
		return nil, err
	}

	if status.Status != "ready" {
		return nil, fmt.Errorf("tool %s is not available", name)
	}

	return s.runner.Run(ctx, name, args...)
}

// GetToolStatus returns the status of all tools.
func (s *Scheduler) GetToolStatus() map[string]*ToolStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*ToolStatus)
	for name := range s.tools {
		if status, exists := s.toolStatus[name]; exists {
			result[name] = status
		} else {
			result[name] = &ToolStatus{
				Name:   name,
				Status: "unknown",
			}
		}
	}
	return result
}

// GetMissingTools returns tools that are not installed.
func (s *Scheduler) GetMissingTools() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	missing := make([]string, 0)
	for name, status := range s.toolStatus {
		if status.Status == "missing" {
			missing = append(missing, name)
		}
	}
	return missing
}

// Runner executes tools.
type Runner struct {
	timeout time.Duration
	logger  Logger
}

// NewRunner creates a new Runner.
func NewRunner(opts ...RunnerOption) *Runner {
	r := &Runner{
		timeout: 10 * time.Minute,
		logger:  &noopLogger{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// RunnerOption is a functional option for Runner.
type RunnerOption func(*Runner)

// WithRunnerLogger sets the logger.
func WithRunnerLogger(l Logger) RunnerOption {
	return func(r *Runner) {
		r.logger = l
	}
}

// WithTimeout sets the timeout.
func WithTimeout(d time.Duration) RunnerOption {
	return func(r *Runner) {
		r.timeout = d
	}
}

// Run executes a command.
func (r *Runner) Run(ctx context.Context, name string, args ...string) (*ExecutionResult, error) {
	start := time.Now()
	result := &ExecutionResult{
		ToolName: name,
		Args:     args,
	}

	// Create timeout context
	runCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Build command
	cmd := exec.CommandContext(runCtx, name, args...)
	result.Command = cmd.String()

	// Execute
	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = err.Error()
		result.Success = false
		return result, fmt.Errorf("command failed: %w", err)
	}

	result.Success = true
	return result, nil
}

// Installer handles tool installation.
type Installer struct {
	logger Logger
}

// NewInstaller creates a new Installer.
func NewInstaller(opts ...InstallerOption) *Installer {
	i := &Installer{
		logger: &noopLogger{},
	}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

// InstallerOption is a functional option for Installer.
type InstallerOption func(*Installer)

// WithInstallerLogger sets the logger.
func WithInstallerLogger(l Logger) InstallerOption {
	return func(i *Installer) {
		i.logger = l
	}
}

// DetectPackageManagers detects available package managers.
func (i *Installer) DetectPackageManagers() []string {
	managers := make([]string, 0)

	// Order by priority
	toCheck := []string{
		// Windows
		"winget", "choco", "scoop",
		// Linux
		"apt", "yum", "dnf", "pacman", "zypper", "apk",
		// macOS
		"brew", "port",
		// Cross-platform
		"go", "pip", "pip3", "npm", "cargo", "gem",
	}

	for _, mgr := range toCheck {
		if _, err := exec.LookPath(mgr); err == nil {
			managers = append(managers, mgr)
		}
	}

	return managers
}

// IsManagerAvailable checks if a package manager is available.
func (i *Installer) IsManagerAvailable(managerType string, available []string) bool {
	for _, m := range available {
		if m == managerType {
			return true
		}
	}
	return false
}

// Install installs a tool using the specified method.
func (i *Installer) Install(ctx context.Context, method *InstallMethod) error {
	var cmd *exec.Cmd

	switch method.Type {
	case "winget":
		cmd = exec.CommandContext(ctx, "winget", "install", "--id", method.Package, "-e", "--accept-source-agreements", "--accept-package-agreements")
	case "choco":
		cmd = exec.CommandContext(ctx, "choco", "install", method.Package, "-y")
	case "scoop":
		cmd = exec.CommandContext(ctx, "scoop", "install", method.Package)
	case "apt":
		cmd = exec.CommandContext(ctx, "apt", "install", "-y", method.Package)
	case "yum":
		cmd = exec.CommandContext(ctx, "yum", "install", "-y", method.Package)
	case "dnf":
		cmd = exec.CommandContext(ctx, "dnf", "install", "-y", method.Package)
	case "pacman":
		cmd = exec.CommandContext(ctx, "pacman", "-S", "--noconfirm", method.Package)
	case "brew":
		cmd = exec.CommandContext(ctx, "brew", "install", method.Package)
	case "go":
		cmd = exec.CommandContext(ctx, "go", "install", method.Package)
	case "pip":
		cmd = exec.CommandContext(ctx, "pip", "install", method.Package)
	case "pip3":
		cmd = exec.CommandContext(ctx, "pip3", "install", method.Package)
	case "npm":
		cmd = exec.CommandContext(ctx, "npm", "install", "-g", method.Package)
	case "cargo":
		cmd = exec.CommandContext(ctx, "cargo", "install", method.Package)
	case "gem":
		cmd = exec.CommandContext(ctx, "gem", "install", method.Package)
	default:
		return fmt.Errorf("unsupported package manager: %s", method.Type)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("installation failed: %w\nOutput: %s", err, string(output))
	}

	// Run post-install command if specified
	if method.PostInstall != "" {
		postCmd := exec.CommandContext(ctx, "cmd", "/c", method.PostInstall)
		if out, err := postCmd.CombinedOutput(); err != nil {
			i.logger.Warn("Post-install command failed: %v\nOutput: %s", err, string(out))
		}
	}

	return nil
}

// noopLogger is a no-op logger.
type noopLogger struct{}

func (l *noopLogger) Debug(msg string, args ...interface{}) {}
func (l *noopLogger) Info(msg string, args ...interface{})  {}
func (l *noopLogger) Warn(msg string, args ...interface{})  {}
func (l *noopLogger) Error(msg string, args ...interface{}) {}
