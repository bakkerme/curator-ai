package web

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/runner"
	rt "github.com/bakkerme/curator-ai/internal/runner/runtime"
)

type RunnerService struct {
	registry   *Registry
	runner     *runner.Runner
	rootLogger *slog.Logger
	envCfg     config.EnvConfig
	runtimes   map[string]*runtimeCloser
	mu         sync.Mutex
}

type runtimeCloser struct {
	Runtime interface{ Close() error }
}

func NewRunnerService(registry *Registry, runnerInstance *runner.Runner, rootLogger *slog.Logger, envCfg config.EnvConfig) *RunnerService {
	return &RunnerService{
		registry:   registry,
		runner:     runnerInstance,
		rootLogger: rootLogger,
		envCfg:     envCfg,
		runtimes:   make(map[string]*runtimeCloser),
	}
}

func (rs *RunnerService) LoadDocs(docPath string) error {
	loadedDocs, err := config.LoadCuratorDocuments(docPath)
	if err != nil {
		return fmt.Errorf("load curator documents: %w", err)
	}

	for _, loaded := range loadedDocs {
		flowRuntime, err := rt.NewFromEnvConfig(rs.rootLogger, rs.envCfg)
		if err != nil {
			return fmt.Errorf("build runtime for %s: %w", loaded.Path, err)
		}

		flow, err := loaded.Document.ParseToFlowWithRuntime(flowRuntime)
		if err != nil {
			return fmt.Errorf("parse flow %s: %w", loaded.Path, err)
		}

		flowID := flowIDForPath(loaded.Path, loaded.Document)

		rs.mu.Lock()
		rs.runtimes[flowID] = &runtimeCloser{Runtime: flowRuntime}
		rs.mu.Unlock()

		rs.registry.RegisterDoc(flowID, flow.Name, loaded.Path)
		rs.rootLogger.Info("web: registered doc", "flow_id", flowID, "name", flow.Name, "path", loaded.Path)
	}

	return nil
}

func (rs *RunnerService) TriggerRun(ctx context.Context, docID string) (runID string, startedAt time.Time, err error) {
	docEntry, ok := rs.registry.LookupDocByID(docID)
	if !ok || docEntry == nil {
		return "", time.Time{}, fmt.Errorf("document not found: %s", docID)
	}

	rs.mu.Lock()
	rc, ok := rs.runtimes[docID]
	rs.mu.Unlock()
	if !ok {
		return "", time.Time{}, fmt.Errorf("runtime not found for doc: %s", docID)
	}

	flowRuntime, ok := rc.Runtime.(*rt.Runtime)
	if !ok {
		return "", time.Time{}, fmt.Errorf("unexpected runtime type for doc: %s", docID)
	}

	loadedDocs, err := config.LoadCuratorDocuments(docEntry.SourcePath)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("reload doc %s: %w", docEntry.SourcePath, err)
	}
	if len(loadedDocs) == 0 {
		return "", time.Time{}, fmt.Errorf("no document found at path: %s", docEntry.SourcePath)
	}

	flow, err := loadedDocs[0].Document.ParseToFlowWithRuntime(flowRuntime.CloneForFlow())
	if err != nil {
		return "", time.Time{}, fmt.Errorf("parse flow: %w", err)
	}
	flow.ID = docID

	runID = fmt.Sprintf("run-%d", time.Now().UnixNano())
	startedAt = time.Now().UTC()

	logEmitter := NewStructuredLogEmitter(docID, runID)

	go func() {
		runCtx := context.Background()
		logger := rs.rootLogger.With("flow_id", docID, "run_id", runID)
		runLogger := NewRegistryLogHandler(logger, rs.registry, docID, runID)
		runCtx = core.WithLogger(runCtx, runLogger)

		docEntry.publishStatus(RunInfo{
			ID:        runID,
			FlowID:    docID,
			StartedAt: startedAt,
			Status:    RunStatusRunning,
		})
		rs.registry.BindRunToDoc(runID, docID)

		runLogger.Info("run started", "flow_id", docID, "run_id", runID)
		logEmitter.Emit("INFO", "run started", map[string]interface{}{"flow_id": docID, "run_id": runID})

		completedAt := time.Now().UTC()
		result, runErr := rs.runner.RunOnce(runCtx, flow)
		if runErr != nil {
			completedAt = time.Now().UTC()
			docEntry.publishStatus(RunInfo{
				ID:          runID,
				FlowID:      docID,
				StartedAt:   startedAt,
				CompletedAt: &completedAt,
				Status:      RunStatusFailed,
				Error:       runErr.Error(),
			})
			runLogger.Error("run failed", "error", runErr)
			logEmitter.Emit("ERROR", "run failed", map[string]interface{}{"error": runErr.Error()})
			return
		}

		completedAt = time.Now().UTC()
		docEntry.publishStatus(RunInfo{
			ID:          runID,
			FlowID:      docID,
			StartedAt:   startedAt,
			CompletedAt: &completedAt,
			Status:      MapCoreRunStatus(string(result.Status)),
		})
		runLogger.Info("run completed", "status", result.Status)
		logEmitter.Emit("INFO", "run completed", map[string]interface{}{"status": string(result.Status)})
	}()

	return runID, startedAt, nil
}

func (rs *RunnerService) Close() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	var errs []error
	for id, rc := range rs.runtimes {
		if err := rc.Runtime.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close runtime %s: %w", id, err))
		}
	}
	rs.runtimes = nil
	return nil
}

func flowIDForPath(path string, doc *config.CuratorDocument) string {
	if doc != nil && doc.Workflow.Name != "" {
		return slugifyName(doc.Workflow.Name)
	}
	return slugifyName(path)
}

func slugifyName(s string) string {
	var b strings.Builder
	needsDash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			if needsDash && b.Len() > 0 {
				b.WriteByte('-')
			}
			needsDash = false
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		needsDash = true
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "flow"
	}
	return out
}
