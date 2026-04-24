package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// LogEntry 日志条目
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Phase     string    `json:"phase"`
	Action    string    `json:"action"`
	Tool      string    `json:"tool"`
	Message   string    `json:"message"`
	Success   bool      `json:"success"`
	Output    string    `json:"output,omitempty"`
}

// SessionLog 会话日志
type SessionLog struct {
	ID          string        `json:"id"`
	Target      string        `json:"target"`
	StartTime   time.Time     `json:"start_time"`
	EndTime     *time.Time    `json:"end_time,omitempty"`
	Status      string        `json:"status"` // running, completed, interrupted
	Results     []PhaseResult `json:"results"`
	History     []HistoryEntry `json:"history"`
	Logs        []LogEntry    `json:"logs"`
	CurrentPhase string       `json:"current_phase"`
	NextAction  string        `json:"next_action,omitempty"` // 下一步建议，用于恢复
}

// PhaseResult 阶段结果
type PhaseResult struct {
	Phase   string `json:"phase"`
	Tool    string `json:"tool"`
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Summary string `json:"summary"`
}

// HistoryEntry 历史记录
type HistoryEntry struct {
	Action  string `json:"action"`
	Tool    string `json:"tool"`
	Result  string `json:"result"`
	Summary string `json:"summary"`
}

// Logger 日志管理器
type Logger struct {
	logDir     string
	sessionID  string
	session    *SessionLog
	file       *os.File
}

// NewLogger 创建新的日志管理器
func NewLogger(logDir string) *Logger {
	if logDir == "" {
		// 默认日志目录
		homeDir, _ := os.UserHomeDir()
		logDir = filepath.Join(homeDir, ".fcapital", "sessions")
	}
	
	// 确保目录存在
	os.MkdirAll(logDir, 0755)
	
	return &Logger{
		logDir: logDir,
	}
}

// NewSession 创建新会话
func (l *Logger) NewSession(target string) *SessionLog {
	l.sessionID = fmt.Sprintf("session_%s", time.Now().Format("20060102_150405"))
	
	l.session = &SessionLog{
		ID:           l.sessionID,
		Target:       target,
		StartTime:    time.Now(),
		Status:       "running",
		Results:      []PhaseResult{},
		History:      []HistoryEntry{},
		Logs:         []LogEntry{},
		CurrentPhase: "信息收集",
	}
	
	// 创建日志文件
	logPath := l.GetLogPath()
	l.file, _ = os.Create(logPath)
	
	l.Log(LevelInfo, "init", "", "会话开始", true, "")
	l.Save()
	
	return l.session
}

// LoadSession 加载已有会话
func (l *Logger) LoadSession(sessionID string) (*SessionLog, error) {
	logPath := filepath.Join(l.logDir, sessionID+".json")
	
	data, err := os.ReadFile(logPath)
	if err != nil {
		return nil, fmt.Errorf("读取会话文件失败: %w", err)
	}
	
	var session SessionLog
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("解析会话文件失败: %w", err)
	}
	
	l.sessionID = sessionID
	l.session = &session
	
	// 以追加模式打开日志文件
	l.file, _ = os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND, 0644)
	
	return l.session, nil
}

// GetSession 获取当前会话
func (l *Logger) GetSession() *SessionLog {
	return l.session
}

// Log 记录日志
func (l *Logger) Log(level LogLevel, phase, tool, message string, success bool, output string) {
	if l.session == nil {
		return
	}
	
	levelStr := "INFO"
	switch level {
	case LevelDebug:
		levelStr = "DEBUG"
	case LevelWarn:
		levelStr = "WARN"
	case LevelError:
		levelStr = "ERROR"
	}
	
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     levelStr,
		Phase:     phase,
		Action:    message,
		Tool:      tool,
		Message:   message,
		Success:   success,
		Output:    output,
	}
	
	l.session.Logs = append(l.session.Logs, entry)
	
	// 控制台输出
	timeStr := entry.Timestamp.Format("15:04:05")
	icon := "✅"
	if !success {
		icon = "❌"
	}
	fmt.Printf("[%s] [%s] %s %s: %s\n", timeStr, levelStr, icon, tool, message)
}

// RecordResult 记录执行结果
func (l *Logger) RecordResult(phase, tool string, success bool, output, summary string) {
	if l.session == nil {
		return
	}
	
	result := PhaseResult{
		Phase:   phase,
		Tool:    tool,
		Success: success,
		Output:  output,
		Summary: summary,
	}
	l.session.Results = append(l.session.Results, result)
	
	l.Log(LevelInfo, phase, tool, summary, success, "")
	l.Save()
}

// RecordHistory 记录历史
func (l *Logger) RecordHistory(action, tool, result, summary string) {
	if l.session == nil {
		return
	}
	
	entry := HistoryEntry{
		Action:  action,
		Tool:    tool,
		Result:  result,
		Summary: summary,
	}
	l.session.History = append(l.session.History, entry)
	l.Save()
}

// SetNextAction 设置下一步操作（用于断点恢复）
func (l *Logger) SetNextAction(action string) {
	if l.session == nil {
		return
	}
	l.session.NextAction = action
	l.Save()
}

// SetCurrentPhase 设置当前阶段
func (l *Logger) SetCurrentPhase(phase string) {
	if l.session == nil {
		return
	}
	l.session.CurrentPhase = phase
	l.Save()
}

// Complete 标记会话完成
func (l *Logger) Complete() {
	if l.session == nil {
		return
	}
	
	now := time.Now()
	l.session.EndTime = &now
	l.session.Status = "completed"
	l.Log(LevelInfo, "end", "", "会话完成", true, "")
	l.Save()
	
	if l.file != nil {
		l.file.Close()
	}
}

// Interrupt 标记会话中断
func (l *Logger) Interrupt() {
	if l.session == nil {
		return
	}
	
	now := time.Now()
	l.session.EndTime = &now
	l.session.Status = "interrupted"
	l.Log(LevelWarn, "interrupt", "", "会话中断", false, "")
	l.Save()
	
	if l.file != nil {
		l.file.Close()
	}
}

// Save 保存会话到文件
func (l *Logger) Save() error {
	if l.session == nil {
		return nil
	}
	
	logPath := l.GetLogPath()
	data, err := json.MarshalIndent(l.session, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化会话失败: %w", err)
	}
	
	return os.WriteFile(logPath, data, 0644)
}

// GetLogPath 获取日志文件路径
func (l *Logger) GetLogPath() string {
	return filepath.Join(l.logDir, l.sessionID+".json")
}

// GetSessionID 获取会话ID
func (l *Logger) GetSessionID() string {
	return l.sessionID
}

// ListSessions 列出所有会话
func ListSessions(logDir string) ([]SessionLog, error) {
	if logDir == "" {
		homeDir, _ := os.UserHomeDir()
		logDir = filepath.Join(homeDir, ".fcapital", "sessions")
	}
	
	files, err := os.ReadDir(logDir)
	if err != nil {
		return nil, err
	}
	
	var sessions []SessionLog
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			data, err := os.ReadFile(filepath.Join(logDir, file.Name()))
			if err != nil {
				continue
			}
			
			var session SessionLog
			if err := json.Unmarshal(data, &session); err != nil {
				continue
			}
			
			sessions = append(sessions, session)
		}
	}
	
	// 按时间倒序排列
	for i := 0; i < len(sessions)/2; i++ {
		sessions[i], sessions[len(sessions)-1-i] = sessions[len(sessions)-1-i], sessions[i]
	}
	
	return sessions, nil
}

// DeleteSession 删除会话
func DeleteSession(logDir, sessionID string) error {
	if logDir == "" {
		homeDir, _ := os.UserHomeDir()
		logDir = filepath.Join(homeDir, ".fcapital", "sessions")
	}
	
	logPath := filepath.Join(logDir, sessionID+".json")
	return os.Remove(logPath)
}
