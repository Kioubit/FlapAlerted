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
	"strings"
	"time"
)

var moduleName = "mod_webhook"

// stringSlice implements flag.Value to allow multiple string values for a flag
type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

var webhookUrlsStart stringSlice
var webhookUrlsEnd stringSlice
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
	flag.Var(&webhookUrlsStart, "webhookUrlStart", "Optional webhook URL for when a flap event is detected (start); can be specified multiple times")
	flag.Var(&webhookUrlsEnd, "webhookUrlEnd", "Optional webhook URL for when a flap event is detected (end); can be specified multiple times")
	webhookTimeout = flag.Duration("webhookTimeout", 10*time.Second, "Timeout for webhook HTTP requests")
	webhookInstanceName = flag.String("webhookInstanceName", "", "Optional webhook instance name to set as X-Instance-Name")

	monitor.RegisterModule(&monitor.Module{
		Name:                     moduleName,
		OnRegisterEventCallbacks: registerEventCallbacks,
	})
}

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})).With("module", moduleName)

func registerEventCallbacks() (callbackStart, callbackEnd func(event monitor.FlapEvent)) {
	start := logFlapStart
	end := logFlapEnd
	if len(webhookUrlsStart) == 0 {
		start = nil
	}
	if len(webhookUrlsEnd) == 0 {
		end = nil
	}
	return start, end
}

func logFlapStart(f monitor.FlapEvent) {
	for _, url := range webhookUrlsStart {
		callWebHook(url, f)
	}
}

func logFlapEnd(f monitor.FlapEvent) {
	for _, url := range webhookUrlsEnd {
		callWebHook(url, f)
	}
}

func callWebHook(URL string, f monitor.FlapEvent) {
	if URL == "" {
		return
	}
	l := logger.With("url", URL, "prefix", f.Prefix)
	eventJSON, err := json.Marshal(f)
	if err != nil {
		l.Error("Marshalling flap information failed", "error", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), *webhookTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, URL, bytes.NewReader(eventJSON))
	if err != nil {
		l.Error("Failed to create webhook request", "error", err, "url", URL)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "FlapAlerted-Webhook")

	if *webhookInstanceName != "" {
		req.Header.Set("X-Instance-Name", *webhookInstanceName)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		l.Error("Failed to send webhook", "error", err, "url", URL)
		return
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if resp.StatusCode != 200 {
		l.Error("Webhook returned error status", "url", URL, "status", resp.StatusCode)
	}
}
