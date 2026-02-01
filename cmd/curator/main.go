package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
	"github.com/bakkerme/curator-ai/internal/observability/otelx"
	"github.com/bakkerme/curator-ai/internal/runner"
	"github.com/bakkerme/curator-ai/internal/runner/factory"
)

// namedFlow ties a Flow back to the Curator Document that produced it.
// This is mainly used to improve error messages and logging when multiple docs are loaded.
type namedFlow struct {
	SourcePath string
	Flow       *core.Flow
}

func main() {
	env := config.LoadEnv()

	configPath := flag.String("config", env.CuratorConfigPath, "path to curator document file or directory")
	flowID := flag.String("flow-id", env.FlowID, "flow identifier")
	runOnce := flag.Bool("run-once", env.RunOnce, "run once and exit")
	allowPartial := flag.Bool("allow-partial", env.AllowPartialSourceErrors, "continue if a source fails")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

	loadedDocs, err := config.LoadCuratorDocuments(*configPath)
	if err != nil {
		log.Panicf("failed to load curator documents: %v", err)
	}

	factory := factory.NewFromEnvConfig(logger, env)

	flows := make([]namedFlow, 0, len(loadedDocs))
	seenFlowIDs := map[string]int{}
	for _, loaded := range loadedDocs {
		flow, err := loaded.Document.ParseToFlowWithFactory(factory)
		if err != nil {
			log.Panicf("failed to parse flow (%s): %v", loaded.Path, err)
		}

		if len(loadedDocs) == 1 {
			flow.ID = *flowID
		} else {
			if *flowID != env.FlowID {
				logger.Warn("ignoring -flow-id when -config points at a directory", "flow_id", *flowID)
			}
			flow.ID = uniqueFlowID(defaultFlowID(loaded.Path, loaded.Document), seenFlowIDs)
		}

		flows = append(flows, namedFlow{SourcePath: loaded.Path, Flow: flow})
	}

	r := runner.NewWithConfig(logger, runner.Config{AllowPartialSourceErrors: *allowPartial})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	shutdown, err := otelx.Init(ctx, logger, env.OTel)
	if err != nil {
		log.Panicf("failed to initialize otel: %v", err)
	}
	defer func() {
		if shutdown == nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			logger.Error("failed to shutdown otel", "error", err)
		}
	}()

	if *runOnce {
		var runErr error
		for _, named := range flows {
			_, err := r.RunOnce(ctx, named.Flow)
			if err != nil {
				runErr = errors.Join(runErr, fmt.Errorf("%s (%s): %w", named.Flow.ID, named.SourcePath, err))
			}
		}
		if runErr != nil {
			log.Panicf("run failed: %v", runErr)
		}
		return
	}

	for _, named := range flows {
		if err := r.Start(ctx, named.Flow); err != nil {
			log.Panicf("failed to start runner (%s): %v", named.Flow.ID, err)
		}
	}

	<-ctx.Done()
	time.Sleep(200 * time.Millisecond)
}

// defaultFlowID derives a stable flow ID from the document, primarily using workflow.name
// (falling back to the document filename). This is only used when loading multiple docs.
func defaultFlowID(docPath string, doc *config.CuratorDocument) string {
	if doc != nil && strings.TrimSpace(doc.Workflow.Name) != "" {
		return slugify(doc.Workflow.Name)
	}
	name := filepath.Base(docPath)
	ext := filepath.Ext(name)
	name = strings.TrimSuffix(name, ext)
	return slugify(name)
}

// uniqueFlowID ensures flow IDs are unique within a single Curator process by suffixing duplicates.
func uniqueFlowID(base string, seen map[string]int) string {
	if base == "" {
		base = "flow"
	}
	n := seen[base]
	seen[base] = n + 1
	if n == 0 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, n+1)
}

// slugify converts a string to a conservative identifier suitable for flow IDs:
// - Lowercases
// - Replaces any run of non-letters/non-digits with a single '-'
// - Trims leading/trailing '-'
//
// Examples:
// - "My RSS Flow" => "my-rss-flow"
// - "AI/ML (Daily)" => "ai-ml-daily"
// - "" => "flow"
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	b.Grow(len(s))

	needsDash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			if needsDash && b.Len() > 0 {
				b.WriteByte('-')
			}
			needsDash = false
			b.WriteRune(r)
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
