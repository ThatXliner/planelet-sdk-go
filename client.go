package planelet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// SuperPlaneClientConfig holds the configuration for connecting to SuperPlane.
type SuperPlaneClientConfig struct {
	// BaseURL is the SuperPlane instance URL (e.g. "https://superplane.example").
	BaseURL string

	// IntegrationID is the Planelets integration ID in SuperPlane.
	IntegrationID string

	// Token is an optional bearer token for authentication.
	Token string

	// HTTPClient is an optional custom HTTP client. If nil, http.DefaultClient is used.
	HTTPClient *http.Client
}

// SuperPlaneClient emits events directly to SuperPlane.
type SuperPlaneClient struct {
	baseURL       *url.URL
	integrationID string
	token         string
	httpClient    *http.Client
}

// NewSuperPlaneClient creates a new client for emitting direct events to SuperPlane.
func NewSuperPlaneClient(config SuperPlaneClientConfig) (*SuperPlaneClient, error) {
	u, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("planelet: invalid base URL: %w", err)
	}

	client := config.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	return &SuperPlaneClient{
		baseURL:       u,
		integrationID: config.IntegrationID,
		token:         config.Token,
		httpClient:    client,
	}, nil
}

// EmitEvent sends a DirectPluginEvent to SuperPlane.
func (c *SuperPlaneClient) EmitEvent(event DirectPluginEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("planelet: failed to marshal event: %w", err)
	}

	path := fmt.Sprintf("/api/v1/integrations/%s/events", url.PathEscape(c.integrationID))
	u := *c.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + path
	u.RawQuery = ""
	u.Fragment = ""

	req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("planelet: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("planelet: event request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("planelet: SuperPlane event request failed with status %d", resp.StatusCode)
	}

	return nil
}
