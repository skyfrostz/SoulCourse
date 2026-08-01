package observability

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/proto"
	"subject-choice-forum/backend/internal/config"
)

func TestInitTracingDisabled(t *testing.T) {
	tracing, err := InitTracing(context.Background(), config.Config{})
	if err != nil || tracing.Enabled() || tracing.Tracer() != nil {
		t.Fatalf("disabled tracing should be a no-op: enabled=%v err=%v", tracing.Enabled(), err)
	}
	if err := tracing.Shutdown(context.Background()); err != nil {
		t.Fatalf("disabled shutdown: %v", err)
	}
}

func TestTLSConfigRejectsInvalidCertificate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, []byte("not a certificate"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := tlsConfig(path); err == nil {
		t.Fatal("expected invalid certificate error")
	}
}

func TestTLSConfigDefaultsAndMissingCA(t *testing.T) {
	cfg, err := tlsConfig("")
	if err != nil || cfg.MinVersion != tls.VersionTLS12 || cfg.RootCAs != nil {
		t.Fatalf("default TLS config = %#v, err=%v", cfg, err)
	}
	_, err = tlsConfig(filepath.Join(t.TempDir(), "missing.pem"))
	if err == nil || !strings.Contains(err.Error(), "read OTLP CA certificate") {
		t.Fatalf("missing CA error = %v", err)
	}
}

func TestInitTracingRejectsInvalidEndpointWithoutLeakingCredentials(t *testing.T) {
	cfg := config.Config{OTLPEndpoint: "https://user:super-secret@collector.invalid/v1/traces", OTLPServiceName: "public-beta"}
	_, err := InitTracing(context.Background(), cfg)
	if err == nil || strings.Contains(err.Error(), "super-secret") || strings.Contains(err.Error(), "user:") {
		t.Fatalf("endpoint error = %v", err)
	}
}

func TestInitTracingExportsToLocalCollectorAndSetsResource(t *testing.T) {
	received := make(chan struct{}, 1)
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := new(bytes.Buffer)
		_, _ = body.ReadFrom(r.Body)
		request := new(tracepb.ExportTraceServiceRequest)
		if err := proto.Unmarshal(body.Bytes(), request); err != nil {
			t.Errorf("decode OTLP request: %v", err)
		}
		if len(request.ResourceSpans) == 0 || request.ResourceSpans[0].Resource.GetAttributes()[0].GetKey() != "service.name" || request.ResourceSpans[0].Resource.GetAttributes()[0].GetValue().GetStringValue() != "public-beta" {
			t.Errorf("resource attributes = %v", request.ResourceSpans)
		}
		if r.Method != http.MethodPost || r.URL.Path != "/v1/traces" {
			t.Errorf("collector request = %s %s", r.Method, r.URL)
		}
		if r.Header.Get("Authorization") != "" || r.Header.Get("X-API-Key") != "" {
			t.Errorf("sensitive auth header was sent")
		}
		w.WriteHeader(http.StatusOK)
		received <- struct{}{}
	}))
	defer collector.Close()

	tracing, err := InitTracing(context.Background(), config.Config{
		OTLPEndpoint: collector.URL + "/v1/traces", OTLPInsecure: true, OTLPServiceName: "public-beta",
	})
	if err != nil || !tracing.Enabled() || tracing.Tracer() == nil {
		t.Fatalf("init tracing: enabled=%v err=%v", tracing.Enabled(), err)
	}
	_, span := tracing.Tracer().Start(context.Background(), "local-test")
	span.End()
	if err := tracing.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("collector did not receive span")
	}
}

func TestInitTracingWithTrustedHTTPSCA(t *testing.T) {
	collector := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer collector.Close()
	certPath := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(certPath, pemEncodeCert(collector.Certificate()), 0o600); err != nil {
		t.Fatal(err)
	}
	tracing, err := InitTracing(context.Background(), config.Config{OTLPEndpoint: collector.URL + "/v1/traces", OTLPServiceName: "tls-test", OTLPCertificate: certPath})
	if err != nil || !tracing.Enabled() {
		t.Fatalf("TLS init: enabled=%v err=%v", tracing.Enabled(), err)
	}
	if err := tracing.Shutdown(context.Background()); err != nil {
		t.Fatalf("TLS shutdown: %v", err)
	}
}

func TestTracingShutdownPropagatesExporterError(t *testing.T) {
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	url := collector.URL
	collector.Close()
	tracing, err := InitTracing(context.Background(), config.Config{OTLPEndpoint: url + "/v1/traces", OTLPInsecure: true, OTLPServiceName: "shutdown-test"})
	if err != nil {
		t.Fatal(err)
	}
	_, span := tracing.Tracer().Start(context.Background(), "will-fail")
	span.End()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := tracing.Shutdown(ctx); err == nil {
		t.Fatal("expected shutdown to report canceled context")
	}
}

func pemEncodeCert(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
}
