//go:build darwin && cgo

package audio

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework CoreAudio -framework CoreFoundation -framework Foundation

#include <CoreAudio/CoreAudio.h>
#include <CoreAudio/AudioHardwareTapping.h>
#include <CoreAudio/CATapDescription.h>
#include <CoreFoundation/CoreFoundation.h>
#include <Foundation/Foundation.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

typedef struct {
	AudioObjectID tap_id;
	AudioObjectID aggregate_device_id;
	AudioDeviceIOProcID io_proc_id;
	float gain;
	uint32_t muted;
	uint64_t callbacks;
	uint64_t frames;
	uint32_t process_object_id;
	uint32_t restore_capable;
	char* bundle_id;
	char* output_device_uid;
} mama_tap_session;

static AudioObjectPropertyAddress mama_tap_property_address(AudioObjectPropertySelector selector,
	AudioObjectPropertyScope scope,
	AudioObjectPropertyElement element) {
	AudioObjectPropertyAddress address = {selector, scope, element};
	return address;
}

static void mama_tap_set_error(char** outErr, const char* msg, OSStatus st) {
	if (outErr == NULL) {
		return;
	}
	char buffer[256];
	snprintf(buffer, sizeof(buffer), "%s (osstatus=%d)", msg, (int)st);
	*outErr = strdup(buffer);
}

static void mama_tap_set_plain_error(char** outErr, const char* msg) {
	if (outErr == NULL) {
		return;
	}
	*outErr = strdup(msg);
}

static char* mama_tap_copy_cfstring(CFStringRef value) {
	if (value == NULL) {
		return NULL;
	}
	CFIndex length = CFStringGetLength(value);
	CFIndex maxSize = CFStringGetMaximumSizeForEncoding(length, kCFStringEncodingUTF8) + 1;
	char* buf = (char*)malloc((size_t)maxSize);
	if (buf == NULL) {
		return NULL;
	}
	if (!CFStringGetCString(value, buf, maxSize, kCFStringEncodingUTF8)) {
		free(buf);
		return NULL;
	}
	return buf;
}

static OSStatus mama_tap_ioproc(AudioObjectID inDevice,
	const AudioTimeStamp* inNow,
	const AudioBufferList* inInputData,
	const AudioTimeStamp* inInputTime,
	AudioBufferList* outOutputData,
	const AudioTimeStamp* inOutputTime,
	void* inClientData) {
	mama_tap_session* session = (mama_tap_session*)inClientData;
	if (session == NULL) {
		return noErr;
	}
	session->callbacks += 1;

	if (outOutputData != NULL) {
		for (UInt32 i = 0; i < outOutputData->mNumberBuffers; i++) {
			AudioBuffer* outBuf = &outOutputData->mBuffers[i];
			if (outBuf->mData != NULL && outBuf->mDataByteSize > 0) {
				memset(outBuf->mData, 0, outBuf->mDataByteSize);
			}
		}
	}
	if (inInputData == NULL || outOutputData == NULL) {
		return noErr;
	}

	UInt32 bufferCount = inInputData->mNumberBuffers < outOutputData->mNumberBuffers ? inInputData->mNumberBuffers : outOutputData->mNumberBuffers;
	float gain = session->muted ? 0.0f : session->gain;
	for (UInt32 i = 0; i < bufferCount; i++) {
		const AudioBuffer* inBuf = &inInputData->mBuffers[i];
		AudioBuffer* outBuf = &outOutputData->mBuffers[i];
		if (inBuf->mData == NULL || outBuf->mData == NULL) {
			continue;
		}

		UInt32 bytes = inBuf->mDataByteSize < outBuf->mDataByteSize ? inBuf->mDataByteSize : outBuf->mDataByteSize;
		if (bytes == 0) {
			continue;
		}

		if (gain <= 0.0001f) {
			memset(outBuf->mData, 0, bytes);
		} else if (gain >= 0.9999f) {
			memcpy(outBuf->mData, inBuf->mData, bytes);
		} else {
			const Float32* inSamples = (const Float32*)inBuf->mData;
			Float32* outSamples = (Float32*)outBuf->mData;
			UInt32 sampleCount = bytes / sizeof(Float32);
			for (UInt32 sample = 0; sample < sampleCount; sample++) {
				outSamples[sample] = inSamples[sample] * gain;
			}
		}

		UInt32 channels = inBuf->mNumberChannels > 0 ? inBuf->mNumberChannels : 1;
		session->frames += (uint64_t)((bytes / sizeof(Float32)) / channels);
	}

	return noErr;
}

static int mama_tap_api_available(void) {
	if (@available(macOS 14.2, *)) {
		return 1;
	}
	return 0;
}

static int mama_tap_restore_available(void) {
	if (@available(macOS 26.0, *)) {
		return 1;
	}
	return 0;
}

static int mama_tap_default_output_device_uid(char** outUID, char** outErr) {
	if (outUID == NULL) {
		return 1;
	}
	*outUID = NULL;
	if (outErr != NULL) {
		*outErr = NULL;
	}

	AudioObjectID deviceID = kAudioObjectUnknown;
	UInt32 size = (UInt32)sizeof(deviceID);
	AudioObjectPropertyAddress address = mama_tap_property_address(
		kAudioHardwarePropertyDefaultOutputDevice,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain);
	OSStatus st = AudioObjectGetPropertyData(kAudioObjectSystemObject, &address, 0, NULL, &size, &deviceID);
	if (st != noErr) {
		mama_tap_set_error(outErr, "read default output device", st);
		return 1;
	}
	if (deviceID == kAudioObjectUnknown) {
		mama_tap_set_plain_error(outErr, "default output device unavailable");
		return 1;
	}

	CFStringRef uid = NULL;
	size = (UInt32)sizeof(uid);
	address = mama_tap_property_address(kAudioDevicePropertyDeviceUID, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain);
	st = AudioObjectGetPropertyData(deviceID, &address, 0, NULL, &size, &uid);
	if (st != noErr || uid == NULL) {
		mama_tap_set_error(outErr, "read default output device uid", st);
		return 1;
	}

	char* copied = mama_tap_copy_cfstring(uid);
	CFRelease(uid);
	if (copied == NULL) {
		mama_tap_set_plain_error(outErr, "copy default output device uid");
		return 1;
	}
	*outUID = copied;
	return 0;
}

static int mama_tap_append_cfstring_property(AudioObjectID objectID,
	AudioObjectPropertySelector selector,
	NSString* value,
	char** outErr) {
	AudioObjectPropertyAddress address = mama_tap_property_address(selector, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain);
	UInt32 size = 0;
	OSStatus st = AudioObjectGetPropertyDataSize(objectID, &address, 0, NULL, &size);
	if (st != noErr) {
		mama_tap_set_error(outErr, "read aggregate property size", st);
		return 0;
	}

	NSMutableArray* items = [NSMutableArray array];
	if (size > 0) {
		CFArrayRef listRef = NULL;
		UInt32 listSize = size;
		st = AudioObjectGetPropertyData(objectID, &address, 0, NULL, &listSize, &listRef);
		if (st != noErr) {
			mama_tap_set_error(outErr, "read aggregate property data", st);
			return 0;
		}
		if (listRef != NULL) {
			NSArray* list = CFBridgingRelease(listRef);
			[items addObjectsFromArray:list];
		}
	}

	if (![items containsObject:value]) {
		[items addObject:value];
	}

	CFArrayRef newList = (__bridge CFArrayRef)items;
	UInt32 writeSize = (UInt32)sizeof(newList);
	st = AudioObjectSetPropertyData(objectID, &address, 0, NULL, writeSize, &newList);
	if (st != noErr) {
		mama_tap_set_error(outErr, "write aggregate property data", st);
		return 0;
	}
	return 1;
}

static int mama_tap_wait_for_streams(AudioObjectID deviceID, uint32_t timeoutMillis) {
	const uint32_t sleepMillis = 50;
	uint32_t waited = 0;

	while (waited <= timeoutMillis) {
		AudioObjectPropertyAddress address = mama_tap_property_address(kAudioDevicePropertyStreams, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain);
		UInt32 size = 0;
		OSStatus st = AudioObjectGetPropertyDataSize(deviceID, &address, 0, NULL, &size);
		if (st == noErr && size >= sizeof(AudioObjectID)) {
			UInt32 streamCount = size / sizeof(AudioObjectID);
			AudioObjectID* streamIDs = (AudioObjectID*)malloc(size);
			if (streamIDs == NULL) {
				return 0;
			}
			st = AudioObjectGetPropertyData(deviceID, &address, 0, NULL, &size, streamIDs);
			if (st == noErr) {
				int inputCount = 0;
				int outputCount = 0;
				for (UInt32 i = 0; i < streamCount; i++) {
					UInt32 direction = 0;
					UInt32 directionSize = (UInt32)sizeof(direction);
					AudioObjectPropertyAddress directionAddress = mama_tap_property_address(kAudioStreamPropertyDirection, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain);
					if (AudioObjectGetPropertyData(streamIDs[i], &directionAddress, 0, NULL, &directionSize, &direction) != noErr) {
						continue;
					}
					if (direction == 0) {
						outputCount++;
					} else {
						inputCount++;
					}
				}
				free(streamIDs);
				if (inputCount > 0 && outputCount > 0) {
					return 1;
				}
			} else {
				free(streamIDs);
			}
		}

		usleep(sleepMillis * 1000);
		waited += sleepMillis;
	}

	return 0;
}

static void mama_tap_cleanup_session(mama_tap_session* session) {
	if (session == NULL) {
		return;
	}
	if (session->aggregate_device_id != kAudioObjectUnknown && session->io_proc_id != NULL) {
		AudioDeviceStop(session->aggregate_device_id, session->io_proc_id);
		AudioDeviceDestroyIOProcID(session->aggregate_device_id, session->io_proc_id);
		session->io_proc_id = NULL;
	}
	if (session->aggregate_device_id != kAudioObjectUnknown) {
		AudioHardwareDestroyAggregateDevice(session->aggregate_device_id);
		session->aggregate_device_id = kAudioObjectUnknown;
	}
	if (session->tap_id != kAudioObjectUnknown) {
		AudioHardwareDestroyProcessTap(session->tap_id);
		session->tap_id = kAudioObjectUnknown;
	}
	if (session->bundle_id != NULL) {
		free(session->bundle_id);
		session->bundle_id = NULL;
	}
	if (session->output_device_uid != NULL) {
		free(session->output_device_uid);
		session->output_device_uid = NULL;
	}
	free(session);
}

static int mama_tap_session_create(uint32_t processObjectID,
	const char* bundleID,
	const char* outputDeviceUID,
	float gain,
	uint32_t muted,
	mama_tap_session** outSession,
	char** outErr) {
	if (outSession == NULL) {
		return 1;
	}
	*outSession = NULL;
	if (outErr != NULL) {
		*outErr = NULL;
	}
	if (!mama_tap_api_available()) {
		mama_tap_set_plain_error(outErr, "process taps require macOS 14.2+");
		return 1;
	}
	if (processObjectID == 0 || outputDeviceUID == NULL || outputDeviceUID[0] == '\0') {
		mama_tap_set_plain_error(outErr, "tap session requires a process object and output device uid");
		return 1;
	}

	NSString* outputUIDString = [NSString stringWithUTF8String:outputDeviceUID];
	if (outputUIDString == nil) {
		mama_tap_set_plain_error(outErr, "invalid output device uid");
		return 1;
	}

	NSString* bundleIDString = nil;
	if (bundleID != NULL && bundleID[0] != '\0') {
		bundleIDString = [NSString stringWithUTF8String:bundleID];
	}

	CATapDescription* tapDescription = [[CATapDescription alloc] initWithProcesses:@[@(processObjectID)] andDeviceUID:outputUIDString withStream:0];
	tapDescription.privateTap = YES;
	tapDescription.muteBehavior = CATapMutedWhenTapped;
	tapDescription.name = @"MAMA Process Tap";
	tapDescription.UUID = [NSUUID UUID];
	uint32_t restoreCapable = 0;
	if (bundleIDString != nil) {
		if (@available(macOS 26.0, *)) {
			tapDescription.bundleIDs = @[bundleIDString];
			tapDescription.processRestoreEnabled = YES;
			restoreCapable = 1;
		}
	}

	AudioObjectID tapID = kAudioObjectUnknown;
	OSStatus st = AudioHardwareCreateProcessTap(tapDescription, &tapID);
	if (st != noErr) {
		mama_tap_set_error(outErr, "create process tap", st);
		return 1;
	}

	CFStringRef tapUIDRef = NULL;
	UInt32 size = (UInt32)sizeof(tapUIDRef);
	AudioObjectPropertyAddress tapUIDAddress = mama_tap_property_address(kAudioTapPropertyUID, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain);
	st = AudioObjectGetPropertyData(tapID, &tapUIDAddress, 0, NULL, &size, &tapUIDRef);
	if (st != noErr || tapUIDRef == NULL) {
		AudioHardwareDestroyProcessTap(tapID);
		mama_tap_set_error(outErr, "read process tap uid", st);
		return 1;
	}
	NSString* tapUIDString = CFBridgingRelease(tapUIDRef);

	NSDictionary* aggregateDescription = @{
		@kAudioAggregateDeviceUIDKey: [[NSUUID UUID] UUIDString],
		@kAudioAggregateDeviceNameKey: @"MAMA Tap Aggregate",
		@kAudioAggregateDeviceIsPrivateKey: @YES,
	};

	AudioObjectID aggregateDeviceID = kAudioObjectUnknown;
	st = AudioHardwareCreateAggregateDevice((__bridge CFDictionaryRef)aggregateDescription, &aggregateDeviceID);
	if (st != noErr) {
		AudioHardwareDestroyProcessTap(tapID);
		mama_tap_set_error(outErr, "create aggregate device", st);
		return 1;
	}

	if (!mama_tap_append_cfstring_property(aggregateDeviceID, kAudioAggregateDevicePropertyFullSubDeviceList, outputUIDString, outErr)) {
		AudioHardwareDestroyAggregateDevice(aggregateDeviceID);
		AudioHardwareDestroyProcessTap(tapID);
		return 1;
	}
	if (!mama_tap_append_cfstring_property(aggregateDeviceID, kAudioAggregateDevicePropertyTapList, tapUIDString, outErr)) {
		AudioHardwareDestroyAggregateDevice(aggregateDeviceID);
		AudioHardwareDestroyProcessTap(tapID);
		return 1;
	}
	if (!mama_tap_wait_for_streams(aggregateDeviceID, 5000)) {
		AudioHardwareDestroyAggregateDevice(aggregateDeviceID);
		AudioHardwareDestroyProcessTap(tapID);
		mama_tap_set_plain_error(outErr, "aggregate device streams did not become ready");
		return 1;
	}

	mama_tap_session* session = (mama_tap_session*)calloc(1, sizeof(mama_tap_session));
	if (session == NULL) {
		AudioHardwareDestroyAggregateDevice(aggregateDeviceID);
		AudioHardwareDestroyProcessTap(tapID);
		mama_tap_set_plain_error(outErr, "allocate tap session");
		return 1;
	}

	session->tap_id = tapID;
	session->aggregate_device_id = aggregateDeviceID;
	session->gain = gain;
	if (session->gain < 0.0f) {
		session->gain = 0.0f;
	}
	if (session->gain > 1.0f) {
		session->gain = 1.0f;
	}
	session->muted = muted ? 1 : 0;
	session->process_object_id = processObjectID;
	session->restore_capable = restoreCapable;
	if (bundleID != NULL && bundleID[0] != '\0') {
		session->bundle_id = strdup(bundleID);
	}
	session->output_device_uid = strdup(outputDeviceUID);

	st = AudioDeviceCreateIOProcID(aggregateDeviceID, mama_tap_ioproc, session, &session->io_proc_id);
	if (st != noErr) {
		mama_tap_cleanup_session(session);
		mama_tap_set_error(outErr, "create aggregate io proc", st);
		return 1;
	}

	st = AudioDeviceStart(aggregateDeviceID, session->io_proc_id);
	if (st != noErr) {
		mama_tap_cleanup_session(session);
		mama_tap_set_error(outErr, "start aggregate device", st);
		return 1;
	}

	*outSession = session;
	return 0;
}

static void mama_tap_session_destroy(mama_tap_session* session) {
	mama_tap_cleanup_session(session);
}

static int mama_tap_session_set_gain(mama_tap_session* session, float gain, char** outErr) {
	if (outErr != NULL) {
		*outErr = NULL;
	}
	if (session == NULL) {
		mama_tap_set_plain_error(outErr, "tap session unavailable");
		return 1;
	}
	if (gain < 0.0f) {
		gain = 0.0f;
	}
	if (gain > 1.0f) {
		gain = 1.0f;
	}
	session->gain = gain;
	return 0;
}

static int mama_tap_session_set_muted(mama_tap_session* session, uint32_t muted, char** outErr) {
	if (outErr != NULL) {
		*outErr = NULL;
	}
	if (session == NULL) {
		mama_tap_set_plain_error(outErr, "tap session unavailable");
		return 1;
	}
	session->muted = muted ? 1 : 0;
	return 0;
}

static void mama_tap_session_get_stats(mama_tap_session* session, uint64_t* outCallbacks, uint64_t* outFrames) {
	if (outCallbacks != NULL) {
		*outCallbacks = session != NULL ? session->callbacks : 0;
	}
	if (outFrames != NULL) {
		*outFrames = session != NULL ? session->frames : 0;
	}
}

static uint32_t mama_tap_session_restore_capable(mama_tap_session* session) {
	if (session == NULL) {
		return 0;
	}
	return session->restore_capable;
}

static void mama_tap_free_error(char* err) {
	if (err != NULL) {
		free(err);
	}
}
*/
import "C"

import (
	"fmt"
	"strings"
	"sync"
	"time"
	"unsafe"
)

type darwinTapTarget struct {
	Key           string
	ProcessObject uint32
	BundleID      string
	Name          string
}

type darwinTapState struct {
	Volume   int
	Muted    bool
	Attached bool
	Wanted   bool
}

type darwinTapStats struct {
	Callbacks      uint64
	Frames         uint64
	Attached       bool
	OutputDeviceID string
	RestoreCapable bool
}

type darwinTapBridge interface {
	Supported() bool
	EnsureTap(target darwinTapTarget) error
	ReleaseTap(sessionKey string) error
	SetGain(sessionKey string, volume int) error
	SetMuted(sessionKey string, muted bool) error
	ReadVirtualState(sessionKey string) darwinTapState
	Reconcile(targets []darwinTapTarget) error
	Stats(sessionKey string) (darwinTapStats, error)
}

type darwinTapRuntime interface {
	Supported() bool
	DefaultOutputDeviceUID() (string, error)
	CreateSession(target darwinTapTarget, outputDeviceUID string, volume int, muted bool) (darwinTapRuntimeSession, error)
}

type darwinTapRuntimeSession interface {
	SetGain(volume int) error
	SetMuted(muted bool) error
	Stats() (darwinTapStats, error)
	Close() error
	RestoreCapable() bool
}

type darwinTapHandle struct {
	session        darwinTapRuntimeSession
	processObject  uint32
	bundleID       string
	outputDeviceID string
	restoreCapable bool
}

type defaultDarwinTapBridge struct {
	runtime darwinTapRuntime

	mu      sync.Mutex
	handles map[string]*darwinTapHandle
	states  map[string]darwinTapState
}

type defaultDarwinTapRuntime struct{}

type cgoDarwinTapRuntimeSession struct {
	session        *C.mama_tap_session
	restoreCapable bool
}

func newDarwinTapBridge() darwinTapBridge {
	return newDefaultDarwinTapBridge(defaultDarwinTapRuntime{})
}

func newDefaultDarwinTapBridge(runtime darwinTapRuntime) *defaultDarwinTapBridge {
	if runtime == nil {
		runtime = defaultDarwinTapRuntime{}
	}
	return &defaultDarwinTapBridge{
		runtime: runtime,
		handles: map[string]*darwinTapHandle{},
		states:  map[string]darwinTapState{},
	}
}

func (defaultDarwinTapRuntime) Supported() bool {
	return C.mama_tap_api_available() != 0
}

func (defaultDarwinTapRuntime) DefaultOutputDeviceUID() (string, error) {
	var cUID *C.char
	var cErr *C.char
	result := C.mama_tap_default_output_device_uid(&cUID, &cErr)
	if cErr != nil {
		defer C.mama_tap_free_error(cErr)
	}
	if cUID != nil {
		defer C.free(unsafe.Pointer(cUID))
	}
	if result != 0 {
		if cErr != nil {
			return "", fmt.Errorf("darwin tap default output device: %s", C.GoString(cErr))
		}
		return "", fmt.Errorf("darwin tap default output device unavailable")
	}
	return strings.TrimSpace(C.GoString(cUID)), nil
}

func (defaultDarwinTapRuntime) CreateSession(target darwinTapTarget, outputDeviceUID string, volume int, muted bool) (darwinTapRuntimeSession, error) {
	if target.ProcessObject == 0 {
		return nil, fmt.Errorf("darwin tap session create: missing process object")
	}
	if strings.TrimSpace(outputDeviceUID) == "" {
		return nil, fmt.Errorf("darwin tap session create: missing output device uid")
	}

	cOutputUID := C.CString(outputDeviceUID)
	defer C.free(unsafe.Pointer(cOutputUID))

	var cBundleID *C.char
	if strings.TrimSpace(target.BundleID) != "" {
		cBundleID = C.CString(target.BundleID)
		defer C.free(unsafe.Pointer(cBundleID))
	}

	var cSession *C.mama_tap_session
	var cErr *C.char
	result := C.mama_tap_session_create(
		C.uint32_t(target.ProcessObject),
		cBundleID,
		cOutputUID,
		C.float(float32(clampPercent(volume))/100.0),
		boolToCUInt(muted),
		&cSession,
		&cErr,
	)
	if cErr != nil {
		defer C.mama_tap_free_error(cErr)
	}
	if result != 0 {
		if cErr != nil {
			return nil, fmt.Errorf("darwin tap session create: %s", C.GoString(cErr))
		}
		return nil, fmt.Errorf("darwin tap session create failed")
	}

	return &cgoDarwinTapRuntimeSession{
		session:        cSession,
		restoreCapable: C.mama_tap_session_restore_capable(cSession) != 0,
	}, nil
}

func (s *cgoDarwinTapRuntimeSession) SetGain(volume int) error {
	if s == nil || s.session == nil {
		return fmt.Errorf("darwin tap session unavailable")
	}
	var cErr *C.char
	result := C.mama_tap_session_set_gain(s.session, C.float(float32(clampPercent(volume))/100.0), &cErr)
	if cErr != nil {
		defer C.mama_tap_free_error(cErr)
	}
	if result != 0 {
		if cErr != nil {
			return fmt.Errorf("darwin tap set gain: %s", C.GoString(cErr))
		}
		return fmt.Errorf("darwin tap set gain failed")
	}
	return nil
}

func (s *cgoDarwinTapRuntimeSession) SetMuted(muted bool) error {
	if s == nil || s.session == nil {
		return fmt.Errorf("darwin tap session unavailable")
	}
	var cErr *C.char
	result := C.mama_tap_session_set_muted(s.session, boolToCUInt(muted), &cErr)
	if cErr != nil {
		defer C.mama_tap_free_error(cErr)
	}
	if result != 0 {
		if cErr != nil {
			return fmt.Errorf("darwin tap set mute: %s", C.GoString(cErr))
		}
		return fmt.Errorf("darwin tap set mute failed")
	}
	return nil
}

func (s *cgoDarwinTapRuntimeSession) Stats() (darwinTapStats, error) {
	if s == nil || s.session == nil {
		return darwinTapStats{}, fmt.Errorf("darwin tap session unavailable")
	}
	var callbacks C.uint64_t
	var frames C.uint64_t
	C.mama_tap_session_get_stats(s.session, &callbacks, &frames)
	return darwinTapStats{
		Callbacks:      uint64(callbacks),
		Frames:         uint64(frames),
		RestoreCapable: s.restoreCapable,
	}, nil
}

func (s *cgoDarwinTapRuntimeSession) Close() error {
	if s == nil || s.session == nil {
		return nil
	}
	C.mama_tap_session_destroy(s.session)
	s.session = nil
	return nil
}

func (s *cgoDarwinTapRuntimeSession) RestoreCapable() bool {
	if s == nil {
		return false
	}
	return s.restoreCapable
}

func (b *defaultDarwinTapBridge) Supported() bool {
	return b != nil && b.runtime != nil && b.runtime.Supported()
}

func (b *defaultDarwinTapBridge) EnsureTap(target darwinTapTarget) error {
	if !b.Supported() {
		return fmt.Errorf("darwin tap bridge unsupported")
	}
	if strings.TrimSpace(target.Key) == "" || target.ProcessObject == 0 {
		return fmt.Errorf("darwin tap target unavailable")
	}

	outputDeviceID, err := b.runtime.DefaultOutputDeviceUID()
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	state := b.stateLocked(target.Key)
	state.Wanted = true
	if b.handleMatchesLocked(target, outputDeviceID) {
		state.Attached = true
		b.saveStateLocked(target.Key, state)
		return nil
	}

	b.closeHandleLocked(target.Key)
	session, err := b.runtime.CreateSession(target, outputDeviceID, state.Volume, state.Muted)
	if err != nil {
		state.Attached = false
		b.saveStateLocked(target.Key, state)
		return err
	}

	b.handles[target.Key] = &darwinTapHandle{
		session:        session,
		processObject:  target.ProcessObject,
		bundleID:       strings.TrimSpace(target.BundleID),
		outputDeviceID: outputDeviceID,
		restoreCapable: session.RestoreCapable(),
	}
	state.Attached = true
	b.saveStateLocked(target.Key, state)
	return nil
}

func (b *defaultDarwinTapBridge) ReleaseTap(sessionKey string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closeHandleLocked(sessionKey)
	state := b.stateLocked(sessionKey)
	state.Attached = false
	state.Wanted = false
	b.saveStateLocked(sessionKey, state)
	return nil
}

func (b *defaultDarwinTapBridge) SetGain(sessionKey string, volume int) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	state := b.stateLocked(sessionKey)
	state.Volume = clampPercent(volume)
	state.Wanted = true

	handle := b.handles[sessionKey]
	if handle == nil || handle.session == nil {
		state.Attached = false
		b.saveStateLocked(sessionKey, state)
		return fmt.Errorf("darwin tap session %q not attached", sessionKey)
	}
	if err := handle.session.SetGain(state.Volume); err != nil {
		state.Attached = false
		b.saveStateLocked(sessionKey, state)
		return err
	}
	state.Attached = true
	b.saveStateLocked(sessionKey, state)
	return nil
}

func (b *defaultDarwinTapBridge) SetMuted(sessionKey string, muted bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	state := b.stateLocked(sessionKey)
	state.Muted = muted
	state.Wanted = true

	handle := b.handles[sessionKey]
	if handle == nil || handle.session == nil {
		state.Attached = false
		b.saveStateLocked(sessionKey, state)
		return fmt.Errorf("darwin tap session %q not attached", sessionKey)
	}
	if err := handle.session.SetMuted(muted); err != nil {
		state.Attached = false
		b.saveStateLocked(sessionKey, state)
		return err
	}
	state.Attached = true
	b.saveStateLocked(sessionKey, state)
	return nil
}

func (b *defaultDarwinTapBridge) ReadVirtualState(sessionKey string) darwinTapState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stateLocked(sessionKey)
}

func (b *defaultDarwinTapBridge) Reconcile(targets []darwinTapTarget) error {
	if !b.Supported() {
		return nil
	}

	outputDeviceID, err := b.runtime.DefaultOutputDeviceUID()
	if err != nil {
		return err
	}

	targetsByKey := make(map[string]darwinTapTarget, len(targets))
	for _, target := range targets {
		if strings.TrimSpace(target.Key) == "" {
			continue
		}
		targetsByKey[target.Key] = target
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for key, handle := range b.handles {
		target, ok := targetsByKey[key]
		state := b.stateLocked(key)
		if !ok {
			if handle.restoreCapable && handle.outputDeviceID == outputDeviceID {
				state.Attached = true
				b.saveStateLocked(key, state)
				continue
			}
			b.closeHandleLocked(key)
			state.Attached = false
			b.saveStateLocked(key, state)
			continue
		}

		if b.handleMatchesForTargetLocked(handle, target, outputDeviceID) {
			state.Attached = true
			b.saveStateLocked(key, state)
			continue
		}

		b.closeHandleLocked(key)
		state.Attached = false
		b.saveStateLocked(key, state)
	}

	var firstErr error
	for key, state := range b.states {
		if !state.Wanted {
			continue
		}
		if _, exists := b.handles[key]; exists {
			continue
		}

		target, ok := targetsByKey[key]
		if !ok {
			continue
		}

		session, err := b.runtime.CreateSession(target, outputDeviceID, state.Volume, state.Muted)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			state.Attached = false
			b.saveStateLocked(key, state)
			continue
		}

		b.handles[key] = &darwinTapHandle{
			session:        session,
			processObject:  target.ProcessObject,
			bundleID:       strings.TrimSpace(target.BundleID),
			outputDeviceID: outputDeviceID,
			restoreCapable: session.RestoreCapable(),
		}
		state.Attached = true
		b.saveStateLocked(key, state)
	}

	return firstErr
}

func (b *defaultDarwinTapBridge) Stats(sessionKey string) (darwinTapStats, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	state := b.stateLocked(sessionKey)
	handle := b.handles[sessionKey]
	if handle == nil || handle.session == nil {
		return darwinTapStats{Attached: state.Attached}, fmt.Errorf("darwin tap session %q not attached", sessionKey)
	}

	stats, err := handle.session.Stats()
	if err != nil {
		return darwinTapStats{}, err
	}
	stats.Attached = true
	stats.OutputDeviceID = handle.outputDeviceID
	stats.RestoreCapable = handle.restoreCapable
	return stats, nil
}

func (b *defaultDarwinTapBridge) stateLocked(sessionKey string) darwinTapState {
	if sessionKey == "" {
		return darwinTapState{Volume: 100}
	}
	if state, ok := b.states[sessionKey]; ok {
		state.Volume = clampPercent(state.Volume)
		return state
	}
	return darwinTapState{Volume: 100}
}

func (b *defaultDarwinTapBridge) saveStateLocked(sessionKey string, state darwinTapState) {
	if sessionKey == "" {
		return
	}
	state.Volume = clampPercent(state.Volume)
	b.states[sessionKey] = state
}

func (b *defaultDarwinTapBridge) handleMatchesLocked(target darwinTapTarget, outputDeviceID string) bool {
	handle := b.handles[target.Key]
	return b.handleMatchesForTargetLocked(handle, target, outputDeviceID)
}

func (b *defaultDarwinTapBridge) handleMatchesForTargetLocked(handle *darwinTapHandle, target darwinTapTarget, outputDeviceID string) bool {
	if handle == nil || handle.session == nil {
		return false
	}
	if handle.outputDeviceID != outputDeviceID {
		return false
	}
	if handle.restoreCapable {
		targetBundleID := strings.TrimSpace(target.BundleID)
		if targetBundleID == "" || strings.TrimSpace(handle.bundleID) == "" {
			return false
		}
		return strings.EqualFold(handle.bundleID, targetBundleID)
	}
	return handle.processObject == target.ProcessObject
}

func (b *defaultDarwinTapBridge) closeHandleLocked(sessionKey string) {
	handle := b.handles[sessionKey]
	if handle == nil {
		return
	}
	_ = handle.session.Close()
	delete(b.handles, sessionKey)
}

type DarwinTapProbeResult struct {
	BundleID        string
	ProcessObject   uint32
	SessionKey      string
	Volume          int
	Muted           bool
	Callbacks       uint64
	Frames          uint64
	OutputDeviceUID string
	RestoreCapable  bool
	Duration        time.Duration
}

func RunDarwinTapProbe(bundleID string, duration time.Duration, volume int, muted bool) (DarwinTapProbeResult, error) {
	if duration <= 0 {
		duration = 5 * time.Second
	}

	discovery := newCoreAudioBridge()
	processes, err := discovery.ListProcesses()
	if err != nil {
		return DarwinTapProbeResult{}, err
	}

	var targetSession darwinAppSession
	for _, process := range processes {
		session, ok := buildDarwinSession(process)
		if !ok {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(session.bundleID), strings.TrimSpace(bundleID)) {
			targetSession = session
			break
		}
	}
	if targetSession.stateKey == "" {
		return DarwinTapProbeResult{}, fmt.Errorf("%w: no running process found for bundle %q", ErrTargetUnavailable, bundleID)
	}

	tapBridge, ok := newDarwinTapBridge().(*defaultDarwinTapBridge)
	if !ok || !tapBridge.Supported() {
		return DarwinTapProbeResult{}, fmt.Errorf("darwin process taps require macOS 14.2+")
	}
	defer func() {
		_ = tapBridge.ReleaseTap(targetSession.stateKey)
	}()

	if err := tapBridge.EnsureTap(targetSession.tapTarget()); err != nil {
		return DarwinTapProbeResult{}, err
	}
	if err := tapBridge.SetGain(targetSession.stateKey, volume); err != nil {
		return DarwinTapProbeResult{}, err
	}
	if err := tapBridge.SetMuted(targetSession.stateKey, muted); err != nil {
		return DarwinTapProbeResult{}, err
	}

	time.Sleep(duration)

	state := tapBridge.ReadVirtualState(targetSession.stateKey)
	stats, err := tapBridge.Stats(targetSession.stateKey)
	if err != nil {
		return DarwinTapProbeResult{}, err
	}
	if stats.Frames == 0 {
		return DarwinTapProbeResult{}, fmt.Errorf("darwin tap probe observed no audio frames for %q", bundleID)
	}

	return DarwinTapProbeResult{
		BundleID:        targetSession.bundleID,
		ProcessObject:   targetSession.processObject,
		SessionKey:      targetSession.stateKey,
		Volume:          state.Volume,
		Muted:           state.Muted,
		Callbacks:       stats.Callbacks,
		Frames:          stats.Frames,
		OutputDeviceUID: stats.OutputDeviceID,
		RestoreCapable:  stats.RestoreCapable,
		Duration:        duration,
	}, nil
}

func clampPercent(volume int) int {
	if volume < 0 {
		return 0
	}
	if volume > 100 {
		return 100
	}
	return volume
}

func boolToCUInt(value bool) C.uint32_t {
	if value {
		return 1
	}
	return 0
}
