package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Server struct {
	registry    *Registry
	mux         *http.ServeMux
	logger      *slog.Logger
	otelCfg     OTelConfig
	server      *http.Server
	triggerRun  func(context.Context, string) (string, time.Time, error)
	runCancel   map[string]context.CancelFunc
	runCancelMu sync.Mutex
}

type OTelConfig struct {
	Enabled     bool
	ServiceName string
	Endpoint    string
	Protocol    string
	Headers     map[string]string
	Insecure    bool
	SampleRatio float64
}

func NewServer(registry *Registry, logger *slog.Logger, otelCfg OTelConfig) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{
		registry:  registry,
		mux:       http.NewServeMux(),
		logger:    logger,
		otelCfg:   otelCfg,
		runCancel: make(map[string]context.CancelFunc),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /api/docs", s.handleListDocs)
	s.mux.HandleFunc("POST /api/docs/{docId}/run", s.handleTriggerRun)
	s.mux.HandleFunc("GET /api/runs/{runId}/logs", s.handleRunLogs)
}

// SetTriggerRunHandler wires the real run launcher into the HTTP server.
func (s *Server) SetTriggerRunHandler(fn func(context.Context, string) (string, time.Time, error)) {
	s.triggerRun = fn
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) Start(ctx context.Context, addr string) error {
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.mux,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("server shutdown error", "error", err)
		}
	}()
	s.logger.Info("web server starting", "addr", addr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen and serve: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, APIResponse{Error: msg})
}

func (s *Server) handleListDocs(w http.ResponseWriter, r *http.Request) {
	docs := s.registry.ListDocs()
	writeJSON(w, http.StatusOK, ListDocsResponse{Docs: docs})
}

func (s *Server) handleTriggerRun(w http.ResponseWriter, r *http.Request) {
	docID := r.PathValue("docId")
	if docID == "" {
		writeError(w, http.StatusBadRequest, "docId is required")
		return
	}

	if _, ok := s.registry.LookupDocByID(docID); !ok {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	if s.triggerRun == nil {
		writeError(w, http.StatusServiceUnavailable, "run trigger is not configured")
		return
	}

	_, cancel := context.WithCancel(r.Context())
	s.runCancelMu.Lock()
	s.runCancel[docID] = cancel
	s.runCancelMu.Unlock()

	runID, started, err := s.triggerRun(r.Context(), docID)
	if err != nil {
		s.logger.Error("run trigger failed", "flow_id", docID, "error", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusAccepted, TriggerRunResponse{
		RunID:     runID,
		FlowID:    docID,
		StartedAt: started,
	})
}

func (s *Server) handleRunLogs(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("runId")
	if runID == "" {
		writeError(w, http.StatusBadRequest, "runId is required")
		return
	}

	docEntry, ok := s.registry.LookupByRunID(runID)
	if !ok {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("websocket upgrade failed", "run_id", runID, "error", err)
		return
	}
	defer func() {
		if err := conn.Close(); err != nil {
			s.logger.Debug("websocket close error", "run_id", runID, "error", err)
		}
	}()

	conn.SetReadLimit(512 * 1024)
	conn.SetWriteDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(appData string) error {
		conn.SetWriteDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()

	logCh := make(chan LogEntry, 64)
	backlog := docEntry.RecentLogs(runID)
	unsubscribe := docEntry.SubscribeLog(func(entry LogEntry) {
		if entry.RunID != runID {
			return
		}
		select {
		case logCh <- entry:
		default:
		}
	})
	defer func() {
		unsubscribe()
		close(logCh)
	}()

	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					s.logger.Debug("websocket read error", "run_id", runID, "error", err)
				}
				return
			}
		}
	}()

	for _, entry := range backlog {
		if err := conn.WriteJSON(entry); err != nil {
			s.logger.Debug("websocket backlog write error", "run_id", runID, "error", err)
			return
		}
	}

	for {
		select {
		case entry, ok := <-logCh:
			if !ok {
				return
			}
			if err := conn.WriteJSON(entry); err != nil {
				s.logger.Debug("websocket write error", "run_id", runID, "error", err)
				return
			}
		case <-pingTicker.C:
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
				s.logger.Debug("websocket ping error", "run_id", runID, "error", err)
				return
			}
		}
	}
}

func (s *Server) NotifyRunStarted(runID, docID string) {
}

func (s *Server) NotifyRunCompleted(runID, docID string, status RunStatus, errMsg string) {
}
