package ui

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mama/internal/audio"
	"mama/internal/config"
	"mama/internal/proto"
	serialx "mama/internal/serial"
)

//go:embed static/*
var staticAssets embed.FS

type Server struct {
	cfgPath            string
	backend            audio.Backend
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

func (s *Server) Run(listenAddr string) error {
	return http.ListenAndServe(listenAddr, s.Handler())
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/status", s.handleStatus)
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
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"configPath": s.cfgPath,
	})
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
	discovered, err := s.backend.ListTargets()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	known := make([]string, 0, len(knownTargets))
	for _, t := range knownTargets {
		known = append(known, string(t))
	}

	supported := make([]string, 0, len(discovered))
	for _, t := range discovered {
		supported = append(supported, t.ID)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"known":      known,
		"supported":  supported,
		"discovered": discovered,
	})
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
