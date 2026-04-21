package errors

import "fmt"

// AppError 应用程序错误类型
type AppError struct {
	Code    string // 错误代码
	Message string // 错误消息
	Cause   error  // 原始错误
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

// 预定义错误代码
const (
	CodeToolNotFound   = "TOOL_NOT_FOUND"
	CodeToolNotReady   = "TOOL_NOT_READY"
	CodeToolInstall    = "TOOL_INSTALL_FAILED"
	CodeScanFailed     = "SCAN_FAILED"
	CodeInvalidInput   = "INVALID_INPUT"
	CodeConfigError    = "CONFIG_ERROR"
	CodeWorkflowError  = "WORKFLOW_ERROR"
	CodeTimeout        = "TIMEOUT"
)

// NewError 创建应用错误
func NewError(code, message string, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// ToolNotFoundError 工具未找到错误
func ToolNotFoundError(toolName string, cause error) *AppError {
	return &AppError{
		Code:    CodeToolNotFound,
		Message: fmt.Sprintf("tool '%s' not found", toolName),
		Cause:   cause,
	}
}

// ToolNotReadyError 工具未就绪错误
func ToolNotReadyError(toolName string) *AppError {
	return &AppError{
		Code:    CodeToolNotReady,
		Message: fmt.Sprintf("tool '%s' is not installed. Run 'fcapital deps install %s'", toolName, toolName),
	}
}

// ToolInstallError 工具安装失败错误
func ToolInstallError(toolName string, cause error) *AppError {
	return &AppError{
		Code:    CodeToolInstall,
		Message: fmt.Sprintf("failed to install tool '%s'", toolName),
		Cause:   cause,
	}
}

// ScanFailedError 扫描失败错误
func ScanFailedError(toolName, target string, cause error) *AppError {
	return &AppError{
		Code:    CodeScanFailed,
		Message: fmt.Sprintf("%s scan failed on %s", toolName, target),
		Cause:   cause,
	}
}

// InvalidInputError 无效输入错误
func InvalidInputError(message string) *AppError {
	return &AppError{
		Code:    CodeInvalidInput,
		Message: message,
	}
}

// ConfigError 配置错误
func ConfigError(message string, cause error) *AppError {
	return &AppError{
		Code:    CodeConfigError,
		Message: message,
		Cause:   cause,
	}
}

// WorkflowError 工作流错误
func WorkflowError(workflowName, stepID string, cause error) *AppError {
	return &AppError{
		Code:    CodeWorkflowError,
		Message: fmt.Sprintf("workflow '%s' failed at step '%s'", workflowName, stepID),
		Cause:   cause,
	}
}

// TimeoutError 超时错误
func TimeoutError(operation string, timeout string) *AppError {
	return &AppError{
		Code:    CodeTimeout,
		Message: fmt.Sprintf("operation '%s' timed out after %s", operation, timeout),
	}
}

// Wrap 包装错误
func Wrap(message string, cause error) error {
	if cause == nil {
		return fmt.Errorf("%s", message)
	}
	return fmt.Errorf("%s: %w", message, cause)
}
