// Package orchestrator provides phase-based workflow orchestration.
// It manages the execution of penetration testing phases with AI decision points.
package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Mal-Suen/fcapital/internal/core/ai"
	appcontext "github.com/Mal-Suen/fcapital/internal/core/context"
	"github.com/Mal-Suen/fcapital/internal/core/scheduler"
)

// Phase defines the interface for a testing phase.
type Phase interface {
	// ID returns the phase identifier.
	ID() string

	// Name returns the human-readable phase name.
	Name() string

	// Description returns the phase description.
	Description() string

	// Execute runs the phase.
	Execute(ctx context.Context, input *PhaseInput) (*PhaseOutput, error)

	// CanSkip returns whether this phase can be skipped.
	CanSkip() bool

	// Dependencies returns the IDs of phases this phase depends on.
	Dependencies() []string
}

// PhaseInput represents input to a phase.
type PhaseInput struct {
	Target      string                  `json:"target"`
	Context     *appcontext.Context     `json:"context"`
	PrevResults map[string]*PhaseOutput `json:"prev_results"`
	Options     map[string]interface{}  `json:"options"`
}

// PhaseOutput represents output from a phase.
type PhaseOutput struct {
	PhaseID    string                 `json:"phase_id"`
	PhaseName  string                 `json:"phase_name"`
	Status     string                 `json:"status"` // completed, failed, skipped
	Findings   map[string]interface{} `json:"findings"`
	NextPhase  string                 `json:"next_phase,omitempty"`
	SkipPhases []string               `json:"skip_phases,omitempty"`
	Error      string                 `json:"error,omitempty"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
}

// RunOptions represents options for running a workflow.
type RunOptions struct {
	AutoContinue    bool          `json:"auto_continue"`    // AI决策后自动继续
	ConfirmCritical bool          `json:"confirm_critical"` // 关键操作需要确认
	SkipPhases      []string      `json:"skip_phases"`      // 要跳过的阶段
	Timeout         time.Duration `json:"timeout"`          // 单阶段超时
}

// RunResult represents the result of a workflow run.
type RunResult struct {
	SessionID    string                  `json:"session_id"`
	Target       string                  `json:"target"`
	Status       string                  `json:"status"`
	PhaseResults map[string]*PhaseOutput `json:"phase_results"`
	StartTime    time.Time               `json:"start_time"`
	EndTime      time.Time               `json:"end_time"`
	Error        string                  `json:"error,omitempty"`
}

// Orchestrator manages phase execution.
type Orchestrator struct {
	phases     map[string]Phase
	phaseOrder []string
	aiEngine   *ai.Engine
	ctxMgr     *appcontext.Manager
	scheduler  *scheduler.Scheduler
	logger     Logger
	mu         sync.RWMutex
	paused     bool
	pauseCond  *sync.Cond
	cancel     context.CancelFunc
}

// Logger interface for logging.
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// Option is a functional option for Orchestrator.
type Option func(*Orchestrator)

// WithAIEngine sets the AI engine.
func WithAIEngine(engine *ai.Engine) Option {
	return func(o *Orchestrator) {
		o.aiEngine = engine
	}
}

// WithContextManager sets the context manager.
func WithContextManager(mgr *appcontext.Manager) Option {
	return func(o *Orchestrator) {
		o.ctxMgr = mgr
	}
}

// WithScheduler sets the scheduler.
func WithScheduler(s *scheduler.Scheduler) Option {
	return func(o *Orchestrator) {
		o.scheduler = s
	}
}

// WithLogger sets the logger.
func WithLogger(l Logger) Option {
	return func(o *Orchestrator) {
		o.logger = l
	}
}

// New creates a new Orchestrator.
func New(opts ...Option) *Orchestrator {
	o := &Orchestrator{
		phases:     make(map[string]Phase),
		phaseOrder: make([]string, 0),
		logger:     &noopLogger{},
	}
	// Initialize pause condition with the mutex
	o.pauseCond = sync.NewCond(&sync.Mutex{})
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// RegisterPhase registers a phase.
func (o *Orchestrator) RegisterPhase(phase Phase) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if _, exists := o.phases[phase.ID()]; exists {
		return fmt.Errorf("phase %s already registered", phase.ID())
	}

	o.phases[phase.ID()] = phase
	o.phaseOrder = append(o.phaseOrder, phase.ID())

	return nil
}

// Run executes the workflow.
func (o *Orchestrator) Run(ctx context.Context, target string, opts *RunOptions) (*RunResult, error) {
	if opts == nil {
		opts = &RunOptions{
			AutoContinue:    false,
			ConfirmCritical: true,
			Timeout:         30 * time.Minute,
		}
	}

	// Create cancellable context
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	o.mu.Lock()
	o.cancel = cancel
	o.mu.Unlock()

	result := &RunResult{
		SessionID:    o.ctxMgr.GetContext().Session.ID,
		Target:       target,
		Status:       "running",
		PhaseResults: make(map[string]*PhaseOutput),
		StartTime:    time.Now(),
	}

	// Set target in context
	o.ctxMgr.SetTarget(target)

	// Initialize AI session
	if o.aiEngine != nil {
		initResp, err := o.aiEngine.InitializeSession(ctx,
			o.ctxMgr.GetSystemInfoSummary(),
			o.ctxMgr.GetToolsSummary())
		if err != nil {
			o.logger.Warn("Failed to initialize AI session: %v", err)
		} else {
			o.ctxMgr.AddMessage("assistant", initResp)
			o.logger.Info("AI session initialized")
		}
	}

	// Build execution order (topological sort based on dependencies)
	execOrder, err := o.buildExecutionOrder(opts.SkipPhases)
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, err
	}

	// Execute phases
	for _, phaseID := range execOrder {
		// Check for pause using condition variable (no busy-wait)
		o.pauseCond.L.Lock()
		for o.paused {
			o.pauseCond.Wait()
		}
		o.pauseCond.L.Unlock()

		phase := o.phases[phaseID]
		o.ctxMgr.SetCurrentPhase(phaseID)
		o.logger.Info("Starting phase: %s", phase.Name())

		// Build phase input
		input := &PhaseInput{
			Target:      target,
			Context:     o.ctxMgr.GetContext(),
			PrevResults: result.PhaseResults,
			Options: map[string]interface{}{
				"auto_continue":    opts.AutoContinue,
				"confirm_critical": opts.ConfirmCritical,
				"timeout":          opts.Timeout,
			},
		}

		// Execute phase
		output, err := o.executePhaseWithAI(runCtx, phase, input, opts)
		if err != nil {
			result.Status = "failed"
			result.Error = fmt.Sprintf("Phase %s failed: %v", phaseID, err)
			o.logger.Error("Phase %s failed: %v", phaseID, err)
			return result, err
		}

		result.PhaseResults[phaseID] = output

		// Add to context history
		phaseResult := &appcontext.PhaseResult{
			PhaseID:   output.PhaseID,
			PhaseName: output.PhaseName,
			StartTime: output.StartTime,
			EndTime:   output.EndTime,
			Status:    output.Status,
			Findings:  output.Findings,
		}
		o.ctxMgr.AddPhaseResult(phaseResult)

		// Handle AI-suggested phase skipping
		if len(output.SkipPhases) > 0 {
			for _, skipID := range output.SkipPhases {
				if _, exists := o.phases[skipID]; exists {
					result.PhaseResults[skipID] = &PhaseOutput{
						PhaseID:   skipID,
						PhaseName: o.phases[skipID].Name(),
						Status:    "skipped",
					}
					o.logger.Info("Skipping phase %s as suggested by AI", skipID)
				}
			}
		}

		// Handle AI-suggested next phase
		if output.NextPhase != "" && output.NextPhase != phaseID {
			// Reorder execution if AI suggests a different next phase
			o.logger.Info("AI suggests next phase: %s", output.NextPhase)
		}
	}

	result.Status = "completed"
	result.EndTime = time.Now()
	o.ctxMgr.CompleteSession()

	return result, nil
}

// executePhaseWithAI executes a phase and gets AI analysis.
func (o *Orchestrator) executePhaseWithAI(ctx context.Context, phase Phase, input *PhaseInput, opts *RunOptions) (*PhaseOutput, error) {
	output := &PhaseOutput{
		PhaseID:   phase.ID(),
		PhaseName: phase.Name(),
		StartTime: time.Now(),
		Findings:  make(map[string]interface{}),
	}

	// Execute the phase
	result, err := phase.Execute(ctx, input)
	if err != nil {
		output.Status = "failed"
		output.Error = err.Error()
		output.EndTime = time.Now()
		return output, err
	}

	output.Findings = result.Findings
	output.Status = "completed"
	output.EndTime = time.Now()

	// Get AI analysis if available
	if o.aiEngine != nil {
		decision, err := o.aiEngine.AnalyzePhaseResult(ctx,
			phase.ID(),
			phase.Name(),
			input.Target,
			output.Findings)

		if err != nil {
			o.logger.Warn("AI analysis failed: %v", err)
		} else {
			// Store AI summary
			output.Findings["ai_analysis"] = decision.Analysis
			output.Findings["ai_priority"] = decision.Priority

			// Store next phase suggestion
			if decision.NextPhase != "" {
				output.NextPhase = decision.NextPhase
			}
			if len(decision.SkipPhases) > 0 {
				output.SkipPhases = decision.SkipPhases
			}

			// Add to conversation
			o.ctxMgr.AddMessage("assistant", fmt.Sprintf(
				"Phase %s analysis: %s\nPriority: %s\nNext action: %s",
				phase.Name(), decision.Analysis, decision.Priority, decision.NextAction))

			// If not auto-continue, wait for user confirmation
			if !opts.AutoContinue {
				o.logger.Info("AI suggests: %s", decision.NextAction)
				// In a real implementation, this would prompt the user
			}
		}
	}

	return output, nil
}

// buildExecutionOrder builds the execution order based on dependencies.
func (o *Orchestrator) buildExecutionOrder(skipPhases []string) ([]string, error) {
	// Create skip set
	skipSet := make(map[string]bool)
	for _, id := range skipPhases {
		skipSet[id] = true
	}

	// Build dependency graph
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	for id := range o.phases {
		if skipSet[id] {
			continue
		}
		graph[id] = make([]string, 0)
		inDegree[id] = 0
	}

	for id, phase := range o.phases {
		if skipSet[id] {
			continue
		}
		for _, dep := range phase.Dependencies() {
			if !skipSet[dep] {
				graph[dep] = append(graph[dep], id)
				inDegree[id]++
			}
		}
	}

	// Topological sort (Kahn's algorithm)
	queue := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	result := make([]string, 0)
	for len(queue) > 0 {
		// Pop from queue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Reduce in-degree of neighbors
		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check for cycles
	if len(result) != len(inDegree) {
		return nil, fmt.Errorf("cycle detected in phase dependencies")
	}

	return result, nil
}

// Pause pauses the workflow.
func (o *Orchestrator) Pause() error {
	o.pauseCond.L.Lock()
	defer o.pauseCond.L.Unlock()
	o.paused = true
	return nil
}

// Resume resumes the workflow.
func (o *Orchestrator) Resume() error {
	o.pauseCond.L.Lock()
	defer o.pauseCond.L.Unlock()
	o.paused = false
	o.pauseCond.Broadcast()
	return nil
}

// Stop stops the workflow.
func (o *Orchestrator) Stop() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.cancel != nil {
		o.cancel()
	}
	return nil
}

// GetCurrentPhase returns the current phase.
func (o *Orchestrator) GetCurrentPhase() Phase {
	o.mu.RLock()
	defer o.mu.RUnlock()

	currentID := o.ctxMgr.GetContext().CurrentPhase
	if currentID == "" {
		return nil
	}
	return o.phases[currentID]
}

// GetPhases returns all registered phases.
func (o *Orchestrator) GetPhases() []Phase {
	o.mu.RLock()
	defer o.mu.RUnlock()

	phases := make([]Phase, 0, len(o.phases))
	for _, id := range o.phaseOrder {
		phases = append(phases, o.phases[id])
	}
	return phases
}

// noopLogger is a no-op logger.
type noopLogger struct{}

func (l *noopLogger) Debug(msg string, args ...interface{}) {}
func (l *noopLogger) Info(msg string, args ...interface{})  {}
func (l *noopLogger) Warn(msg string, args ...interface{})  {}
func (l *noopLogger) Error(msg string, args ...interface{}) {}
