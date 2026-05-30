# Planelet SDK for Go

Build Planelet plugin servers for SuperPlane in Go. Define actions and webhook triggers, and the SDK serves the full Planelet protocol over HTTP using only the standard library.

## Install

```sh
go get github.com/superplane-dev/planelet-sdk-go
```

## Quick start

```go
package main

import (
    "fmt"
    planelet "github.com/superplane-dev/planelet-sdk-go"
)

func main() {
    plugin := planelet.NewPlugin(planelet.PluginConfig{
        ID:    "demo",
        Label: "Demo Plugin",
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

    plugin.ListenAndServe(":3000")
}
```

Point the Planelets integration in SuperPlane at `http://host.docker.internal:3000` when SuperPlane is running in Docker, or at the reachable host/port for your environment.

## Webhook trigger

```go
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
        sig := planelet.FirstHeader(req.Request.Headers, "x-provider-signature")
        body := planelet.DecodeRawBodyText(req.Request.RawBodyBase64)

        if sig == "" {
            return planelet.WebhookError("Missing signature", 401)
        }

        return planelet.WebhookEmit("incoming.received", map[string]any{
            "body": body,
        }, nil)
    },
    Cleanup: func(req planelet.CleanupTriggerRequest) planelet.CleanupTriggerResponse {
        return planelet.CleanupTriggerResponse{Success: true}
    },
})
```

Use `Metadata` for provider webhook IDs or setup state. Do not put secrets in the manifest.

## Direct events

Plugins can emit events into SuperPlane without waiting for a third-party webhook:

```go
client, err := planelet.NewSuperPlaneClient(planelet.SuperPlaneClientConfig{
    BaseURL:       "https://superplane.example",
    IntegrationID: "integration-id",
    Token:         os.Getenv("SUPERPLANE_TOKEN"),
})
if err != nil {
    log.Fatal(err)
}

err = client.EmitEvent(planelet.DirectPluginEvent{
    EventType: "build.finished",
    Payload:   map[string]any{"status": "passed"},
})
```

## Using as an http.Handler

If you want to mount the plugin routes on an existing server or add middleware:

```go
handler := plugin.Handler()
http.ListenAndServe(":3000", handler)
```

## Utilities

- `planelet.DecodeRawBody(base64str)` -- decodes the base64 webhook body to `[]byte`
- `planelet.DecodeRawBodyText(base64str)` -- decodes the base64 webhook body to a UTF-8 string
- `planelet.FirstHeader(headers, name)` -- returns the first value for a header (case-insensitive)
- `planelet.Ptr(v)` -- returns a pointer to `v` (useful for `Required: planelet.Ptr(true)`)

## Webhook response constructors

The `HandleTriggerWebhookResponse` is a union type. Use these constructors:

- `planelet.WebhookEmit(eventType, payload, response)` -- emit an event into the workflow
- `planelet.WebhookSkip(reason, response)` -- skip the event (no workflow trigger)
- `planelet.WebhookError(error, status)` -- report a failure

## Plugin API

A Plugin is an HTTP server that exposes a manifest and optional action/trigger endpoints for SuperPlane.

### Routes

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/manifest` | Returns plugin metadata, actions, and triggers |
| `POST` | `/actions/{actionId}/execute` | Executes an action |
| `POST` | `/triggers/{triggerId}/setup` | Registers a webhook with an external service |
| `POST` | `/triggers/{triggerId}/webhook` | Handles an incoming third-party webhook |
| `POST` | `/triggers/{triggerId}/cleanup` | Tears down a webhook registration |

### Required behavior

- Returns `404` for unknown actions or triggers.
- Returns `400` for malformed JSON bodies.
- Handler panics are recovered and returned as `{ success: false, error: "..." }`.
- Keep IDs stable once users may have configured workflows with them.
- Do not put secrets in manifests.
- Use `Metadata` for provider webhook IDs or other setup state.
- Decode `rawBodyBase64` before signature verification.
- If a webhook should not start a workflow, return `WebhookSkip`.

## Requirements

- Go 1.22+ (uses the new `http.ServeMux` routing patterns)
- No external dependencies
