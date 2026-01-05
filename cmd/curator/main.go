package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/runner"
	"github.com/bakkerme/curator-ai/internal/runner/factory"
	"gopkg.in/yaml.v3"
)

func main() {
	configPath := flag.String("config", getenv("CURATOR_CONFIG", "curator.yaml"), "path to curator document")
	flowID := flag.String("flow-id", getenv("FLOW_ID", "flow-1"), "flow identifier")
	runOnce := flag.Bool("run-once", getenvBool("RUN_ONCE", false), "run once and exit")
	allowPartial := flag.Bool("allow-partial", getenvBool("ALLOW_PARTIAL_SOURCE_ERRORS", false), "continue if a source fails")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

	doc, err := loadDocument(*configPath)
	if err != nil {
		log.Fatalf("failed to load document: %v", err)
	}

	factory := factory.NewFromEnv()
	flow, err := doc.ParseToFlowWithFactory(factory)
	if err != nil {
		log.Fatalf("failed to parse flow: %v", err)
	}
	flow.ID = *flowID

	runner := runner.NewWithConfig(logger, runner.Config{AllowPartialSourceErrors: *allowPartial})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if *runOnce {
		_, err := runner.RunOnce(ctx, flow)
		if err != nil {
			log.Fatalf("run failed: %v", err)
		}
		return
	}

	if err := runner.Start(ctx, flow); err != nil {
		log.Fatalf("failed to start runner: %v", err)
	}

	<-ctx.Done()
	time.Sleep(200 * time.Millisecond)
}

func loadDocument(path string) (*config.CuratorDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc config.CuratorDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse curator document: %w", err)
	}
	return &doc, nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "TRUE", "True", "yes", "YES", "Yes":
		return true
	default:
		return false
	}
}
