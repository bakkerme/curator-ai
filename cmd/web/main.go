package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/observability/otelx"
	"github.com/bakkerme/curator-ai/internal/runner"
	"github.com/bakkerme/curator-ai/internal/web"
)

func main() {
	env, err := config.LoadEnv()
	if err != nil {
		log.Panicf("failed to load environment: %v", err)
	}

	cDocPath := flag.String("config", env.CuratorDocPath, "path to curator document file or directory")
	webAddr := flag.String("web-addr", envString("WEB_ADDR", "localhost:8080"), "address for the web dashboard server")
	allowPartial := flag.Bool("allow-partial", env.AllowPartialSourceErrors, "continue if a source fails")
	flag.Parse()

	resolvedConfigPath := strings.TrimSpace(*cDocPath)
	if resolvedConfigPath == "" {
		log.Panicf("config path cannot be empty (set -config or CURATOR_DOC_PATH)")
	}
	log.Printf("loading curator document(s) from: %s", resolvedConfigPath)
	log.Printf("web dashboard will listen on: %s", *webAddr)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(shutdownCtx); err != nil {
			logger.Error("failed to shutdown otel", "error", err)
		}
	}()

	registry := web.NewRegistry()
	web.SetGlobalRegistry(registry)

	runSvc := runner.NewWithConfig(logger, runner.Config{AllowPartialSourceErrors: *allowPartial})

	runnerSvc := web.NewRunnerService(registry, runSvc, logger, env)
	if err := runnerSvc.LoadDocs(resolvedConfigPath); err != nil {
		log.Panicf("failed to load curator documents: %v", err)
	}

	webServer := web.NewServer(registry, logger, web.OTelConfig{
		Enabled:     env.OTel.Enabled,
		ServiceName: env.OTel.ServiceName,
		Endpoint:    env.OTel.Endpoint,
		Protocol:    env.OTel.Protocol,
		Headers:     env.OTel.Headers,
		Insecure:    env.OTel.Insecure,
		SampleRatio: env.OTel.SampleRatio,
	})
	webServer.SetTriggerRunHandler(runnerSvc.TriggerRun)

	go func() {
		if err := webServer.Start(ctx, *webAddr); err != nil {
			logger.Error("web server error", "error", err)
		}
	}()

	<-ctx.Done()
	time.Sleep(200 * time.Millisecond)
}

func envString(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
