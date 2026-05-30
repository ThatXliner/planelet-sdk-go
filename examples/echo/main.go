package main

import (
	"fmt"
	"log"
	"os"

	planelet "github.com/superplane-dev/planelet-sdk-go"
)

func main() {
	plugin := planelet.NewPlugin(planelet.PluginConfig{
		ID:          "echo",
		Label:       "Echo Plugin",
		Description: "A simple echo plugin that returns the input message.",
	})

	plugin.Action(planelet.ActionDefinition{
		ID:    "echo",
		Label: "Echo",
		Parameters: []planelet.ParameterManifest{
			{ID: "message", Label: "Message", Type: "string", Required: planelet.Ptr(true)},
		},
		Execute: func(req planelet.ExecuteActionRequest) planelet.ExecuteActionResponse {
			return planelet.ExecuteActionResponse{
				Success: true,
				Data:    map[string]any{"message": req.Parameters["message"]},
			}
		},
	})

	plugin.Trigger(planelet.TriggerDefinition{
		ID:         "incoming",
		Label:      "Incoming Webhook",
		Parameters: []planelet.ParameterManifest{},
		Setup: func(req planelet.SetupTriggerRequest) planelet.SetupTriggerResponse {
			return planelet.SetupTriggerResponse{
				Success:  true,
				Metadata: map[string]any{"webhookUrl": req.Webhook.URL},
			}
		},
		Webhook: func(req planelet.HandleTriggerWebhookRequest) planelet.HandleTriggerWebhookResponse {
			body := planelet.DecodeRawBodyText(req.Request.RawBodyBase64)
			return planelet.WebhookEmit("webhook.received", map[string]any{
				"method": req.Request.Method,
				"body":   body,
			}, nil)
		},
		Cleanup: func(req planelet.CleanupTriggerRequest) planelet.CleanupTriggerResponse {
			return planelet.CleanupTriggerResponse{Success: true}
		},
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	addr := fmt.Sprintf(":%s", port)
	log.Fatal(plugin.ListenAndServe(addr))
}
