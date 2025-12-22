//go:build !disable_mod_webhook

package webhook

import (
	"FlapAlerted/analyze"
	"FlapAlerted/monitor"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"
)

var (
	webhookUrlsStart    = stringSliceFlag("webhookUrlStart", "Optional webhook URL for when a flap event is detected (start); can be specified multiple times")
	webhookUrlsEnd      = stringSliceFlag("webhookUrlEnd", "Optional webhook URL for when a flap event is detected (end); can be specified multiple times")
	webhookTimeout      = flag.Duration("webhookTimeout", 10*time.Second, "Timeout for webhook HTTP requests")
	webhookInstanceName = flag.String("webhookInstanceName", "", "Optional webhook instance name to set as X-Instance-Name")
)

func stringSliceFlag(name, usage string) *[]string {
	s := &[]string{}
	flag.Func(name, usage, func(val string) error {
		*s = append(*s, val)
		return nil
	})
	return s
}

type Module struct {
	name       string
	logger     *slog.Logger
	httpClient *http.Client
}

func (m *Module) Name() string {
	return m.name
}

func (m *Module) OnStart() bool {
	if len(*webhookUrlsStart) == 0 && len(*webhookUrlsEnd) == 0 {
		return false
	}

	m.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})).With("module", m.Name())
	m.httpClient = &http.Client{}
	return true
}

func (m *Module) OnEvent(f analyze.FlapEvent, isStart bool) {
	if isStart {
		for _, url := range *webhookUrlsStart {
			m.callWebHook(url, f)
		}
	} else {
		for _, url := range *webhookUrlsEnd {
			m.callWebHook(url, f)
		}
	}
}

func (m *Module) callWebHook(URL string, f analyze.FlapEvent) {
	if URL == "" {
		return
	}
	l := m.logger.With("url", URL, "prefix", f.Prefix)
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

	resp, err := m.httpClient.Do(req)
	if err != nil {
		l.Error("Failed to send webhook", "error", err, "url", URL)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != 200 {
		l.Error("Webhook returned error status", "url", URL, "status", resp.StatusCode)
	}
}

func init() {
	monitor.RegisterModule(&Module{
		name: "mod_webhook",
	})
}
