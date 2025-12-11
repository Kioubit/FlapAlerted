//go:build !disable_mod_webhook

package webhook

import (
	"FlapAlerted/monitor"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"
)

var moduleName = "mod_webhook"
var webhookUrlStart *string
var webhookUrlEnd *string
var webhookTimeout *time.Duration
var webhookInstanceName *string

var httpClient = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:       2,
		IdleConnTimeout:    10 * time.Second,
		DisableCompression: false,
	},
}

func init() {
	webhookUrlStart = flag.String("webhookUrlStart", "", "Optional webhook URL for when a flap event is detected (start)")
	webhookUrlEnd = flag.String("webhookUrlEnd", "", "Optional webhook URL for when a flap event is detected (end)")
	webhookTimeout = flag.Duration("webhookTimeout", 10*time.Second, "Timeout for webhook HTTP requests")
	webhookInstanceName = flag.String("webhookInstanceName", "", "Optional webhook instance name to set as X-Instance-Name")

	monitor.RegisterModule(&monitor.Module{
		Name:          moduleName,
		CallbackStart: logFlapStart,
		CallbackEnd:   logFlapEnd,
	})
}

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})).With("module", moduleName)

func logFlapStart(f monitor.FlapEvent) {
	callWebHook(*webhookUrlStart, f)
}

func logFlapEnd(f monitor.FlapEvent) {
	callWebHook(*webhookUrlEnd, f)
}

func callWebHook(URL string, f monitor.FlapEvent) {
	if URL == "" {
		return
	}
	eventJSON, err := json.Marshal(f)
	if err != nil {
		logger.Error("Marshalling flap information failed", "error", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), *webhookTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, URL, bytes.NewReader(eventJSON))
	if err != nil {
		logger.Error("Failed to create webhook request", "error", err, "url", URL)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "FlapAlerted-Webhook")

	if *webhookInstanceName != "" {
		req.Header.Set("X-Instance-Name", *webhookInstanceName)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Error("Failed to send webhook", "error", err, "url", URL)
		return
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if resp.StatusCode != 200 {
		logger.Error("Webhook returned error status", "url", URL, "status", resp.StatusCode)
	}
}
