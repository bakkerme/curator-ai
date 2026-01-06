package otelx

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/bakkerme/curator-ai/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

func Init(ctx context.Context, logger *slog.Logger, cfg config.OTelEnvConfig) (func(context.Context) error, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if !cfg.Enabled {
		return nil, nil
	}

	serviceName := strings.TrimSpace(cfg.ServiceName)
	if serviceName == "" {
		serviceName = "curator-ai"
	}

	sampleRatio := cfg.SampleRatio
	if sampleRatio < 0 {
		sampleRatio = 0
	}
	if sampleRatio > 1 {
		sampleRatio = 1
	}

	exp, err := newTraceExporter(ctx, cfg)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(
		ctx,
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
		resource.WithProcess(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exp, sdktrace.WithBatchTimeout(2*time.Second)),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRatio))),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info(
		"otel initialized",
		"service_name", serviceName,
		"otlp_endpoint", endpointOrDefault(cfg),
		"otlp_protocol", protocolOrDefault(cfg),
		"sample_ratio", sampleRatio,
	)

	return tp.Shutdown, nil
}

func newTraceExporter(ctx context.Context, cfg config.OTelEnvConfig) (*otlptrace.Exporter, error) {
	headers := cfg.Headers
	insecure := cfg.Insecure

	protocol := protocolOrDefault(cfg)
	switch protocol {
	case "http/protobuf":
		opts := []otlptracehttp.Option{}
		endpoint := endpointOrDefault(cfg)
		if strings.Contains(endpoint, "://") {
			opts = append(opts, otlptracehttp.WithEndpointURL(endpoint))
		} else {
			opts = append(opts, otlptracehttp.WithEndpoint(endpoint))
		}
		if insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if len(headers) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(headers))
		}
		return otlptracehttp.New(ctx, opts...)
	case "grpc":
		opts := []otlptracegrpc.Option{}
		endpoint := endpointOrDefault(cfg)
		if strings.Contains(endpoint, "://") {
			u, err := url.Parse(endpoint)
			if err != nil {
				return nil, fmt.Errorf("parse OTEL_EXPORTER_OTLP_ENDPOINT: %w", err)
			}
			endpoint = u.Host
		}
		opts = append(opts, otlptracegrpc.WithEndpoint(endpoint))
		if insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		if len(headers) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(headers))
		}
		return otlptracegrpc.New(ctx, opts...)
	default:
		return nil, fmt.Errorf("unsupported OTEL_EXPORTER_OTLP_PROTOCOL %q (expected grpc or http/protobuf)", protocol)
	}
}

func endpointOrDefault(cfg config.OTelEnvConfig) string {
	if v := strings.TrimSpace(cfg.Endpoint); v != "" {
		return v
	}
	switch protocolOrDefault(cfg) {
	case "http/protobuf":
		return "localhost:4318"
	default:
		return "localhost:4317"
	}
}

func protocolOrDefault(cfg config.OTelEnvConfig) string {
	if v := strings.ToLower(strings.TrimSpace(cfg.Protocol)); v != "" {
		if v == "http" {
			return "http/protobuf"
		}
		return v
	}
	return "grpc"
}
