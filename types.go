package planelet

import "encoding/json"

// ParameterType enumerates the supported parameter types.
type ParameterType = string

const (
	ParamString ParameterType = "string"
	ParamText   ParameterType = "text"
	ParamNumber ParameterType = "number"
	ParamBool   ParameterType = "bool"
	ParamSelect ParameterType = "select"
	ParamObject ParameterType = "object"
)

// ParameterOption represents a selectable option for a "select" parameter.
type ParameterOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// ParameterManifest describes a single parameter in an action or trigger.
type ParameterManifest struct {
	ID          string            `json:"id"`
	Label       string            `json:"label"`
	Type        ParameterType     `json:"type"`
	Description string            `json:"description,omitempty"`
	Required    *bool             `json:"required,omitempty"`
	Default     any               `json:"default,omitempty"`
	Options     []ParameterOption `json:"options,omitempty"`
}

// ActionManifest describes an action exposed by a plugin.
type ActionManifest struct {
	ID          string              `json:"id"`
	Label       string              `json:"label"`
	Icon        string              `json:"icon,omitempty"`
	IconURL     string              `json:"iconUrl,omitempty"`
	Description string              `json:"description,omitempty"`
	Parameters  []ParameterManifest `json:"parameters"`
}

// TriggerManifest describes a trigger exposed by a plugin.
type TriggerManifest struct {
	ID          string              `json:"id"`
	Label       string              `json:"label"`
	Icon        string              `json:"icon,omitempty"`
	IconURL     string              `json:"iconUrl,omitempty"`
	Description string              `json:"description,omitempty"`
	Parameters  []ParameterManifest `json:"parameters"`
}

// PluginManifest is the top-level manifest returned by GET /manifest.
type PluginManifest struct {
	ID          string            `json:"id"`
	Label       string            `json:"label"`
	Icon        string            `json:"icon,omitempty"`
	IconURL     string            `json:"iconUrl,omitempty"`
	Description string            `json:"description,omitempty"`
	Actions     []ActionManifest  `json:"actions"`
	Triggers    []TriggerManifest `json:"triggers"`
}

// --- Action types ---

// ExecuteActionRequest is the JSON body sent to POST /actions/{id}/execute.
type ExecuteActionRequest struct {
	Parameters map[string]any `json:"parameters"`
	Input      any            `json:"input,omitempty"`
}

// ExecuteActionResponse is returned from an action handler.
type ExecuteActionResponse struct {
	Success bool           `json:"success"`
	Data    map[string]any `json:"data,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// --- Trigger setup types ---

// WebhookInfo holds the webhook URL (and optional secret) provided by SuperPlane.
type WebhookInfo struct {
	URL    string `json:"url"`
	Secret string `json:"secret,omitempty"`
}

// SetupTriggerRequest is the JSON body sent to POST /triggers/{id}/setup.
type SetupTriggerRequest struct {
	Parameters map[string]any `json:"parameters"`
	Webhook    WebhookInfo    `json:"webhook"`
}

// SetupTriggerResponse is returned from a trigger setup handler.
type SetupTriggerResponse struct {
	Success  bool           `json:"success"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Error    string         `json:"error,omitempty"`
}

// --- Trigger webhook types ---

// WebhookHTTPRequest represents the forwarded HTTP request from the third party.
type WebhookHTTPRequest struct {
	Method        string              `json:"method"`
	Headers       map[string][]string `json:"headers"`
	Query         map[string][]string `json:"query,omitempty"`
	RawBodyBase64 string              `json:"rawBodyBase64"`
}

// HandleTriggerWebhookRequest is the JSON body sent to POST /triggers/{id}/webhook.
type HandleTriggerWebhookRequest struct {
	Parameters map[string]any     `json:"parameters"`
	Metadata   map[string]any     `json:"metadata,omitempty"`
	Request    WebhookHTTPRequest `json:"request"`
}

// WebhookHTTPResponse is an optional HTTP response to send back to the third party.
type WebhookHTTPResponse struct {
	Status  int               `json:"status,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// HandleTriggerWebhookResponse is a union type representing the three possible
// webhook handler outcomes: emit, skip, or error. Use the constructors
// WebhookEmit, WebhookSkip, and WebhookError to create instances.
type HandleTriggerWebhookResponse struct {
	// Common
	Success bool `json:"success"`

	// Emit fields (Success=true, Emit=true)
	Emit      *bool  `json:"-"`
	EventType string `json:"-"`
	Payload   any    `json:"-"`

	// Skip fields (Success=true, Emit=false)
	Reason string `json:"-"`

	// Shared optional response for emit/skip
	Response *WebhookHTTPResponse `json:"-"`

	// Error fields (Success=false)
	Error  string `json:"-"`
	Status int    `json:"-"`
}

// WebhookEmit creates a HandleTriggerWebhookResponse that emits an event.
func WebhookEmit(eventType string, payload any, response *WebhookHTTPResponse) HandleTriggerWebhookResponse {
	t := true
	return HandleTriggerWebhookResponse{
		Success:   true,
		Emit:      &t,
		EventType: eventType,
		Payload:   payload,
		Response:  response,
	}
}

// WebhookSkip creates a HandleTriggerWebhookResponse that skips (no event emitted).
func WebhookSkip(reason string, response *WebhookHTTPResponse) HandleTriggerWebhookResponse {
	f := false
	return HandleTriggerWebhookResponse{
		Success:  true,
		Emit:     &f,
		Reason:   reason,
		Response: response,
	}
}

// WebhookError creates a HandleTriggerWebhookResponse indicating a failure.
func WebhookError(errMsg string, status int) HandleTriggerWebhookResponse {
	return HandleTriggerWebhookResponse{
		Success: false,
		Error:   errMsg,
		Status:  status,
	}
}

// MarshalJSON implements custom JSON serialization for the webhook response union type.
func (r HandleTriggerWebhookResponse) MarshalJSON() ([]byte, error) {
	if !r.Success {
		// Error case
		m := map[string]any{"success": false, "error": r.Error}
		if r.Status != 0 {
			m["status"] = r.Status
		}
		return json.Marshal(m)
	}

	if r.Emit != nil && *r.Emit {
		// Emit case
		m := map[string]any{
			"success":   true,
			"emit":      true,
			"eventType": r.EventType,
			"payload":   r.Payload,
		}
		if r.Response != nil {
			m["response"] = r.Response
		}
		return json.Marshal(m)
	}

	// Skip case
	m := map[string]any{
		"success": true,
		"emit":    false,
	}
	if r.Reason != "" {
		m["reason"] = r.Reason
	}
	if r.Response != nil {
		m["response"] = r.Response
	}
	return json.Marshal(m)
}

// --- Trigger cleanup types ---

// CleanupTriggerRequest is the JSON body sent to POST /triggers/{id}/cleanup.
type CleanupTriggerRequest struct {
	Parameters map[string]any `json:"parameters"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// CleanupTriggerResponse is returned from a trigger cleanup handler.
type CleanupTriggerResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// --- Direct event ---

// DirectPluginEvent is used to emit events directly to SuperPlane.
type DirectPluginEvent struct {
	EventType string `json:"eventType"`
	Payload   any    `json:"payload"`
}

// --- Handler function types ---

// ActionHandler handles action execution requests.
type ActionHandler func(ExecuteActionRequest) ExecuteActionResponse

// TriggerSetupHandler handles trigger setup requests.
type TriggerSetupHandler func(SetupTriggerRequest) SetupTriggerResponse

// TriggerWebhookHandler handles incoming webhook requests.
type TriggerWebhookHandler func(HandleTriggerWebhookRequest) HandleTriggerWebhookResponse

// TriggerCleanupHandler handles trigger cleanup requests.
type TriggerCleanupHandler func(CleanupTriggerRequest) CleanupTriggerResponse
