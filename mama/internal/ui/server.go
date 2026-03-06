package ui

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"mama/internal/audio"
	"mama/internal/config"
	"mama/internal/mixer"
	"mama/internal/proto"
	serialx "mama/internal/serial"
)

//go:embed static/*
var staticAssets embed.FS

type Server struct {
	cfgPath            string
	backend            audio.Backend
	mixerService       *mixer.Service
	runtimeSnapshot    func() (config.SerialCfg, mixer.Status, bool)
	listPorts          func() ([]string, error)
	probeProtocolHello func(string, int, time.Duration) (int, error)
}

func New(cfgPath string) *Server {
	return &Server{
		cfgPath:            cfgPath,
		backend:            audio.NewBackend(),
		listPorts:          serialx.ListPorts,
		probeProtocolHello: serialx.ProbeProtocolHello,
	}
}

func (s *Server) SetMixerService(service *mixer.Service) {
	s.mixerService = service
}

func (s *Server) serialRuntimeSnapshot() (config.SerialCfg, mixer.Status, bool) {
	if s.runtimeSnapshot != nil {
		return s.runtimeSnapshot()
	}
	if s.mixerService == nil {
		return config.SerialCfg{}, mixer.Status{}, false
	}
	return s.mixerService.ConfigSnapshot().Serial, s.mixerService.Status(), true
}

func (s *Server) Run(listenAddr string) error {
	return http.ListenAndServe(listenAddr, s.Handler())
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/runtime", s.handleRuntime)
	mux.HandleFunc("/api/mapping-status", s.handleMappingStatus)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/ports", s.handlePorts)
	mux.HandleFunc("/api/port-test", s.handlePortTest)
	mux.HandleFunc("/api/port-autodetect", s.handlePortAutoDetect)
	mux.HandleFunc("/api/targets", s.handleTargets)
	mux.HandleFunc("/api/identify", s.handleIdentify)
	mux.HandleFunc("/api/startup", s.handleStartup)
	return mux
}

type portTestRequest struct {
	Port string `json:"port"`
	Baud int    `json:"baud"`
}

type startupRequest struct {
	Enabled bool `json:"enabled"`
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	b, err := staticAssets.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(b)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	cfgPath := s.cfgPath
	if abs, err := filepath.Abs(cfgPath); err == nil {
		cfgPath = abs
	}
	resp := map[string]any{
		"ok":         true,
		"configPath": cfgPath,
	}
	if s.mixerService != nil {
		resp["runtime"] = s.mixerService.Status()
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleRuntime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	if s.mixerService == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":            true,
			"running":       false,
			"observedKnobs": []int{},
			"status": map[string]any{
				"state":     "inactive",
				"connected": false,
				"message":   "runtime service not configured",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"running":       true,
		"status":        s.mixerService.Status(),
		"metrics":       s.mixerService.MetricsSnapshot(),
		"serial":        s.mixerService.ConfigSnapshot().Serial,
		"observedKnobs": s.mixerService.ObservedKnobs(),
	})
}

type mappingStatusItem struct {
	Knob             int               `json:"knob"`
	Target           config.TargetType `json:"target"`
	Selector         string            `json:"selector,omitempty"`
	Display          string            `json:"display,omitempty"`
	Available        bool              `json:"available"`
	Volume           int               `json:"volume,omitempty"`
	Muted            bool              `json:"muted"`
	SessionCount     int               `json:"sessionCount,omitempty"`
	FallbackEnabled  bool              `json:"fallbackEnabled,omitempty"`
	FallbackTarget   config.TargetType `json:"fallbackTarget,omitempty"`
	FallbackName     string            `json:"fallbackName,omitempty"`
	FallbackToMaster bool              `json:"fallbackToMaster,omitempty"`
	Health           string            `json:"health"`
	HealthMessage    string            `json:"healthMessage,omitempty"`
	Error            string            `json:"error,omitempty"`
}

func (s *Server) handleMappingStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	cfg, err := s.activeConfig()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	items := make([]mappingStatusItem, 0, len(cfg.ActiveMappings()))
	for _, mapping := range cfg.ActiveMappings() {
		selectorToken := mappingSelectorToken(mapping)
		fallbackTarget, fallbackName, fallbackEnabled := mapping.EffectiveFallback()
		item := mappingStatusItem{
			Knob:             mapping.Knob,
			Target:           mapping.Target,
			Selector:         selectorToken,
			Display:          mappingDisplayName(mapping),
			Available:        false,
			FallbackEnabled:  fallbackEnabled,
			FallbackTarget:   fallbackTarget,
			FallbackName:     fallbackName,
			FallbackToMaster: fallbackEnabled && fallbackTarget == config.TargetMasterOut && strings.TrimSpace(fallbackName) == "",
			Health:           "unknown",
		}

		targetState, readErr := s.backend.ReadState(mapping.Target, selectorToken)
		if readErr == nil {
			item.Available = targetState.Available
			item.Volume = targetState.Volume
			item.Muted = targetState.Muted
			item.SessionCount = targetState.SessionCount
		} else {
			item.Error = readErr.Error()
		}
		item.Health, item.HealthMessage = classifyMappingHealth(mapping, item, readErr)

		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Knob < items[j].Knob
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"mappings":      items,
		"activeProfile": cfg.ActiveProfile,
	})
}

func (s *Server) activeConfig() (config.Config, error) {
	if s.mixerService != nil {
		return s.mixerService.ConfigSnapshot(), nil
	}
	return config.Load(s.cfgPath)
}

func mappingSelectorToken(mapping config.Mapping) string {
	if mapping.Target == config.TargetApp && mapping.Selector != nil {
		return string(mapping.Selector.Kind) + ":" + mapping.Selector.Value
	}
	if mapping.Target == config.TargetGroup && len(mapping.Selectors) > 0 {
		if b, err := json.Marshal(mapping.Selectors); err == nil {
			return string(b)
		}
	}
	return strings.TrimSpace(mapping.Name)
}

func mappingDisplayName(mapping config.Mapping) string {
	switch mapping.Target {
	case config.TargetMasterOut:
		if name := strings.TrimSpace(mapping.Name); name != "" {
			return name
		}
		return "System Output"
	case config.TargetMicIn:
		if name := strings.TrimSpace(mapping.Name); name != "" {
			return name
		}
		return "System Microphone"
	case config.TargetLineIn:
		if name := strings.TrimSpace(mapping.Name); name != "" {
			return name
		}
		return "System Line Input"
	case config.TargetApp:
		if mapping.Selector != nil {
			return mappingSelectorForUI(*mapping.Selector)
		}
		if name := strings.TrimSpace(mapping.Name); name != "" {
			return name
		}
		return "Application"
	case config.TargetGroup:
		if len(mapping.Selectors) > 0 {
			parts := make([]string, 0, len(mapping.Selectors))
			for _, selector := range mapping.Selectors {
				parts = append(parts, mappingSelectorForUI(selector))
			}
			return strings.Join(parts, "; ")
		}
		if name := strings.TrimSpace(mapping.Name); name != "" {
			return name
		}
		return "Group"
	default:
		return strings.TrimSpace(mapping.Name)
	}
}

func mappingSelectorForUI(selector config.Selector) string {
	kind := strings.TrimSpace(string(selector.Kind))
	value := strings.TrimSpace(selector.Value)
	if value == "" {
		return ""
	}
	if kind == "" || kind == string(config.SelectorExact) {
		return value
	}
	return kind + ":" + value
}

func classifyMappingHealth(mapping config.Mapping, item mappingStatusItem, readErr error) (string, string) {
	if readErr == nil {
		if item.Available {
			return "ok", "target reachable"
		}
		return "unavailable", "target not reachable right now"
	}

	msg := readErr.Error()
	if errors.Is(readErr, audio.ErrTargetUnavailable) {
		if fallbackTarget, _, fallbackEnabled := mapping.EffectiveFallback(); fallbackEnabled && (mapping.Target == config.TargetApp || mapping.Target == config.TargetGroup) {
			return "fallback", fmt.Sprintf("target unavailable, fallback to %s is enabled", fallbackTarget)
		}
		return "unavailable", "target unavailable"
	}

	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "permission"), strings.Contains(lower, "access denied"), strings.Contains(lower, "coinitialize"):
		return "permission", msg
	case strings.Contains(lower, "unsupported"):
		return "unsupported", msg
	case strings.Contains(lower, "not found"), strings.Contains(lower, "impossibile trovare elemento"):
		return "missing", msg
	default:
		return "error", msg
	}
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := config.Load(s.cfgPath)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, cfg)

	case http.MethodPost:
		var in config.Config
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid json: %v", err)})
			return
		}

		if err := config.Save(s.cfgPath, in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		cfg, err := config.Load(s.cfgPath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if s.mixerService != nil {
			if err := s.mixerService.UpdateConfig(cfg); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		}
		writeJSON(w, http.StatusOK, cfg)

	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handlePorts(w http.ResponseWriter, r *http.Request) {
	ports, err := s.listPorts()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if ports == nil {
		ports = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ports": ports,
	})
}

func (s *Server) handlePortTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var in portTestRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid json: %v", err)})
		return
	}

	in.Port = strings.TrimSpace(in.Port)
	if in.Port == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "port is required"})
		return
	}
	if in.Baud <= 0 {
		in.Baud = config.DefaultBaud
	}

	if runtimeSerial, status, ok := s.serialRuntimeSnapshot(); ok {
		samePort := strings.EqualFold(strings.TrimSpace(runtimeSerial.Port), in.Port)
		sameBaud := runtimeSerial.Baud == in.Baud
		if samePort && sameBaud {
			if status.Connected {
				if status.ProtocolVersion <= 0 {
					writeJSON(w, http.StatusBadRequest, map[string]string{
						"error": "runtime is connected but MAMA protocol handshake is not confirmed yet",
					})
					return
				}
				if !proto.IsProtocolCompatible(status.ProtocolVersion) {
					writeJSON(w, http.StatusBadRequest, map[string]string{
						"error": fmt.Sprintf("protocol mismatch: firmware=%d host=%d", status.ProtocolVersion, proto.HostProtocolVersion),
					})
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{
					"ok":              true,
					"port":            in.Port,
					"baud":            in.Baud,
					"protocolVersion": status.ProtocolVersion,
					"message":         fmt.Sprintf("MAMA runtime already connected on %s @ %d", in.Port, in.Baud),
				})
				return
			}

			msg := status.Error
			if msg == "" {
				msg = status.Message
			}
			if msg == "" {
				msg = "runtime is not connected yet"
			}
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
			return
		}
	}

	version, err := s.probeProtocolHello(in.Port, in.Baud, 0)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"port":            in.Port,
		"baud":            in.Baud,
		"protocolVersion": version,
		"message":         fmt.Sprintf("MAMA device detected (protocol v%d)", version),
	})
}

func (s *Server) handlePortAutoDetect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	baud := config.DefaultBaud
	if rawBaud := strings.TrimSpace(r.URL.Query().Get("baud")); rawBaud != "" {
		v, err := strconv.Atoi(rawBaud)
		if err != nil || v <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid baud query parameter"})
			return
		}
		baud = v
	}

	ports, err := s.listPorts()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if ports == nil {
		ports = []string{}
	}

	attempts := make([]map[string]any, 0, len(ports))
	if runtimeSerial, status, ok := s.serialRuntimeSnapshot(); ok && status.Connected && status.ProtocolVersion > 0 && proto.IsProtocolCompatible(status.ProtocolVersion) {
		runtimePort := strings.TrimSpace(runtimeSerial.Port)
		if runtimePort != "" {
			containsRuntimePort := false
			for _, port := range ports {
				if strings.EqualFold(strings.TrimSpace(port), runtimePort) {
					containsRuntimePort = true
					break
				}
			}
			if !containsRuntimePort {
				ports = append([]string{runtimePort}, ports...)
			}
			attempts = append(attempts, map[string]any{
				"port":            runtimePort,
				"ok":              true,
				"source":          "runtime",
				"protocolVersion": status.ProtocolVersion,
			})
			writeJSON(w, http.StatusOK, map[string]any{
				"ok":              true,
				"ports":           ports,
				"attempts":        attempts,
				"detected":        runtimePort,
				"baud":            runtimeSerial.Baud,
				"protocolVersion": status.ProtocolVersion,
				"message":         fmt.Sprintf("MAMA runtime already connected on %s @ %d", runtimePort, runtimeSerial.Baud),
			})
			return
		}
	}

	for _, port := range ports {
		attempt := map[string]any{
			"port": port,
		}
		version, probeErr := s.probeProtocolHello(port, baud, 0)
		if probeErr != nil {
			attempt["ok"] = false
			attempt["error"] = probeErr.Error()
			attempts = append(attempts, attempt)
			continue
		}

		attempt["ok"] = true
		attempt["protocolVersion"] = version
		attempts = append(attempts, attempt)
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":              true,
			"ports":           ports,
			"attempts":        attempts,
			"detected":        port,
			"baud":            baud,
			"protocolVersion": version,
			"message":         fmt.Sprintf("MAMA device detected on %s (protocol v%d)", port, version),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"ports":    ports,
		"attempts": attempts,
		"detected": "",
		"baud":     baud,
		"message":  "no MAMA device detected on available serial ports",
	})
}

func (s *Server) handleTargets(w http.ResponseWriter, r *http.Request) {
	knownTargets := config.KnownTargets()
	availableTargets := knownTargets
	if provider, ok := s.backend.(interface{ SupportedTargetTypes() []config.TargetType }); ok {
		if supported := provider.SupportedTargetTypes(); len(supported) > 0 {
			availableTargets = supported
		}
	}
	discovered, err := s.backend.ListTargets()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	known := make([]string, 0, len(knownTargets))
	for _, t := range knownTargets {
		known = append(known, string(t))
	}

	available := make([]string, 0, len(availableTargets))
	for _, t := range availableTargets {
		available = append(available, string(t))
	}

	supported := make([]string, 0, len(discovered))
	for _, t := range discovered {
		supported = append(supported, t.ID)
	}

	resp := map[string]any{
		"known":          known,
		"availableTypes": available,
		"supported":      supported,
		"discovered":     discovered,
	}
	if diagnostics := summarizeTargetsDiagnostics(discovered); diagnostics != nil {
		resp["diagnostics"] = diagnostics
	}

	writeJSON(w, http.StatusOK, resp)
}

type targetsDiagnostics struct {
	AppSessions *appSessionDiagnostics `json:"appSessions,omitempty"`
}

type appSessionDiagnostics struct {
	Total               int `json:"total"`
	WithCapabilities    int `json:"withCapabilities,omitempty"`
	UnknownCapabilities int `json:"unknownCapabilities,omitempty"`
	VolumeSupported     int `json:"volumeSupported,omitempty"`
	MuteSupported       int `json:"muteSupported,omitempty"`
	FullySupported      int `json:"fullySupported,omitempty"`
	WriteUnsupported    int `json:"writeUnsupported,omitempty"`
}

func summarizeTargetsDiagnostics(discovered []audio.DiscoveredTarget) *targetsDiagnostics {
	summary := appSessionDiagnostics{}
	hasAppTargets := false

	for _, target := range discovered {
		if target.Type != config.TargetApp {
			continue
		}
		hasAppTargets = true
		summary.Total++
		if target.Capabilities == nil {
			continue
		}
		summary.WithCapabilities++

		volumeKnown := target.Capabilities.Volume != nil
		volumeSupported := volumeKnown && *target.Capabilities.Volume
		muteKnown := target.Capabilities.Mute != nil
		muteSupported := muteKnown && *target.Capabilities.Mute

		if volumeSupported {
			summary.VolumeSupported++
		}
		if muteSupported {
			summary.MuteSupported++
		}
		if volumeSupported && muteSupported {
			summary.FullySupported++
		}
		if (volumeKnown && !volumeSupported) || (muteKnown && !muteSupported) {
			summary.WriteUnsupported++
		}
	}

	if !hasAppTargets {
		return nil
	}
	summary.UnknownCapabilities = summary.Total - summary.WithCapabilities

	return &targetsDiagnostics{
		AppSessions: &summary,
	}
}

func (s *Server) handleStartup(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		status, err := startupStatusForConfig(s.cfgPath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, status)
	case http.MethodPost:
		var in startupRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid json: %v", err)})
			return
		}
		status, err := configureStartupForConfig(s.cfgPath, in.Enabled)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if !status.Supported {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": status.Message})
			return
		}
		writeJSON(w, http.StatusOK, status)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handleIdentify(w http.ResponseWriter, r *http.Request) {
	if s.mixerService != nil {
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		stream, cancel := s.mixerService.Subscribe(256)
		defer cancel()

		heartbeat := time.NewTicker(5 * time.Second)
		defer heartbeat.Stop()

		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-stream:
				if !ok {
					return
				}
				if err := writeSSE(w, ev); err != nil {
					return
				}
				flusher.Flush()
			case <-heartbeat.C:
				if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}

	port := strings.TrimSpace(r.URL.Query().Get("port"))
	if port == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing port query parameter"})
		return
	}

	baud := config.DefaultBaud
	if rawBaud := strings.TrimSpace(r.URL.Query().Get("baud")); rawBaud != "" {
		v, err := strconv.Atoi(rawBaud)
		if err != nil || v <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid baud query parameter"})
			return
		}
		baud = v
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}

	reader, err := serialx.Open(port, baud)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	defer func() { _ = reader.Close() }()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Closing the port on client disconnect interrupts scanner reads promptly.
	go func() {
		<-ctx.Done()
		_ = reader.Close()
	}()

	lines := make(chan string, 32)
	errC := make(chan error, 1)
	go func() {
		errC <- reader.ReadLines(ctx, lines)
	}()

	heartbeat := time.NewTicker(5 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errC:
			if err != nil && !errors.Is(err, context.Canceled) && err.Error() != "serial closed" {
				_ = writeSSE(w, map[string]any{
					"type":    "error",
					"message": err.Error(),
				})
				flusher.Flush()
			}
			return
		case line, ok := <-lines:
			if !ok {
				return
			}

			resp := map[string]any{
				"type": "raw",
				"raw":  line,
			}

			if ev, err := proto.ParseLine(line); err == nil {
				resp["type"] = "event"
				switch ev.Kind {
				case proto.EventEncoderDelta:
					resp["kind"] = "encoder_delta"
				case proto.EventButtonPress:
					resp["kind"] = "button_press"
				case proto.EventProtocolHello:
					resp["kind"] = "protocol_hello"
					resp["protocolVersion"] = ev.ProtocolVersion
				default:
					resp["kind"] = "unknown"
				}
				resp["knobId"] = ev.KnobID
				resp["delta"] = ev.Delta
			}

			if err := writeSSE(w, resp); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeSSE(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", b)
	return err
}
