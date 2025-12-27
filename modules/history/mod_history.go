//go:build !disable_mod_history

package history

import (
	"FlapAlerted/analyze"
	"FlapAlerted/monitor"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	enableHistory   = flag.Bool("historyEnable", false, "Enable flap event history storage")
	historyDir      = flag.String("historyDir", "./flap_history", "Directory to store event JSON files")
	historyMaxAge   = flag.Duration("historyRetention", 24*time.Hour, "How long to keep history files on disk")
	historyMaxFiles = flag.Int("historyMaxCount", 50, "Maximum number of history files to keep")
)

type Module struct {
	name      string
	enabled   bool
	hasFailed bool
	logger    *slog.Logger
}

func (m *Module) Name() string {
	return m.name
}

func (m *Module) OnStart() bool {
	if !*enableHistory {
		return false
	}

	m.logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{})).With("module", m.name)

	if err := os.MkdirAll(*historyDir, 0755); err != nil {
		m.logger.Error("failed to create history directory", "error", err)
		return false
	}

	m.enabled = true
	return true
}

func (m *Module) OnEvent(f analyze.FlapEvent, isStart bool) {
	if isStart || m.hasFailed {
		return
	}
	m.saveToDisk(f)
	m.rotate()
}

// ActiveHistoryProvider implements HistoryProvider
func (m *Module) ActiveHistoryProvider() bool {
	return m.enabled
}

// metaToFilename converts meta to "timestamp_prefix.json"
// Replacing '/' in prefix with '_' for valid filename
func (m *Module) metaToFilename(meta monitor.HistoricalEventMeta) string {
	safePrefix := strings.ReplaceAll(meta.Prefix.String(), "/", "_")
	return fmt.Sprintf("%d_%s.json", meta.Timestamp, safePrefix)
}

func (m *Module) saveToDisk(f analyze.FlapEvent) {
	meta := monitor.HistoricalEventMeta{
		Prefix:    f.Prefix,
		Timestamp: time.Now().Unix(),
	}

	filename := m.metaToFilename(meta)
	path := filepath.Join(*historyDir, filename)

	jsonData, err := json.Marshal(f)
	if err != nil {
		m.logger.Error("failed to marshal flap event", "error", err)
		return
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		m.logger.Error("failed to write history file", "path", path, "error", err)
	}
}

// GetHistoricalEventList implements HistoryProvider
func (m *Module) GetHistoricalEventList() ([]monitor.HistoricalEventMeta, error) {
	files, err := os.ReadDir(*historyDir)
	if err != nil {
		return nil, err
	}

	var list = make([]monitor.HistoricalEventMeta, 0)
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		// Expected format: <timestamp>_<prefix_ip>_<prefix_len>.json
		base := strings.TrimSuffix(file.Name(), ".json")
		parts := strings.SplitN(base, "_", 2)
		if len(parts) < 2 {
			continue
		}

		ts, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}

		prefixStr := strings.ReplaceAll(parts[1], "_", "/")
		prefix, err := netip.ParsePrefix(prefixStr)
		if err != nil {
			continue
		}

		list = append(list, monitor.HistoricalEventMeta{
			Prefix:    prefix,
			Timestamp: ts,
		})
	}

	// Sort newest first
	sort.Slice(list, func(i, j int) bool {
		return list[i].Timestamp > list[j].Timestamp
	})

	return list, nil
}

// GetHistoricalEvent implements HistoryProvider
func (m *Module) GetHistoricalEvent(meta monitor.HistoricalEventMeta) (f *analyze.FlapEvent, err error) {
	filename := m.metaToFilename(meta)

	root, err := os.OpenRoot(*historyDir)
	if err != nil {
		err = fmt.Errorf("could not open event directory: %w", err)
		return
	}
	defer func() {
		_ = root.Close()
	}()

	if _, err = root.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			err = nil
			return
		}
		return
	}

	data, err := root.ReadFile(filename)
	if err != nil {
		err = fmt.Errorf("could not read event file %s: %w", filename, err)
		return
	}

	if err = json.Unmarshal(data, &f); err != nil {
		err = fmt.Errorf("failed to unmarshal event: %w", err)
		return
	}

	return
}

// GetHistoricalEventLatest implements HistoryProvider
func (m *Module) GetHistoricalEventLatest(prefix netip.Prefix) (f *analyze.FlapEvent, meta monitor.HistoricalEventMeta, err error) {
	list, err := m.GetHistoricalEventList()
	if err != nil {
		err = fmt.Errorf("failed to list history: %w", err)
		return
	}

	// Find the first occurrence (the newest) matching the prefix
	for _, meta = range list {
		if meta.Prefix == prefix {
			f, err = m.GetHistoricalEvent(meta)
			return
		}
	}
	return
}

func (m *Module) rotate() {
	files, err := os.ReadDir(*historyDir)
	if err != nil {
		return
	}

	now := time.Now()
	var remainingFiles []os.DirEntry

	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > *historyMaxAge {
			if err = os.Remove(filepath.Join(*historyDir, f.Name())); err != nil {
				m.logger.Error("failed to remove history file. Will stop saving events.", "path", f.Name(), "error", err)
				m.hasFailed = true
			}
			continue
		}
		remainingFiles = append(remainingFiles, f)
	}

	if len(remainingFiles) > *historyMaxFiles {
		// Sort by ModTime ascending (oldest first)
		sort.Slice(remainingFiles, func(i, j int) bool {
			infoI, _ := remainingFiles[i].Info()
			infoJ, _ := remainingFiles[j].Info()
			return infoI.ModTime().Before(infoJ.ModTime())
		})

		toDelete := len(remainingFiles) - *historyMaxFiles
		for i := 0; i < toDelete; i++ {
			if err = os.Remove(filepath.Join(*historyDir, remainingFiles[i].Name())); err != nil {
				m.logger.Error("failed to remove history file. Will stop saving events.", "path", remainingFiles[i].Name(), "error", err)
				m.hasFailed = true
			}
		}
	}
}

func init() {
	monitor.RegisterModule(&Module{
		name: "mod_history",
	})
}
