//go:build darwin && cgo

package audio

import "testing"

type fakeDarwinTapRuntime struct {
	supported       bool
	outputDeviceUID string
	restoreCapable  bool

	createCalls []*fakeDarwinTapRuntimeSession
}

type fakeDarwinTapRuntimeSession struct {
	target         darwinTapTarget
	outputDeviceID string
	volume         int
	muted          bool
	restoreCapable bool
	closed         bool
}

func (f *fakeDarwinTapRuntime) Supported() bool {
	return f.supported
}

func (f *fakeDarwinTapRuntime) DefaultOutputDeviceUID() (string, error) {
	return f.outputDeviceUID, nil
}

func (f *fakeDarwinTapRuntime) CreateSession(target darwinTapTarget, outputDeviceUID string, volume int, muted bool) (darwinTapRuntimeSession, error) {
	session := &fakeDarwinTapRuntimeSession{
		target:         target,
		outputDeviceID: outputDeviceUID,
		volume:         clampPercent(volume),
		muted:          muted,
		restoreCapable: f.restoreCapable,
	}
	f.createCalls = append(f.createCalls, session)
	return session, nil
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
	return darwinTapStats{Attached: !s.closed, Callbacks: 1, Frames: 2048, RestoreCapable: s.restoreCapable}, nil
}

func (s *fakeDarwinTapRuntimeSession) Close() error {
	s.closed = true
	return nil
}

func (s *fakeDarwinTapRuntimeSession) RestoreCapable() bool {
	return s.restoreCapable
}

func TestDarwinTapBridgeReconcileRestoresRestartedSession(t *testing.T) {
	runtime := &fakeDarwinTapRuntime{
		supported:       true,
		outputDeviceUID: "BuiltInSpeakerDevice",
		restoreCapable:  false,
	}
	bridge := newDefaultDarwinTapBridge(runtime)

	const key = "bundle:com.spotify.client"
	first := darwinTapTarget{
		Key:           key,
		ProcessObject: 10,
		BundleID:      "com.spotify.client",
		Name:          "Spotify",
	}

	if err := bridge.EnsureTap(first); err != nil {
		t.Fatalf("EnsureTap(first) error: %v", err)
	}
	if err := bridge.SetGain(key, 55); err != nil {
		t.Fatalf("SetGain error: %v", err)
	}
	if err := bridge.SetMuted(key, true); err != nil {
		t.Fatalf("SetMuted error: %v", err)
	}

	if err := bridge.Reconcile(nil); err != nil {
		t.Fatalf("Reconcile(nil) error: %v", err)
	}
	state := bridge.ReadVirtualState(key)
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
		Key:           key,
		ProcessObject: 20,
		BundleID:      "com.spotify.client",
		Name:          "Spotify",
	}
	if err := bridge.Reconcile([]darwinTapTarget{restarted}); err != nil {
		t.Fatalf("Reconcile(restarted) error: %v", err)
	}

	state = bridge.ReadVirtualState(key)
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

func TestDarwinTapBridgeRebuildsOnOutputDeviceChange(t *testing.T) {
	runtime := &fakeDarwinTapRuntime{
		supported:       true,
		outputDeviceUID: "BuiltInSpeakerDevice",
		restoreCapable:  true,
	}
	bridge := newDefaultDarwinTapBridge(runtime)

	target := darwinTapTarget{
		Key:           "bundle:com.discord.Discord",
		ProcessObject: 42,
		BundleID:      "com.discord.Discord",
		Name:          "Discord",
	}
	if err := bridge.EnsureTap(target); err != nil {
		t.Fatalf("EnsureTap error: %v", err)
	}
	if len(runtime.createCalls) != 1 {
		t.Fatalf("createCalls=%d want 1", len(runtime.createCalls))
	}

	runtime.outputDeviceUID = "USBHeadset"
	if err := bridge.Reconcile([]darwinTapTarget{target}); err != nil {
		t.Fatalf("Reconcile error: %v", err)
	}
	if len(runtime.createCalls) != 2 {
		t.Fatalf("createCalls=%d want 2 after device switch", len(runtime.createCalls))
	}
	if !runtime.createCalls[0].closed {
		t.Fatalf("first session should have been closed on device switch")
	}
	if runtime.createCalls[1].outputDeviceID != "USBHeadset" {
		t.Fatalf("output device=%q want USBHeadset", runtime.createCalls[1].outputDeviceID)
	}
}
