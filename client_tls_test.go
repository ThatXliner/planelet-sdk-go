//go:build integration

package planelet_test

import (
	"crypto/tls"
	"errors"
	"net/http"
	"testing"
	"time"

	planelet "github.com/superplane-dev/planelet-sdk-go"
)

func TestEmudhraTLS(t *testing.T) {
	transport := &http.Transport{}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{
		Transport: transport,
		Timeout:   20 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	t.Run("trusted control", func(t *testing.T) {
		resp, err := client.Get("https://example.com/")
		if err != nil {
			t.Fatalf("trusted HTTPS control failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("trusted HTTPS control returned %s", resp.Status)
		}
		if resp.TLS == nil || len(resp.TLS.VerifiedChains) == 0 {
			t.Fatal("trusted HTTPS control has no verified certificate chain")
		}
		t.Logf("trusted HTTPS control: %s, verified certificate chain", resp.Status)
	})

	for _, host := range []string{
		"dv-g3b-emudhra.we-have-a-problem.com",
		"dv-g3b-aaa-emudhra.we-have-a-problem.com",
	} {
		t.Run(host, func(t *testing.T) {
			sdk, err := planelet.NewSuperPlaneClient(planelet.SuperPlaneClientConfig{
				BaseURL:       "https://" + host,
				IntegrationID: "tls-security-check",
				HTTPClient:    client,
			})
			if err != nil {
				t.Fatal(err)
			}
			err = sdk.EmitEvent(planelet.DirectPluginEvent{EventType: "tls-security-check"})
			var certificateError *tls.CertificateVerificationError
			if !errors.As(err, &certificateError) {
				t.Fatalf("expected TLS certificate rejection; got %v (network errors are inconclusive)", err)
			}
			t.Logf("SDK rejected certificate: %v", certificateError)
		})
	}
}
