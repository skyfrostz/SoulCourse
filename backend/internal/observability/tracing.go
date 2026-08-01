package observability

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"subject-choice-forum/backend/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

type Tracing struct {
	provider *sdktrace.TracerProvider
}

func InitTracing(ctx context.Context, cfg config.Config) (*Tracing, error) {
	if cfg.OTLPEndpoint == "" {
		return &Tracing{}, nil
	}
	if err := cfg.ValidateOTLP(); err != nil {
		return nil, err
	}

	options := []otlptracehttp.Option{otlptracehttp.WithEndpointURL(cfg.OTLPEndpoint)}
	if cfg.OTLPInsecure {
		options = append(options, otlptracehttp.WithInsecure())
	} else {
		tlsConfig, err := tlsConfig(cfg.OTLPCertificate)
		if err != nil {
			return nil, err
		}
		options = append(options, otlptracehttp.WithTLSClientConfig(tlsConfig))
	}
	exporter, err := otlptracehttp.New(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("initialize OTLP trace exporter: %w", err)
	}
	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(cfg.OTLPServiceName)))
	if err != nil {
		return nil, fmt.Errorf("initialize tracing resource: %w", err)
	}
	provider := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithResource(res))
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return &Tracing{provider: provider}, nil
}

func (t *Tracing) Enabled() bool {
	return t != nil && t.provider != nil
}

func (t *Tracing) Tracer() trace.Tracer {
	if !t.Enabled() {
		return nil
	}
	return t.provider.Tracer("subject-choice-forum/http")
}

func (t *Tracing) Shutdown(ctx context.Context) error {
	if !t.Enabled() {
		return nil
	}
	return t.provider.Shutdown(ctx)
}

func tlsConfig(certificatePath string) (*tls.Config, error) {
	config := &tls.Config{MinVersion: tls.VersionTLS12}
	if certificatePath == "" {
		return config, nil
	}
	pem, err := os.ReadFile(certificatePath)
	if err != nil {
		return nil, fmt.Errorf("read OTLP CA certificate: %w", err)
	}
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("OTEL_EXPORTER_OTLP_CERTIFICATE contains no valid certificates")
	}
	config.RootCAs = pool
	return config, nil
}
