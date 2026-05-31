//go:build !disable_mod_history

package history

import (
	"FlapAlerted/analyze"
	"FlapAlerted/monitor"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"math"
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

// keyToFilename converts meta to "timestamp_prefix.json"
// Replacing '/' in prefix with '_' for valid filename
func (m *Module) keyToFilename(meta monitor.HistoricalEventKey) string {
	safePrefix := strings.ReplaceAll(meta.Prefix.String(), "/", "_")
	return fmt.Sprintf("%d_%s.json", meta.Timestamp, safePrefix)
}

func (m *Module) filenameToKey(filename string) (monitor.HistoricalEventKey, error) {
	var key monitor.HistoricalEventKey

	name := strings.TrimSuffix(filename, ".json")
	if name == filename {
		return key, fmt.Errorf("invalid filename %q: missing .json suffix", filename)
	}

	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return key, fmt.Errorf("invalid filename %q", filename)
	}

	timestamp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return key, fmt.Errorf("invalid timestamp in filename %q: %w", filename, err)
	}

	prefixStr := strings.ReplaceAll(parts[1], "_", "/")

	key.Timestamp = timestamp
	key.Prefix, err = netip.ParsePrefix(prefixStr)
	if err != nil {
		return key, fmt.Errorf("invalid prefix in filename %q: %w", filename, err)
	}

	return key, nil
}

func (m *Module) saveToDisk(f analyze.FlapEvent) {
	now := time.Now().Unix()
	key := monitor.HistoricalEventKey{
		Prefix:    f.Prefix,
		Timestamp: now,
	}

	filename := m.keyToFilename(key)
	path := filepath.Join(*historyDir, filename)
	logger := m.logger.With("path", path)

	var avgChangeRate60 = math.NaN()
	if n := len(f.RateSecHistory); n > 0 {
		var rsSum uint32
		for _, rs := range f.RateSecHistory {
			rsSum += uint32(rs)
		}
		avgChangeRate60 = float64(rsSum) / float64(n)
	}

	var avgRate float64
	duration := now - f.FirstSeen
	if duration > 0 {
		avgRate = float64(f.TotalPathChanges) / float64(duration)
	} else {
		avgRate = math.NaN()
	}

	header := monitor.HistoricalEventMeta{
		AvgChangeRate60: avgChangeRate60,
		AvgChangeRate:   avgRate,
	}

	var success = false

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		logger.Error("failed to open file to write", "error", err)
		return
	}
	defer func() {
		_ = file.Close()
		if !success {
			_ = os.Remove(path)
		}
	}()

	enc := json.NewEncoder(file)

	if err = enc.Encode(header); err != nil {
		logger.Error("failed to marshal event header", "error", err)
		return
	}

	if err = enc.Encode(f); err != nil {
		logger.Error("failed to marshal event", "error", err)
		return
	}

	success = true
}

// GetHistoricalEventList implements HistoryProvider
func (m *Module) GetHistoricalEventList() ([]monitor.HistoricalEvent, error) {
	files, err := os.ReadDir(*historyDir)
	if err != nil {
		return nil, err
	}

	var list = make([]monitor.HistoricalEvent, 0)
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

		readFunc := func() (hdr monitor.HistoricalEventMeta, err error) {
			f, err := os.Open(filepath.Join(*historyDir, file.Name()))
			if err != nil {
				return
			}
			defer func() {
				_ = f.Close()
			}()
			reader := bufio.NewReader(f)
			headerBytes, err := reader.ReadSlice('\n')
			if err != nil {
				// ErrBufferFull is not relevant here
				return
			}
			err = json.Unmarshal(headerBytes, &hdr)
			if err != nil {
				return
			}
			return
		}
		hdr, err := readFunc()
		if err != nil {
			continue
		}

		list = append(list, monitor.HistoricalEvent{
			HistoricalEventKey: monitor.HistoricalEventKey{
				Prefix:    prefix,
				Timestamp: ts,
			},
			HistoricalEventMeta: hdr,
		})
	}

	// Sort newest first
	sort.Slice(list, func(i, j int) bool {
		return list[i].Timestamp > list[j].Timestamp
	})

	return list, nil
}

// GetHistoricalEvent implements HistoryProvider
func (m *Module) GetHistoricalEvent(meta monitor.HistoricalEventKey) (f *analyze.FlapEvent, err error) {
	filename := m.keyToFilename(meta)

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

	file, err := root.Open(filename)
	if err != nil {
		err = fmt.Errorf("could not open event file %s: %w", filename, err)
		return
	}
	defer func() {
		_ = file.Close()
	}()

	reader := bufio.NewReader(file)
	_, err = reader.ReadSlice('\n')
	if err != nil {
		// ErrBufferFull is not relevant here
		err = fmt.Errorf("error finding delimiter for second line %s: %w", filename, err)
		return
	}

	dec := json.NewDecoder(reader)

	if err = dec.Decode(&f); err != nil {
		err = fmt.Errorf("failed to unmarshal event: %w", err)
		return
	}

	return
}

// GetHistoricalEventLatest implements HistoryProvider
func (m *Module) GetHistoricalEventLatest(prefix netip.Prefix) (f *analyze.FlapEvent, meta monitor.HistoricalEventKey, err error) {
	list, err := m.GetHistoricalEventList()
	if err != nil {
		err = fmt.Errorf("failed to list history: %w", err)
		return
	}

	// Find the first occurrence (the newest) matching the prefix
	for _, mk := range list {
		if mk.Prefix == prefix {
			meta = mk.HistoricalEventKey
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
	cutoff := now.Add(-*historyMaxAge).Unix()
	var remainingFiles []os.DirEntry

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}

		key, err := m.filenameToKey(f.Name())
		if err != nil {
			continue
		}

		if key.Timestamp < cutoff {
			if err = os.Remove(filepath.Join(*historyDir, f.Name())); err != nil {
				m.logger.Error("failed to remove history file. Will stop saving events.", "path", f.Name(), "error", err)
				m.hasFailed = true
			}
			continue
		}
		remainingFiles = append(remainingFiles, f)
	}

	if len(remainingFiles) > *historyMaxFiles {
		// Sort by time ascending (oldest first)
		sort.Slice(remainingFiles, func(i, j int) bool {
			keyI, errI := m.filenameToKey(remainingFiles[i].Name())
			if errI != nil {
				return true
			}

			keyJ, errJ := m.filenameToKey(remainingFiles[j].Name())
			if errJ != nil {
				return false
			}

			return keyI.Timestamp < keyJ.Timestamp
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
