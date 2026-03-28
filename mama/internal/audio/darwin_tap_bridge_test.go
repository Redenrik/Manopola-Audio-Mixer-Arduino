//go:build darwin && cgo

package audio

import (
	"errors"
	"testing"
	"time"
)

type fakeDarwinTapRuntime struct {
	supported        bool
	outputRoute      darwinTapOutputRoute
	restoreCapable   bool
	defaultOutputErr error
	createErr        error
	listener         *fakeDarwinTapOutputListener

	createCalls []*fakeDarwinTapRuntimeSession
}

type fakeDarwinTapRuntimeSession struct {
	target         darwinTapTarget
	outputRoute    darwinTapOutputRoute
	volume         int
	muted          bool
	restoreCapable bool
	closed         bool
}

type fakeDarwinTapOutputListener struct {
	generation uint64
}

func (f *fakeDarwinTapRuntime) Supported() bool {
	return f.supported
}

func (f *fakeDarwinTapRuntime) DefaultOutputRoute() (darwinTapOutputRoute, error) {
	if f.defaultOutputErr != nil {
		return darwinTapOutputRoute{}, f.defaultOutputErr
	}
	return f.outputRoute, nil
}

func (f *fakeDarwinTapRuntime) CreateSession(target darwinTapTarget, outputRoute darwinTapOutputRoute, volume int, muted bool) (darwinTapRuntimeSession, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	session := &fakeDarwinTapRuntimeSession{
		target:         target,
		outputRoute:    outputRoute,
		volume:         clampPercent(volume),
		muted:          muted,
		restoreCapable: f.restoreCapable,
	}
	f.createCalls = append(f.createCalls, session)
	return session, nil
}

func (f *fakeDarwinTapRuntime) StartDefaultOutputListener() (darwinTapOutputListener, error) {
	if f.listener == nil {
		return nil, nil
	}
	return f.listener, nil
}

func (s *fakeDarwinTapRuntimeSession) SetGain(volume int) error {
	s.volume = clampPercent(volume)
	return nil
}

func (s *fakeDarwinTapRuntimeSession) SetMuted(muted bool) error {
	s.muted = muted
	return nil
}

func (s *fakeDarwinTapRuntimeSession) Stats() (darwinTapStats, error) {
	return darwinTapStats{
		Attached:        !s.closed,
		Callbacks:       10,
		Frames:          2048,
		RenderCallbacks: 11,
		RenderFrames:    2048,
		Underruns:       0,
		Overflows:       0,
		RestoreCapable:  s.restoreCapable,
		SampleRate:      48000,
		Channels:        2,
		BitsPerChannel:  32,
		BytesPerFrame:   8,
	}, nil
}

func (s *fakeDarwinTapRuntimeSession) Close() error {
	s.closed = true
	return nil
}

func (s *fakeDarwinTapRuntimeSession) RestoreCapable() bool {
	return s.restoreCapable
}

func (l *fakeDarwinTapOutputListener) Generation() uint64 {
	return l.generation
}

func (l *fakeDarwinTapOutputListener) Close() error {
	return nil
}

func (l *fakeDarwinTapOutputListener) Trigger() {
	l.generation++
}

func TestDarwinTapBridgeReconcileRestoresRestartedSession(t *testing.T) {
	runtime := &fakeDarwinTapRuntime{
		supported:      true,
		outputRoute:    darwinTapOutputRoute{DeviceUID: "BuiltInSpeakerDevice", StreamIndex: 0},
		restoreCapable: false,
	}
	bridge := newDefaultDarwinTapBridge(runtime)

	const stateKey = "bundle:com.spotify.client"
	first := darwinTapTarget{
		StateKey:      stateKey,
		RuntimeKey:    "process:10",
		ProcessObject: 10,
		BundleID:      "com.spotify.client",
		Name:          "Spotify",
	}

	if err := bridge.EnsureTap(first); err != nil {
		t.Fatalf("EnsureTap(first) error: %v", err)
	}
	if err := bridge.SetGain(stateKey, 55); err != nil {
		t.Fatalf("SetGain error: %v", err)
	}
	if err := bridge.SetMuted(stateKey, true); err != nil {
		t.Fatalf("SetMuted error: %v", err)
	}

	if err := bridge.Reconcile(nil); err != nil {
		t.Fatalf("Reconcile(nil) error: %v", err)
	}
	state := bridge.ReadVirtualState(stateKey)
	if state.Attached {
		t.Fatalf("state.Attached=%t want false after exit", state.Attached)
	}
	if len(runtime.createCalls) != 1 {
		t.Fatalf("createCalls=%d want 1 before restart", len(runtime.createCalls))
	}
	if !runtime.createCalls[0].closed {
		t.Fatalf("first tap session should have been closed after exit")
	}

	restarted := darwinTapTarget{
		StateKey:      stateKey,
		RuntimeKey:    "process:20",
		ProcessObject: 20,
		BundleID:      "com.spotify.client",
		Name:          "Spotify",
	}
	if err := bridge.Reconcile([]darwinTapTarget{restarted}); err != nil {
		t.Fatalf("Reconcile(restarted) error: %v", err)
	}

	state = bridge.ReadVirtualState(stateKey)
	if !state.Attached {
		t.Fatalf("state.Attached=%t want true after restart", state.Attached)
	}
	if state.Volume != 55 || !state.Muted {
		t.Fatalf("state=%+v want volume=55 muted=true", state)
	}
	if len(runtime.createCalls) != 2 {
		t.Fatalf("createCalls=%d want 2 after restart", len(runtime.createCalls))
	}
	second := runtime.createCalls[1]
	if second.target.ProcessObject != 20 {
		t.Fatalf("recreated process object=%d want 20", second.target.ProcessObject)
	}
	if second.volume != 55 || !second.muted {
		t.Fatalf("recreated session volume=%d muted=%t want 55,true", second.volume, second.muted)
	}
}

func TestDarwinTapBridgeGainAndMuteFanOutAcrossRuntimeSessions(t *testing.T) {
	runtime := &fakeDarwinTapRuntime{
		supported:   true,
		outputRoute: darwinTapOutputRoute{DeviceUID: "BuiltInSpeakerDevice", StreamIndex: 0},
	}
	bridge := newDefaultDarwinTapBridge(runtime)

	const stateKey = "bundle:com.spotify.client"
	targets := []darwinTapTarget{
		{
			StateKey:      stateKey,
			RuntimeKey:    "process:10",
			ProcessObject: 10,
			BundleID:      "com.spotify.client",
			Name:          "Spotify",
		},
		{
			StateKey:      stateKey,
			RuntimeKey:    "process:11",
			ProcessObject: 11,
			BundleID:      "com.spotify.client",
			Name:          "Spotify",
		},
	}
	for _, target := range targets {
		if err := bridge.EnsureTap(target); err != nil {
			t.Fatalf("EnsureTap(%s) error: %v", target.RuntimeKey, err)
		}
	}

	if err := bridge.SetGain(stateKey, 63); err != nil {
		t.Fatalf("SetGain error: %v", err)
	}
	if err := bridge.SetMuted(stateKey, true); err != nil {
		t.Fatalf("SetMuted error: %v", err)
	}

	if len(runtime.createCalls) != 2 {
		t.Fatalf("createCalls=%d want 2", len(runtime.createCalls))
	}
	for i, session := range runtime.createCalls {
		if session.volume != 63 || !session.muted {
			t.Fatalf("session[%d] volume=%d muted=%t want 63,true", i, session.volume, session.muted)
		}
	}

	state := bridge.ReadVirtualState(stateKey)
	if !state.Attached || state.Volume != 63 || !state.Muted {
		t.Fatalf("state=%+v want attached=true volume=63 muted=true", state)
	}
	stats, err := bridge.Stats(stateKey)
	if err != nil {
		t.Fatalf("Stats error: %v", err)
	}
	if stats.MatchCount != 2 {
		t.Fatalf("MatchCount=%d want 2", stats.MatchCount)
	}
	if stats.Callbacks != 20 || stats.RenderFrames != 4096 {
		t.Fatalf("stats=%+v want aggregated callbacks/render frames", stats)
	}
}

func TestDarwinTapBridgeRebuildsOnOutputDeviceChange(t *testing.T) {
	listener := &fakeDarwinTapOutputListener{}
	runtime := &fakeDarwinTapRuntime{
		supported:      true,
		outputRoute:    darwinTapOutputRoute{DeviceUID: "BuiltInSpeakerDevice", StreamIndex: 0},
		restoreCapable: true,
		listener:       listener,
	}
	bridge := newDefaultDarwinTapBridge(runtime)

	target := darwinTapTarget{
		StateKey:      "bundle:com.discord.Discord",
		RuntimeKey:    "process:42",
		ProcessObject: 42,
		BundleID:      "com.discord.Discord",
		Name:          "Discord",
	}
	if err := bridge.EnsureTap(target); err != nil {
		t.Fatalf("EnsureTap error: %v", err)
	}
	if err := bridge.SetGain(target.StateKey, 63); err != nil {
		t.Fatalf("SetGain error: %v", err)
	}
	if err := bridge.SetMuted(target.StateKey, true); err != nil {
		t.Fatalf("SetMuted error: %v", err)
	}

	time.Sleep(150 * time.Millisecond)
	runtime.outputRoute = darwinTapOutputRoute{DeviceUID: "USBHeadset", StreamIndex: 1}
	listener.Trigger()

	deadline := time.Now().Add(2 * time.Second)
	for len(runtime.createCalls) != 2 && time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
	}
	if len(runtime.createCalls) != 2 {
		t.Fatalf("createCalls=%d want 2 after device switch", len(runtime.createCalls))
	}
	if !runtime.createCalls[0].closed {
		t.Fatalf("first session should have been closed on device switch")
	}
	if runtime.createCalls[1].outputRoute.DeviceUID != "USBHeadset" || runtime.createCalls[1].outputRoute.StreamIndex != 1 {
		t.Fatalf("output route=%+v want USBHeadset/1", runtime.createCalls[1].outputRoute)
	}
	if runtime.createCalls[1].volume != 63 || !runtime.createCalls[1].muted {
		t.Fatalf("recreated session volume=%d muted=%t want 63,true", runtime.createCalls[1].volume, runtime.createCalls[1].muted)
	}

	state := bridge.ReadVirtualState(target.StateKey)
	if state.Volume != 63 || !state.Muted || !state.Attached {
		t.Fatalf("state=%+v want volume=63 muted=true attached=true", state)
	}
}

func TestDarwinTapBridgeReleaseTapClosesAllRuntimeHandlesForLogicalState(t *testing.T) {
	runtime := &fakeDarwinTapRuntime{
		supported:   true,
		outputRoute: darwinTapOutputRoute{DeviceUID: "BuiltInSpeakerDevice", StreamIndex: 0},
	}
	bridge := newDefaultDarwinTapBridge(runtime)

	targets := []darwinTapTarget{
		{
			StateKey:      "bundle:com.spotify.client",
			RuntimeKey:    "process:10",
			ProcessObject: 10,
			BundleID:      "com.spotify.client",
		},
		{
			StateKey:      "bundle:com.spotify.client",
			RuntimeKey:    "process:11",
			ProcessObject: 11,
			BundleID:      "com.spotify.client",
		},
	}
	for _, target := range targets {
		if err := bridge.EnsureTap(target); err != nil {
			t.Fatalf("EnsureTap(%s) error: %v", target.RuntimeKey, err)
		}
	}
	if err := bridge.ReleaseTap("bundle:com.spotify.client"); err != nil {
		t.Fatalf("ReleaseTap error: %v", err)
	}
	for i, session := range runtime.createCalls {
		if !session.closed {
			t.Fatalf("session[%d] should be closed", i)
		}
	}
	state := bridge.ReadVirtualState("bundle:com.spotify.client")
	if state.Attached || state.Wanted {
		t.Fatalf("state=%+v want detached and unwanted", state)
	}
}

func TestDarwinTapBridgeRekeysRestoreCapableHandle(t *testing.T) {
	runtime := &fakeDarwinTapRuntime{
		supported:      true,
		outputRoute:    darwinTapOutputRoute{DeviceUID: "BuiltInSpeakerDevice", StreamIndex: 0},
		restoreCapable: true,
	}
	bridge := newDefaultDarwinTapBridge(runtime)

	first := darwinTapTarget{
		StateKey:      "bundle:com.spotify.client",
		RuntimeKey:    "process:10",
		ProcessObject: 10,
		BundleID:      "com.spotify.client",
	}
	if err := bridge.EnsureTap(first); err != nil {
		t.Fatalf("EnsureTap(first) error: %v", err)
	}
	if err := bridge.Reconcile(nil); err != nil {
		t.Fatalf("Reconcile(nil) error: %v", err)
	}
	restarted := darwinTapTarget{
		StateKey:      "bundle:com.spotify.client",
		RuntimeKey:    "process:20",
		ProcessObject: 20,
		BundleID:      "com.spotify.client",
	}
	if err := bridge.Reconcile([]darwinTapTarget{restarted}); err != nil {
		t.Fatalf("Reconcile(restarted) error: %v", err)
	}
	if len(runtime.createCalls) != 1 {
		t.Fatalf("createCalls=%d want 1 when restore-capable handle is reused", len(runtime.createCalls))
	}
	state := bridge.ReadVirtualState("bundle:com.spotify.client")
	if !state.Attached {
		t.Fatalf("state.Attached=%t want true", state.Attached)
	}
}

func TestDarwinTapBridgeClassifiesTypedErrors(t *testing.T) {
	target := darwinTapTarget{
		StateKey:      "bundle:com.spotify.client",
		RuntimeKey:    "process:10",
		ProcessObject: 10,
		BundleID:      "com.spotify.client",
		Name:          "Spotify",
	}

	t.Run("permission", func(t *testing.T) {
		runtime := &fakeDarwinTapRuntime{
			supported:   true,
			outputRoute: darwinTapOutputRoute{DeviceUID: "BuiltInSpeakerDevice", StreamIndex: 0},
			createErr:       errors.New("permission denied: create process tap"),
		}
		bridge := newDefaultDarwinTapBridge(runtime)
		err := bridge.EnsureTap(target)
		if !errors.Is(err, ErrPermissionDenied) {
			t.Fatalf("errors.Is(permission) = false, err=%v", err)
		}
	})

	t.Run("output_device", func(t *testing.T) {
		runtime := &fakeDarwinTapRuntime{
			supported:        true,
			defaultOutputErr: errors.New("output device unavailable: no default output"),
		}
		bridge := newDefaultDarwinTapBridge(runtime)
		err := bridge.EnsureTap(target)
		if !errors.Is(err, ErrOutputDeviceUnavailable) {
			t.Fatalf("errors.Is(output device unavailable) = false, err=%v", err)
		}
	})
}
