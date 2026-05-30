package planelet

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// ActionDefinition combines an action's manifest with its handler.
type ActionDefinition struct {
	ID          string
	Label       string
	Icon        string
	IconURL     string
	Description string
	Parameters  []ParameterManifest
	Execute     ActionHandler
}

// TriggerDefinition combines a trigger's manifest with its lifecycle handlers.
type TriggerDefinition struct {
	ID          string
	Label       string
	Icon        string
	IconURL     string
	Description string
	Parameters  []ParameterManifest
	Setup       TriggerSetupHandler
	Webhook     TriggerWebhookHandler
	Cleanup     TriggerCleanupHandler
}

// PluginConfig holds the top-level plugin metadata.
type PluginConfig struct {
	ID          string
	Label       string
	Icon        string
	IconURL     string
	Description string
}

// Plugin is the main SDK entry point. Register actions and triggers,
// then use Handler() or ListenAndServe() to start serving.
type Plugin struct {
	config   PluginConfig
	actions  map[string]*ActionDefinition
	triggers map[string]*TriggerDefinition
	// Preserve insertion order for manifest.
	actionOrder  []string
	triggerOrder []string
}

// NewPlugin creates a new Plugin with the given configuration.
func NewPlugin(config PluginConfig) *Plugin {
	return &Plugin{
		config:   config,
		actions:  make(map[string]*ActionDefinition),
		triggers: make(map[string]*TriggerDefinition),
	}
}

// Action registers an action definition with the plugin.
// Panics if a duplicate action ID is registered.
func (p *Plugin) Action(def ActionDefinition) {
	if _, exists := p.actions[def.ID]; exists {
		panic(fmt.Sprintf("planelet: duplicate action id %q", def.ID))
	}
	if def.Parameters == nil {
		def.Parameters = []ParameterManifest{}
	}
	p.actions[def.ID] = &def
	p.actionOrder = append(p.actionOrder, def.ID)
}

// Trigger registers a trigger definition with the plugin.
// Panics if a duplicate trigger ID is registered.
func (p *Plugin) Trigger(def TriggerDefinition) {
	if _, exists := p.triggers[def.ID]; exists {
		panic(fmt.Sprintf("planelet: duplicate trigger id %q", def.ID))
	}
	if def.Parameters == nil {
		def.Parameters = []ParameterManifest{}
	}
	p.triggers[def.ID] = &def
	p.triggerOrder = append(p.triggerOrder, def.ID)
}

// Manifest builds and returns the plugin manifest.
func (p *Plugin) Manifest() PluginManifest {
	actions := make([]ActionManifest, 0, len(p.actionOrder))
	for _, id := range p.actionOrder {
		def := p.actions[id]
		actions = append(actions, ActionManifest{
			ID:          def.ID,
			Label:       def.Label,
			Icon:        def.Icon,
			IconURL:     def.IconURL,
			Description: def.Description,
			Parameters:  def.Parameters,
		})
	}

	triggers := make([]TriggerManifest, 0, len(p.triggerOrder))
	for _, id := range p.triggerOrder {
		def := p.triggers[id]
		triggers = append(triggers, TriggerManifest{
			ID:          def.ID,
			Label:       def.Label,
			Icon:        def.Icon,
			IconURL:     def.IconURL,
			Description: def.Description,
			Parameters:  def.Parameters,
		})
	}

	return PluginManifest{
		ID:          p.config.ID,
		Label:       p.config.Label,
		Icon:        p.config.Icon,
		IconURL:     p.config.IconURL,
		Description: p.config.Description,
		Actions:     actions,
		Triggers:    triggers,
	}
}

// Handler returns an http.Handler that routes requests according to the
// Planelet protocol.
func (p *Plugin) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /manifest", p.handleManifest)
	mux.HandleFunc("POST /actions/{actionId}/execute", p.handleActionExecute)
	mux.HandleFunc("POST /triggers/{triggerId}/setup", p.handleTriggerSetup)
	mux.HandleFunc("POST /triggers/{triggerId}/webhook", p.handleTriggerWebhook)
	mux.HandleFunc("POST /triggers/{triggerId}/cleanup", p.handleTriggerCleanup)

	return mux
}

// ListenAndServe starts the plugin HTTP server on the given address.
func (p *Plugin) ListenAndServe(addr string) error {
	log.Printf("Planelet plugin %q listening on %s", p.config.Label, addr)
	return http.ListenAndServe(addr, p.Handler())
}

// --- HTTP handlers ---

func (p *Plugin) handleManifest(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, p.Manifest())
}

func (p *Plugin) handleActionExecute(w http.ResponseWriter, r *http.Request) {
	actionID := r.PathValue("actionId")
	action, ok := p.actions[actionID]
	if !ok {
		writeJSON(w, http.StatusNotFound, errorBody(fmt.Sprintf("Unknown action %q", actionID)))
		return
	}

	var req ExecuteActionRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody(err.Error()))
		return
	}
	if req.Parameters == nil {
		req.Parameters = make(map[string]any)
	}

	resp := p.safeExecuteAction(action, req)
	writeJSON(w, http.StatusOK, resp)
}

func (p *Plugin) handleTriggerSetup(w http.ResponseWriter, r *http.Request) {
	triggerID := r.PathValue("triggerId")
	trigger, ok := p.triggers[triggerID]
	if !ok {
		writeJSON(w, http.StatusNotFound, errorBody(fmt.Sprintf("Unknown trigger %q", triggerID)))
		return
	}
	if trigger.Setup == nil {
		writeJSON(w, http.StatusNotFound, errorBody(fmt.Sprintf("Trigger %q does not implement setup", triggerID)))
		return
	}

	var req SetupTriggerRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody(err.Error()))
		return
	}
	if req.Parameters == nil {
		req.Parameters = make(map[string]any)
	}

	resp := p.safeSetupTrigger(trigger, req)
	writeJSON(w, http.StatusOK, resp)
}

func (p *Plugin) handleTriggerWebhook(w http.ResponseWriter, r *http.Request) {
	triggerID := r.PathValue("triggerId")
	trigger, ok := p.triggers[triggerID]
	if !ok {
		writeJSON(w, http.StatusNotFound, errorBody(fmt.Sprintf("Unknown trigger %q", triggerID)))
		return
	}
	if trigger.Webhook == nil {
		writeJSON(w, http.StatusNotFound, errorBody(fmt.Sprintf("Trigger %q does not implement webhook", triggerID)))
		return
	}

	var req HandleTriggerWebhookRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody(err.Error()))
		return
	}
	if req.Parameters == nil {
		req.Parameters = make(map[string]any)
	}

	resp := p.safeWebhookTrigger(trigger, req)
	writeJSON(w, http.StatusOK, resp)
}

func (p *Plugin) handleTriggerCleanup(w http.ResponseWriter, r *http.Request) {
	triggerID := r.PathValue("triggerId")
	trigger, ok := p.triggers[triggerID]
	if !ok {
		writeJSON(w, http.StatusNotFound, errorBody(fmt.Sprintf("Unknown trigger %q", triggerID)))
		return
	}
	if trigger.Cleanup == nil {
		writeJSON(w, http.StatusNotFound, errorBody(fmt.Sprintf("Trigger %q does not implement cleanup", triggerID)))
		return
	}

	var req CleanupTriggerRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorBody(err.Error()))
		return
	}
	if req.Parameters == nil {
		req.Parameters = make(map[string]any)
	}

	resp := p.safeCleanupTrigger(trigger, req)
	writeJSON(w, http.StatusOK, resp)
}

// --- Panic-safe handler wrappers ---

func (p *Plugin) safeExecuteAction(action *ActionDefinition, req ExecuteActionRequest) (resp ExecuteActionResponse) {
	defer func() {
		if r := recover(); r != nil {
			resp = ExecuteActionResponse{Success: false, Error: fmt.Sprintf("handler panic: %v", r)}
		}
	}()
	return action.Execute(req)
}

func (p *Plugin) safeSetupTrigger(trigger *TriggerDefinition, req SetupTriggerRequest) (resp SetupTriggerResponse) {
	defer func() {
		if r := recover(); r != nil {
			resp = SetupTriggerResponse{Success: false, Error: fmt.Sprintf("handler panic: %v", r)}
		}
	}()
	return trigger.Setup(req)
}

func (p *Plugin) safeWebhookTrigger(trigger *TriggerDefinition, req HandleTriggerWebhookRequest) (resp HandleTriggerWebhookResponse) {
	defer func() {
		if r := recover(); r != nil {
			resp = WebhookError(fmt.Sprintf("handler panic: %v", r), http.StatusInternalServerError)
		}
	}()
	return trigger.Webhook(req)
}

func (p *Plugin) safeCleanupTrigger(trigger *TriggerDefinition, req CleanupTriggerRequest) (resp CleanupTriggerResponse) {
	defer func() {
		if r := recover(); r != nil {
			resp = CleanupTriggerResponse{Success: false, Error: fmt.Sprintf("handler panic: %v", r)}
		}
	}()
	return trigger.Cleanup(req)
}

// --- JSON helpers ---

func readJSON(r *http.Request, v any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	if len(body) == 0 {
		return fmt.Errorf("expected a JSON request body")
	}

	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("invalid JSON request body: %w", err)
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type simpleError struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func errorBody(msg string) simpleError {
	return simpleError{Success: false, Error: msg}
}

// Ptr is a helper that returns a pointer to the given value.
// Useful for setting ParameterManifest.Required.
func Ptr[T any](v T) *T {
	return &v
}
