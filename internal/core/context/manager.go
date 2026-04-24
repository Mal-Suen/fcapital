// Package context provides context management for the fcapital framework.
// It collects and manages system information, tool status, and testing history.
package context

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Session represents a testing session.
type Session struct {
	ID        string    `json:"id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Target    string    `json:"target"`
	Status    string    `json:"status"` // running, paused, completed, failed
}

// SystemInfo represents system information.
type SystemInfo struct {
	OS              string            `json:"os"`               // windows, linux, darwin
	OSVersion       string            `json:"os_version"`
	Arch            string            `json:"arch"`             // amd64, arm64
	Hostname        string            `json:"hostname"`
	Username        string            `json:"username"`
	NetworkInfo     *NetworkInfo      `json:"network_info"`
	Environment     map[string]string `json:"environment"`
	PackageManagers []string          `json:"package_managers"`
}

// NetworkInfo represents network information.
type NetworkInfo struct {
	Interfaces  []NetworkInterface `json:"interfaces"`
	DNSServers  []string           `json:"dns_servers"`
	PublicIP    string             `json:"public_ip,omitempty"`
}

// NetworkInterface represents a network interface.
type NetworkInterface struct {
	Name        string   `json:"name"`
	IPAddresses []string `json:"ip_addresses"`
	MAC         string   `json:"mac"`
}

// ToolInfo represents tool information.
type ToolInfo struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Path         string   `json:"path"`
	Status       string   `json:"status"` // ready, missing, error
	Capabilities []string `json:"capabilities"`
	Category     string   `json:"category"`
}

// PhaseResult represents a phase execution result.
type PhaseResult struct {
	PhaseID      string                 `json:"phase_id"`
	PhaseName    string                 `json:"phase_name"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	Status       string                 `json:"status"`
	Findings     map[string]interface{} `json:"findings"`
	ToolsUsed    []string               `json:"tools_used"`
	AISummary    string                 `json:"ai_summary"`
	RawOutput    string                 `json:"raw_output,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

// Message represents an AI conversation message.
type Message struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// Context represents the complete testing context.
type Context struct {
	Session       Session               `json:"session"`
	SystemInfo    SystemInfo            `json:"system_info"`
	Tools         []ToolInfo            `json:"tools"`
	PhaseHistory  []PhaseResult         `json:"phase_history"`
	CurrentPhase  string                `json:"current_phase"`
	Conversation  []Message             `json:"conversation"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

// Manager manages the testing context.
type Manager struct {
	ctx       *Context
	collector *Collector
	mu        sync.RWMutex
}

// NewManager creates a new context manager.
func NewManager() *Manager {
	return &Manager{
		ctx: &Context{
			Session: Session{
				ID:        generateSessionID(),
				StartTime: time.Now(),
				Status:    "initialized",
			},
			PhaseHistory: make([]PhaseResult, 0),
			Conversation: make([]Message, 0),
			Metadata:     make(map[string]interface{}),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		collector: NewCollector(),
	}
}

// Initialize initializes the context by collecting system information.
func (m *Manager) Initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Collect system information
	sysInfo, err := m.collector.CollectSystemInfo()
	if err != nil {
		return fmt.Errorf("failed to collect system info: %w", err)
	}
	m.ctx.SystemInfo = *sysInfo

	// Initialize empty tools list (will be populated by tool manager)
	m.ctx.Tools = make([]ToolInfo, 0)

	m.ctx.UpdatedAt = time.Now()
	return nil
}

// GetContext returns the current context.
func (m *Manager) GetContext() *Context {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ctx
}

// SetTarget sets the target for the session.
func (m *Manager) SetTarget(target string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ctx.Session.Target = target
	m.ctx.Session.Status = "running"
	m.ctx.UpdatedAt = time.Now()
}

// SetTools sets the tools information.
func (m *Manager) SetTools(tools []ToolInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ctx.Tools = tools
	m.ctx.UpdatedAt = time.Now()
}

// AddPhaseResult adds a phase result to the history.
func (m *Manager) AddPhaseResult(result *PhaseResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ctx.PhaseHistory = append(m.ctx.PhaseHistory, *result)
	m.ctx.UpdatedAt = time.Now()
}

// SetCurrentPhase sets the current phase.
func (m *Manager) SetCurrentPhase(phaseID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ctx.CurrentPhase = phaseID
	m.ctx.UpdatedAt = time.Now()
}

// AddMessage adds a message to the conversation history.
func (m *Manager) AddMessage(role, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ctx.Conversation = append(m.ctx.Conversation, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	m.ctx.UpdatedAt = time.Now()
}

// GetConversation returns the conversation history.
func (m *Manager) GetConversation() []Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ctx.Conversation
}

// GetLastPhaseResult returns the last phase result.
func (m *Manager) GetLastPhaseResult() *PhaseResult {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.ctx.PhaseHistory) == 0 {
		return nil
	}
	return &m.ctx.PhaseHistory[len(m.ctx.PhaseHistory)-1]
}

// GetReadyTools returns tools that are ready to use.
func (m *Manager) GetReadyTools() []ToolInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ready := make([]ToolInfo, 0)
	for _, tool := range m.ctx.Tools {
		if tool.Status == "ready" {
			ready = append(ready, tool)
		}
	}
	return ready
}

// GetMissingTools returns tools that are missing.
func (m *Manager) GetMissingTools() []ToolInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	missing := make([]ToolInfo, 0)
	for _, tool := range m.ctx.Tools {
		if tool.Status == "missing" {
			missing = append(missing, tool)
		}
	}
	return missing
}

// UpdateToolStatus updates a tool's status.
func (m *Manager) UpdateToolStatus(name, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, tool := range m.ctx.Tools {
		if tool.Name == name {
			m.ctx.Tools[i].Status = status
			m.ctx.UpdatedAt = time.Now()
			return
		}
	}
}

// GetSystemInfoSummary returns a human-readable summary of system info.
func (m *Manager) GetSystemInfoSummary() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("OS: %s %s (%s)\n", m.ctx.SystemInfo.OS, m.ctx.SystemInfo.OSVersion, m.ctx.SystemInfo.Arch))
	sb.WriteString(fmt.Sprintf("Hostname: %s\n", m.ctx.SystemInfo.Hostname))
	sb.WriteString(fmt.Sprintf("Username: %s\n", m.ctx.SystemInfo.Username))
	sb.WriteString(fmt.Sprintf("Package Managers: %s\n", strings.Join(m.ctx.SystemInfo.PackageManagers, ", ")))
	return sb.String()
}

// GetToolsSummary returns a human-readable summary of tools.
func (m *Manager) GetToolsSummary() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ready := 0
	missing := 0
	for _, tool := range m.ctx.Tools {
		if tool.Status == "ready" {
			ready++
		} else if tool.Status == "missing" {
			missing++
		}
	}

	return fmt.Sprintf("Tools: %d ready, %d missing", ready, missing)
}

// ToJSON returns the context as JSON.
func (m *Manager) ToJSON() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.MarshalIndent(m.ctx, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Save saves the context to a file.
func (m *Manager) Save(path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.MarshalIndent(m.ctx, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// Load loads the context from a file.
func (m *Manager) Load(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, m.ctx)
}

// CompleteSession marks the session as completed.
func (m *Manager) CompleteSession() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ctx.Session.Status = "completed"
	m.ctx.Session.EndTime = time.Now()
	m.ctx.UpdatedAt = time.Now()
}

// Collector collects system information.
type Collector struct{}

// NewCollector creates a new collector.
func NewCollector() *Collector {
	return &Collector{}
}

// CollectSystemInfo collects system information.
func (c *Collector) CollectSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		Environment: make(map[string]string),
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err == nil {
		info.Hostname = hostname
	}

	// Get username
	info.Username = os.Getenv("USERNAME")
	if info.Username == "" {
		info.Username = os.Getenv("USER")
	}

	// Get OS version
	info.OSVersion = c.getOSVersion()

	// Get network info
	info.NetworkInfo = c.getNetworkInfo()

	// Get relevant environment variables
	for _, key := range []string{"PATH", "HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY"} {
		if val := os.Getenv(key); val != "" {
			info.Environment[key] = val
		}
	}

	// Detect available package managers
	info.PackageManagers = c.detectPackageManagers()

	return info, nil
}

// getOSVersion returns the OS version.
func (c *Collector) getOSVersion() string {
	switch runtime.GOOS {
	case "windows":
		// Try to get Windows version
		out, err := exec.Command("cmd", "/c", "ver").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	case "linux":
		// Try to get Linux distribution
		if data, err := os.ReadFile("/etc/os-release"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
				}
			}
		}
	case "darwin":
		out, err := exec.Command("sw_vers", "-productVersion").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	return "unknown"
}

// getNetworkInfo returns network information.
func (c *Collector) getNetworkInfo() *NetworkInfo {
	info := &NetworkInfo{
		Interfaces: make([]NetworkInterface, 0),
	}

	// Get network interfaces
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
				continue
			}

			netIface := NetworkInterface{
				Name: iface.Name,
				MAC:  iface.HardwareAddr.String(),
			}

			addrs, err := iface.Addrs()
			if err == nil {
				for _, addr := range addrs {
					netIface.IPAddresses = append(netIface.IPAddresses, addr.String())
				}
			}

			info.Interfaces = append(info.Interfaces, netIface)
		}
	}

	return info
}

// detectPackageManagers detects available package managers.
func (c *Collector) detectPackageManagers() []string {
	managers := make([]string, 0)

	// Common package managers to check
	toCheck := []string{
		// Windows
		"winget", "choco", "scoop",
		// Linux
		"apt", "yum", "dnf", "pacman", "zypper", "apk", "emerge",
		// macOS
		"brew", "port",
		// Cross-platform
		"go", "pip", "pip3", "npm", "cargo", "gem",
	}

	for _, mgr := range toCheck {
		if c.commandExists(mgr) {
			managers = append(managers, mgr)
		}
	}

	return managers
}

// commandExists checks if a command exists.
func (c *Collector) commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// generateSessionID generates a unique session ID.
func generateSessionID() string {
	return fmt.Sprintf("sess-%d", time.Now().UnixNano())
}
