package web

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type RunEvents struct {
	Status func(info RunInfo)
	Log    func(entry LogEntry)
}

type RunHandle struct {
	ID     string
	Cancel context.CancelFunc
	Done   <-chan struct{}
	Logger *slog.Logger
}

type DocEntry struct {
	ID         string
	Name       string
	SourcePath string

	lastRun    *RunInfo
	activeRun  *RunHandle
	recentLogs []LogEntry
	mu         sync.RWMutex

	logFanout   []func(LogEntry)
	logFanoutMu sync.RWMutex

	statusSubs   []func(RunInfo)
	statusSubsMu sync.RWMutex

	eventsFn func(RunEvents)
}

func (d *DocEntry) LastRun() *RunInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.lastRun
}

func (d *DocEntry) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.activeRun != nil
}

func (d *DocEntry) ActiveRunID() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.activeRun == nil {
		return ""
	}
	return d.activeRun.ID
}

func (d *DocEntry) DocStatus() DocStatus {
	d.mu.RLock()
	lastRun := d.lastRun
	activeRun := d.activeRun
	d.mu.RUnlock()

	status := DocStatus{
		ID:         d.ID,
		Name:       d.Name,
		SourcePath: d.SourcePath,
		LastRun:    lastRun,
		IsRunning:  activeRun != nil,
	}
	if activeRun != nil {
		status.ActiveRunID = activeRun.ID
	}
	return status
}

func (d *DocEntry) publishStatus(info RunInfo) {
	d.mu.Lock()
	if info.Status == RunStatusCompleted || info.Status == RunStatusFailed || info.Status == RunStatusCancelled {
		d.lastRun = &info
		d.activeRun = nil
	} else {
		newLast := d.lastRun
		if newLast == nil {
			newLast = &info
		}
		d.lastRun = &info
		if info.Status == RunStatusRunning && d.activeRun == nil {
			d.activeRun = &RunHandle{ID: info.ID}
		}
		_ = newLast
	}
	d.mu.Unlock()

	d.statusSubsMu.RLock()
	cbs := d.statusSubs
	d.statusSubsMu.RUnlock()
	for _, cb := range cbs {
		cb(info)
	}
}

func (d *DocEntry) publishLog(entry LogEntry) {
	d.mu.Lock()
	d.recentLogs = append(d.recentLogs, entry)
	if len(d.recentLogs) > 256 {
		d.recentLogs = append([]LogEntry(nil), d.recentLogs[len(d.recentLogs)-256:]...)
	}
	d.mu.Unlock()

	d.logFanoutMu.RLock()
	cbs := d.logFanout
	d.logFanoutMu.RUnlock()
	for _, cb := range cbs {
		cb(entry)
	}
}

// RecentLogs returns a copy of the buffered log entries for a specific run.
func (d *DocEntry) RecentLogs(runID string) []LogEntry {
	d.mu.RLock()
	defer d.mu.RUnlock()

	out := make([]LogEntry, 0, len(d.recentLogs))
	for _, entry := range d.recentLogs {
		if entry.RunID == runID {
			out = append(out, entry)
		}
	}
	return out
}

func (d *DocEntry) onStatusChange(cb func(RunInfo)) func() {
	d.statusSubsMu.Lock()
	d.statusSubs = append(d.statusSubs, cb)
	d.statusSubsMu.Unlock()
	return func() {
		d.statusSubsMu.Lock()
		defer d.statusSubsMu.Unlock()
		for i, f := range d.statusSubs {
			if equalStatusFunc(f, cb) {
				d.statusSubs = append(d.statusSubs[:i], d.statusSubs[i+1:]...)
				break
			}
		}
	}
}

func equalStatusFunc(a, b func(RunInfo)) bool {
	return fmt.Sprintf("%p", a) == fmt.Sprintf("%p", b)
}

func (d *DocEntry) onLog(cb func(LogEntry)) func() {
	d.logFanoutMu.Lock()
	d.logFanout = append(d.logFanout, cb)
	d.logFanoutMu.Unlock()
	return func() {
		d.logFanoutMu.Lock()
		defer d.logFanoutMu.Unlock()
		for i, f := range d.logFanout {
			if equalFunc(f, cb) {
				d.logFanout = append(d.logFanout[:i], d.logFanout[i+1:]...)
				break
			}
		}
	}
}

func equalFunc(a, b func(LogEntry)) bool {
	return fmt.Sprintf("%p", a) == fmt.Sprintf("%p", b)
}

func (d *DocEntry) SubscribeLog(cb func(LogEntry)) (done func()) {
	return d.onLog(cb)
}

func (r *Registry) LookupByRunID(runID string) (*DocEntry, bool) {
	r.mu.RLock()
	docID, ok := r.runToDoc[runID]
	r.mu.RUnlock()
	if !ok {
		return nil, false
	}
	r.mu.RLock()
	entry, ok := r.docs[docID]
	r.mu.RUnlock()
	return entry, ok
}

func (r *Registry) BindRunToDoc(runID, docID string) {
	r.mu.Lock()
	r.runToDoc[runID] = docID
	r.mu.Unlock()
}

func (d *DocEntry) startRun(ctx context.Context, id string, cancel context.CancelFunc, logger *slog.Logger) *RunHandle {
	d.mu.Lock()
	if d.activeRun != nil {
		d.mu.Unlock()
		return nil
	}
	d.activeRun = &RunHandle{
		ID:     id,
		Cancel: cancel,
		Done:   ctx.Done(),
		Logger: logger,
	}
	d.mu.Unlock()
	return d.activeRun
}

func (d *DocEntry) completeRun() {
	d.mu.Lock()
	d.activeRun = nil
	d.mu.Unlock()
}

type Registry struct {
	docs     map[string]*DocEntry
	flowIDs  map[string]string
	runToDoc map[string]string
	mu       sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		docs:     make(map[string]*DocEntry),
		flowIDs:  make(map[string]string),
		runToDoc: make(map[string]string),
	}
}

func (r *Registry) RegisterDoc(id, name, sourcePath string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.docs[id]; !ok {
		r.docs[id] = &DocEntry{
			ID:         id,
			Name:       name,
			SourcePath: sourcePath,
		}
	}
	r.flowIDs[sourcePath] = id
}

func (r *Registry) DocStatus(id string) (DocStatus, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.docs[id]
	if !ok {
		return DocStatus{}, false
	}
	return entry.DocStatus(), true
}

// LookupDocByID returns the registered document entry for a flow ID.
func (r *Registry) LookupDocByID(id string) (*DocEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.docs[id]
	return entry, ok
}

func (r *Registry) ListDocs() []DocStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]DocStatus, 0, len(r.docs))
	for _, entry := range r.docs {
		out = append(out, entry.DocStatus())
	}
	return out
}

func (r *Registry) RegisterRun(runID, docID string) *DocEntry {
	r.mu.RLock()
	entry, ok := r.docs[docID]
	r.mu.RUnlock()
	if !ok {
		return nil
	}
	return entry
}

func (r *Registry) LookupBySourcePath(sourcePath string) *DocEntry {
	r.mu.RLock()
	id, ok := r.flowIDs[sourcePath]
	r.mu.RUnlock()
	if !ok {
		return nil
	}
	r.mu.RLock()
	entry, ok := r.docs[id]
	r.mu.RUnlock()
	if !ok {
		return nil
	}
	return entry
}
