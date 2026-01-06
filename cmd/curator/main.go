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
	"github.com/bakkerme/curator-ai/internal/observability/otelx"
	"github.com/bakkerme/curator-ai/internal/runner"
	"github.com/bakkerme/curator-ai/internal/runner/factory"
	"gopkg.in/yaml.v3"
)

func main() {
	env := config.LoadEnv()

	configPath := flag.String("config", env.CuratorConfigPath, "path to curator document")
	flowID := flag.String("flow-id", env.FlowID, "flow identifier")
	runOnce := flag.Bool("run-once", env.RunOnce, "run once and exit")
	allowPartial := flag.Bool("allow-partial", env.AllowPartialSourceErrors, "continue if a source fails")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

	doc, err := loadDocument(*configPath)
	if err != nil {
		log.Panicf("failed to load document: %v", err)
	}

	factory := factory.NewFromEnvConfig(logger, env)
	flow, err := doc.ParseToFlowWithFactory(factory)
	if err != nil {
		log.Panicf("failed to parse flow: %v", err)
	}
	flow.ID = *flowID

	runner := runner.NewWithConfig(logger, runner.Config{AllowPartialSourceErrors: *allowPartial})

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
		_, err := runner.RunOnce(ctx, flow)
		if err != nil {
			log.Panicf("run failed: %v", err)
		}
		return
	}

	if err := runner.Start(ctx, flow); err != nil {
		log.Panicf("failed to start runner: %v", err)
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
