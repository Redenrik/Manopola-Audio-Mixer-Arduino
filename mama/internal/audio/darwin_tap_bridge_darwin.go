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
#include <stdatomic.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

typedef struct {
	AudioObjectID tap_id;
	AudioObjectID aggregate_device_id;
	AudioDeviceIOProcID io_proc_id;
	_Atomic(float) gain;
	_Atomic(uint32_t) muted;
	_Atomic(uint64_t) callbacks;
	_Atomic(uint64_t) frames;
	uint32_t process_object_id;
	uint32_t restore_capable;
	char* bundle_id;
	char* output_device_uid;
} mama_tap_session;

typedef struct {
	_Atomic(uint64_t) generation;
	uint32_t registered;
} mama_output_device_listener;

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

static void mama_tap_set_status_error(char** outErr, const char* msg, OSStatus st) {
	if (outErr == NULL) {
		return;
	}
	if (st == kAudioDevicePermissionsError) {
		char buffer[256];
		snprintf(buffer, sizeof(buffer), "permission denied: %s (osstatus=%d)", msg, (int)st);
		*outErr = strdup(buffer);
		return;
	}
	if (st == kAudioHardwareUnsupportedOperationError) {
		char buffer[256];
		snprintf(buffer, sizeof(buffer), "unsupported: %s (osstatus=%d)", msg, (int)st);
		*outErr = strdup(buffer);
		return;
	}
	if (st == kAudioHardwareNotReadyError) {
		char buffer[256];
		snprintf(buffer, sizeof(buffer), "transient tap failure: %s (osstatus=%d)", msg, (int)st);
		*outErr = strdup(buffer);
		return;
	}
	mama_tap_set_error(outErr, msg, st);
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
	atomic_fetch_add_explicit(&session->callbacks, 1, memory_order_relaxed);

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
	uint32_t muted = atomic_load_explicit(&session->muted, memory_order_relaxed);
	float gain = muted ? 0.0f : atomic_load_explicit(&session->gain, memory_order_relaxed);
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
		atomic_fetch_add_explicit(&session->frames, (uint64_t)((bytes / sizeof(Float32)) / channels), memory_order_relaxed);
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
		mama_tap_set_status_error(outErr, "output device unavailable: read default output device", st);
		return 1;
	}
	if (deviceID == kAudioObjectUnknown) {
		mama_tap_set_plain_error(outErr, "output device unavailable: default output device unavailable");
		return 1;
	}

	CFStringRef uid = NULL;
	size = (UInt32)sizeof(uid);
	address = mama_tap_property_address(kAudioDevicePropertyDeviceUID, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain);
	st = AudioObjectGetPropertyData(deviceID, &address, 0, NULL, &size, &uid);
	if (st != noErr || uid == NULL) {
		mama_tap_set_status_error(outErr, "output device unavailable: read default output device uid", st);
		return 1;
	}

	char* copied = mama_tap_copy_cfstring(uid);
	CFRelease(uid);
	if (copied == NULL) {
		mama_tap_set_plain_error(outErr, "output device unavailable: copy default output device uid");
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
		mama_tap_set_status_error(outErr, "transient tap failure: read aggregate property size", st);
		return 0;
	}

	NSMutableArray* items = [NSMutableArray array];
	if (size > 0) {
		CFArrayRef listRef = NULL;
		UInt32 listSize = size;
		st = AudioObjectGetPropertyData(objectID, &address, 0, NULL, &listSize, &listRef);
		if (st != noErr) {
			mama_tap_set_status_error(outErr, "transient tap failure: read aggregate property data", st);
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
		mama_tap_set_status_error(outErr, "transient tap failure: write aggregate property data", st);
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
		mama_tap_set_plain_error(outErr, "unsupported: process taps require macOS 14.2+");
		return 1;
	}
	if (processObjectID == 0 || outputDeviceUID == NULL || outputDeviceUID[0] == '\0') {
		mama_tap_set_plain_error(outErr, "target unavailable: tap session requires a process object and output device uid");
		return 1;
	}

	NSString* outputUIDString = [NSString stringWithUTF8String:outputDeviceUID];
	if (outputUIDString == nil) {
		mama_tap_set_plain_error(outErr, "output device unavailable: invalid output device uid");
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
		mama_tap_set_status_error(outErr, "create process tap", st);
		return 1;
	}

	CFStringRef tapUIDRef = NULL;
	UInt32 size = (UInt32)sizeof(tapUIDRef);
	AudioObjectPropertyAddress tapUIDAddress = mama_tap_property_address(kAudioTapPropertyUID, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain);
	st = AudioObjectGetPropertyData(tapID, &tapUIDAddress, 0, NULL, &size, &tapUIDRef);
	if (st != noErr || tapUIDRef == NULL) {
		AudioHardwareDestroyProcessTap(tapID);
		mama_tap_set_status_error(outErr, "transient tap failure: read process tap uid", st);
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
		mama_tap_set_status_error(outErr, "transient tap failure: create aggregate device", st);
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
		mama_tap_set_plain_error(outErr, "transient tap failure: aggregate device streams did not become ready");
		return 1;
	}

	mama_tap_session* session = (mama_tap_session*)calloc(1, sizeof(mama_tap_session));
	if (session == NULL) {
		AudioHardwareDestroyAggregateDevice(aggregateDeviceID);
		AudioHardwareDestroyProcessTap(tapID);
		mama_tap_set_plain_error(outErr, "transient tap failure: allocate tap session");
		return 1;
	}

	session->tap_id = tapID;
	session->aggregate_device_id = aggregateDeviceID;
	float clampedGain = gain;
	if (clampedGain < 0.0f) {
		clampedGain = 0.0f;
	}
	if (clampedGain > 1.0f) {
		clampedGain = 1.0f;
	}
	atomic_store_explicit(&session->gain, clampedGain, memory_order_relaxed);
	atomic_store_explicit(&session->muted, muted ? 1 : 0, memory_order_relaxed);
	session->process_object_id = processObjectID;
	session->restore_capable = restoreCapable;
	if (bundleID != NULL && bundleID[0] != '\0') {
		session->bundle_id = strdup(bundleID);
	}
	session->output_device_uid = strdup(outputDeviceUID);

	st = AudioDeviceCreateIOProcID(aggregateDeviceID, mama_tap_ioproc, session, &session->io_proc_id);
	if (st != noErr) {
		mama_tap_cleanup_session(session);
		mama_tap_set_status_error(outErr, "transient tap failure: create aggregate io proc", st);
		return 1;
	}

	st = AudioDeviceStart(aggregateDeviceID, session->io_proc_id);
	if (st != noErr) {
		mama_tap_cleanup_session(session);
		mama_tap_set_status_error(outErr, "transient tap failure: start aggregate device", st);
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
		mama_tap_set_plain_error(outErr, "target unavailable: tap session unavailable");
		return 1;
	}
	if (gain < 0.0f) {
		gain = 0.0f;
	}
	if (gain > 1.0f) {
		gain = 1.0f;
	}
	atomic_store_explicit(&session->gain, gain, memory_order_relaxed);
	return 0;
}

static int mama_tap_session_set_muted(mama_tap_session* session, uint32_t muted, char** outErr) {
	if (outErr != NULL) {
		*outErr = NULL;
	}
	if (session == NULL) {
		mama_tap_set_plain_error(outErr, "target unavailable: tap session unavailable");
		return 1;
	}
	atomic_store_explicit(&session->muted, muted ? 1 : 0, memory_order_relaxed);
	return 0;
}

static void mama_tap_session_get_stats(mama_tap_session* session, uint64_t* outCallbacks, uint64_t* outFrames) {
	if (outCallbacks != NULL) {
		*outCallbacks = session != NULL ? atomic_load_explicit(&session->callbacks, memory_order_relaxed) : 0;
	}
	if (outFrames != NULL) {
		*outFrames = session != NULL ? atomic_load_explicit(&session->frames, memory_order_relaxed) : 0;
	}
}

static uint32_t mama_tap_session_restore_capable(mama_tap_session* session) {
	if (session == NULL) {
		return 0;
	}
	return session->restore_capable;
}

static OSStatus mama_output_device_listener_proc(AudioObjectID inObjectID,
	UInt32 inNumberAddresses,
	const AudioObjectPropertyAddress inAddresses[],
	void* inClientData) {
	mama_output_device_listener* listener = (mama_output_device_listener*)inClientData;
	if (listener == NULL) {
		return noErr;
	}
	atomic_fetch_add_explicit(&listener->generation, 1, memory_order_relaxed);
	return noErr;
}

static int mama_tap_output_device_listener_create(mama_output_device_listener** outListener, char** outErr) {
	if (outListener == NULL) {
		return 1;
	}
	*outListener = NULL;
	if (outErr != NULL) {
		*outErr = NULL;
	}

	mama_output_device_listener* listener = (mama_output_device_listener*)calloc(1, sizeof(mama_output_device_listener));
	if (listener == NULL) {
		mama_tap_set_plain_error(outErr, "transient tap failure: allocate output device listener");
		return 1;
	}

	AudioObjectPropertyAddress address = mama_tap_property_address(
		kAudioHardwarePropertyDefaultOutputDevice,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain);
	OSStatus st = AudioObjectAddPropertyListener(kAudioObjectSystemObject, &address, mama_output_device_listener_proc, listener);
	if (st != noErr) {
		free(listener);
		mama_tap_set_status_error(outErr, "transient tap failure: register default output device listener", st);
		return 1;
	}

	listener->registered = 1;
	*outListener = listener;
	return 0;
}

static uint64_t mama_tap_output_device_listener_generation(mama_output_device_listener* listener) {
	if (listener == NULL) {
		return 0;
	}
	return atomic_load_explicit(&listener->generation, memory_order_relaxed);
}

static void mama_tap_output_device_listener_destroy(mama_output_device_listener* listener) {
	if (listener == NULL) {
		return;
	}
	if (listener->registered) {
		AudioObjectPropertyAddress address = mama_tap_property_address(
			kAudioHardwarePropertyDefaultOutputDevice,
			kAudioObjectPropertyScopeGlobal,
			kAudioObjectPropertyElementMain);
		AudioObjectRemovePropertyListener(kAudioObjectSystemObject, &address, mama_output_device_listener_proc, listener);
	}
	free(listener);
}

static void mama_tap_free_error(char* err) {
	if (err != NULL) {
		free(err);
	}
}
*/
import "C"

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unsafe"

	runtimelog "mama/internal/runtime"
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
	StartDefaultOutputListener() (darwinTapOutputListener, error)
}

type darwinTapRuntimeSession interface {
	SetGain(volume int) error
	SetMuted(muted bool) error
	Stats() (darwinTapStats, error)
	Close() error
	RestoreCapable() bool
}

type darwinTapOutputListener interface {
	Generation() uint64
	Close() error
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

	liveTargets          map[string]darwinTapTarget
	outputListener       darwinTapOutputListener
	outputListenerStart  sync.Once
	outputChangeDebounce time.Duration
	outputPollInterval   time.Duration
}

type defaultDarwinTapRuntime struct{}

type cgoDarwinTapRuntimeSession struct {
	session        *C.mama_tap_session
	restoreCapable bool
}

type cgoDarwinTapOutputListener struct {
	listener *C.mama_output_device_listener
}

func newDarwinTapBridge() darwinTapBridge {
	return newDefaultDarwinTapBridge(defaultDarwinTapRuntime{})
}

func newDefaultDarwinTapBridge(runtime darwinTapRuntime) *defaultDarwinTapBridge {
	if runtime == nil {
		runtime = defaultDarwinTapRuntime{}
	}
	return &defaultDarwinTapBridge{
		runtime:              runtime,
		handles:              map[string]*darwinTapHandle{},
		states:               map[string]darwinTapState{},
		liveTargets:          map[string]darwinTapTarget{},
		outputChangeDebounce: 250 * time.Millisecond,
		outputPollInterval:   100 * time.Millisecond,
	}
}

func (defaultDarwinTapRuntime) Supported() bool {
	return C.mama_tap_api_available() != 0
}

func (defaultDarwinTapRuntime) StartDefaultOutputListener() (darwinTapOutputListener, error) {
	var cListener *C.mama_output_device_listener
	var cErr *C.char
	result := C.mama_tap_output_device_listener_create(&cListener, &cErr)
	if cErr != nil {
		defer C.mama_tap_free_error(cErr)
	}
	if result != 0 {
		if cErr != nil {
			return nil, wrapDarwinTapError("darwin tap default output listener", errors.New(C.GoString(cErr)))
		}
		return nil, wrapDarwinTapError("darwin tap default output listener", errors.New("listener unavailable"))
	}
	return &cgoDarwinTapOutputListener{listener: cListener}, nil
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
			return "", wrapDarwinTapError("darwin tap default output device", errors.New(C.GoString(cErr)))
		}
		return "", wrapDarwinTapError("darwin tap default output device", errors.New("output device unavailable"))
	}
	return strings.TrimSpace(C.GoString(cUID)), nil
}

func (defaultDarwinTapRuntime) CreateSession(target darwinTapTarget, outputDeviceUID string, volume int, muted bool) (darwinTapRuntimeSession, error) {
	if target.ProcessObject == 0 {
		return nil, wrapDarwinTapError("darwin tap session create", errors.New("target unavailable: missing process object"))
	}
	if strings.TrimSpace(outputDeviceUID) == "" {
		return nil, wrapDarwinTapError("darwin tap session create", errors.New("output device unavailable: missing output device uid"))
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
			return nil, wrapDarwinTapError("darwin tap session create", errors.New(C.GoString(cErr)))
		}
		return nil, wrapDarwinTapError("darwin tap session create", errors.New("transient tap failure: create failed"))
	}

	return &cgoDarwinTapRuntimeSession{
		session:        cSession,
		restoreCapable: C.mama_tap_session_restore_capable(cSession) != 0,
	}, nil
}

func (s *cgoDarwinTapRuntimeSession) SetGain(volume int) error {
	if s == nil || s.session == nil {
		return wrapDarwinTapError("darwin tap set gain", errors.New("target unavailable: tap session unavailable"))
	}
	var cErr *C.char
	result := C.mama_tap_session_set_gain(s.session, C.float(float32(clampPercent(volume))/100.0), &cErr)
	if cErr != nil {
		defer C.mama_tap_free_error(cErr)
	}
	if result != 0 {
		if cErr != nil {
			return wrapDarwinTapError("darwin tap set gain", errors.New(C.GoString(cErr)))
		}
		return wrapDarwinTapError("darwin tap set gain", errors.New("transient tap failure: set gain failed"))
	}
	return nil
}

func (s *cgoDarwinTapRuntimeSession) SetMuted(muted bool) error {
	if s == nil || s.session == nil {
		return wrapDarwinTapError("darwin tap set mute", errors.New("target unavailable: tap session unavailable"))
	}
	var cErr *C.char
	result := C.mama_tap_session_set_muted(s.session, boolToCUInt(muted), &cErr)
	if cErr != nil {
		defer C.mama_tap_free_error(cErr)
	}
	if result != 0 {
		if cErr != nil {
			return wrapDarwinTapError("darwin tap set mute", errors.New(C.GoString(cErr)))
		}
		return wrapDarwinTapError("darwin tap set mute", errors.New("transient tap failure: set mute failed"))
	}
	return nil
}

func (s *cgoDarwinTapRuntimeSession) Stats() (darwinTapStats, error) {
	if s == nil || s.session == nil {
		return darwinTapStats{}, wrapDarwinTapError("darwin tap stats", errors.New("target unavailable: tap session unavailable"))
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

func (l *cgoDarwinTapOutputListener) Generation() uint64 {
	if l == nil || l.listener == nil {
		return 0
	}
	return uint64(C.mama_tap_output_device_listener_generation(l.listener))
}

func (l *cgoDarwinTapOutputListener) Close() error {
	if l == nil || l.listener == nil {
		return nil
	}
	C.mama_tap_output_device_listener_destroy(l.listener)
	l.listener = nil
	return nil
}

func darwinTapErrorSentinel(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrPermissionDenied):
		return ErrPermissionDenied
	case errors.Is(err, ErrUnsupported):
		return ErrUnsupported
	case errors.Is(err, ErrTargetUnavailable):
		return ErrTargetUnavailable
	case errors.Is(err, ErrOutputDeviceUnavailable):
		return ErrOutputDeviceUnavailable
	case errors.Is(err, ErrTransientTapFailure):
		return ErrTransientTapFailure
	}

	message := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(message, "permission denied"),
		strings.Contains(message, "access denied"),
		strings.Contains(message, "not permitted"),
		strings.Contains(message, "coinitialize"):
		return ErrPermissionDenied
	case strings.Contains(message, "unsupported"),
		strings.Contains(message, "macos 14.2+"),
		strings.Contains(message, "not supported"):
		return ErrUnsupported
	case strings.Contains(message, "output device unavailable"),
		strings.Contains(message, "default output device"):
		return ErrOutputDeviceUnavailable
	case strings.Contains(message, "target unavailable"),
		strings.Contains(message, "missing process object"),
		strings.Contains(message, "not attached"),
		strings.Contains(message, "tap session unavailable"):
		return ErrTargetUnavailable
	default:
		return ErrTransientTapFailure
	}
}

func wrapDarwinTapError(prefix string, err error) error {
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		message = "darwin tap failure"
	}
	wrapped := fmt.Errorf("%w: %s", darwinTapErrorSentinel(err), message)
	if strings.TrimSpace(prefix) == "" {
		return wrapped
	}
	return fmt.Errorf("%s: %w", prefix, wrapped)
}

func (b *defaultDarwinTapBridge) Supported() bool {
	return b != nil && b.runtime != nil && b.runtime.Supported()
}

func (b *defaultDarwinTapBridge) EnsureTap(target darwinTapTarget) error {
	if !b.Supported() {
		return wrapDarwinTapError("darwin tap ensure", errors.New("unsupported: darwin tap bridge unavailable"))
	}
	if strings.TrimSpace(target.Key) == "" || target.ProcessObject == 0 {
		return wrapDarwinTapError("darwin tap ensure", errors.New("target unavailable: missing tap target identity"))
	}
	b.ensureOutputWatcher()

	outputDeviceID, err := b.runtime.DefaultOutputDeviceUID()
	if err != nil {
		err = wrapDarwinTapError("darwin tap default output device", err)
		b.log("darwin_tap_attach_failed", runtimelog.Fields{
			"error":          err,
			"session_key":    target.Key,
			"bundle_id":      target.BundleID,
			"process_object": target.ProcessObject,
			"reason":         "default_output_lookup",
		})
		return err
	}

	b.mu.Lock()
	state := b.stateLocked(target.Key)
	state.Wanted = true
	b.liveTargets[target.Key] = target
	if b.handleMatchesLocked(target, outputDeviceID) {
		state.Attached = true
		b.saveStateLocked(target.Key, state)
		b.mu.Unlock()
		return nil
	}

	b.closeHandleLocked(target.Key, "replace_attach")
	session, err := b.runtime.CreateSession(target, outputDeviceID, state.Volume, state.Muted)
	if err != nil {
		err = wrapDarwinTapError("darwin tap session create", err)
		state.Attached = false
		b.saveStateLocked(target.Key, state)
		b.mu.Unlock()
		b.log("darwin_tap_attach_failed", runtimelog.Fields{
			"error":          err,
			"session_key":    target.Key,
			"bundle_id":      target.BundleID,
			"output_device":  outputDeviceID,
			"process_object": target.ProcessObject,
		})
		return err
	}

	handle := &darwinTapHandle{
		session:        session,
		processObject:  target.ProcessObject,
		bundleID:       strings.TrimSpace(target.BundleID),
		outputDeviceID: outputDeviceID,
		restoreCapable: session.RestoreCapable(),
	}
	b.handles[target.Key] = handle
	state.Attached = true
	b.saveStateLocked(target.Key, state)
	b.mu.Unlock()
	b.log("darwin_tap_created", runtimelog.Fields{
		"session_key":    target.Key,
		"bundle_id":      target.BundleID,
		"output_device":  outputDeviceID,
		"process_object": target.ProcessObject,
		"restore":        handle.restoreCapable,
	})
	return nil
}

func (b *defaultDarwinTapBridge) ReleaseTap(sessionKey string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.liveTargets, sessionKey)
	b.closeHandleLocked(sessionKey, "release")
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
		return wrapDarwinTapError("darwin tap set gain", errors.New("target unavailable: tap session not attached"))
	}
	if err := handle.session.SetGain(state.Volume); err != nil {
		err = wrapDarwinTapError("darwin tap set gain", err)
		state.Attached = false
		b.saveStateLocked(sessionKey, state)
		b.log("darwin_tap_attach_failed", runtimelog.Fields{
			"error":       err,
			"session_key": sessionKey,
			"reason":      "set_gain",
		})
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
		return wrapDarwinTapError("darwin tap set mute", errors.New("target unavailable: tap session not attached"))
	}
	if err := handle.session.SetMuted(muted); err != nil {
		err = wrapDarwinTapError("darwin tap set mute", err)
		state.Attached = false
		b.saveStateLocked(sessionKey, state)
		b.log("darwin_tap_attach_failed", runtimelog.Fields{
			"error":       err,
			"session_key": sessionKey,
			"reason":      "set_mute",
		})
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
	b.ensureOutputWatcher()

	outputDeviceID, err := b.runtime.DefaultOutputDeviceUID()
	if err != nil {
		return wrapDarwinTapError("darwin tap default output device", err)
	}

	targetsByKey := make(map[string]darwinTapTarget, len(targets))
	for _, target := range targets {
		if strings.TrimSpace(target.Key) == "" {
			continue
		}
		targetsByKey[target.Key] = target
	}

	b.mu.Lock()
	b.liveTargets = cloneDarwinTapTargetsMap(targetsByKey)

	for key, handle := range b.handles {
		target, ok := targetsByKey[key]
		state := b.stateLocked(key)
		if !ok {
			if handle.restoreCapable && handle.outputDeviceID == outputDeviceID {
				state.Attached = true
				b.saveStateLocked(key, state)
				continue
			}
			b.closeHandleLocked(key, "target_gone")
			state.Attached = false
			b.saveStateLocked(key, state)
			continue
		}

		if b.handleMatchesForTargetLocked(handle, target, outputDeviceID) {
			state.Attached = true
			b.saveStateLocked(key, state)
			continue
		}

		b.closeHandleLocked(key, "reconcile_mismatch")
		state.Attached = false
		b.saveStateLocked(key, state)
	}

	var firstErr error
	type createRecord struct {
		key    string
		target darwinTapTarget
		handle *darwinTapHandle
	}
	created := make([]createRecord, 0)
	recovered := make([]createRecord, 0)
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
			err = wrapDarwinTapError("darwin tap session create", err)
			if firstErr == nil {
				firstErr = err
			}
			state.Attached = false
			b.saveStateLocked(key, state)
			continue
		}

		handle := &darwinTapHandle{
			session:        session,
			processObject:  target.ProcessObject,
			bundleID:       strings.TrimSpace(target.BundleID),
			outputDeviceID: outputDeviceID,
			restoreCapable: session.RestoreCapable(),
		}
		b.handles[key] = handle
		state.Attached = true
		b.saveStateLocked(key, state)
		record := createRecord{key: key, target: target, handle: handle}
		created = append(created, record)
		if state.Wanted {
			recovered = append(recovered, record)
		}
	}
	b.mu.Unlock()
	for _, record := range created {
		b.log("darwin_tap_created", runtimelog.Fields{
			"session_key":    record.key,
			"bundle_id":      record.target.BundleID,
			"output_device":  record.handle.outputDeviceID,
			"process_object": record.target.ProcessObject,
			"restore":        record.handle.restoreCapable,
		})
	}
	for _, record := range recovered {
		b.log("darwin_tap_restart_recovered", runtimelog.Fields{
			"session_key":    record.key,
			"bundle_id":      record.target.BundleID,
			"output_device":  record.handle.outputDeviceID,
			"process_object": record.target.ProcessObject,
			"restore":        record.handle.restoreCapable,
		})
	}
	if firstErr != nil {
		b.log("darwin_tap_attach_failed", runtimelog.Fields{
			"error":  firstErr,
			"reason": "reconcile_create",
		})
	}
	return firstErr
}

func (b *defaultDarwinTapBridge) Stats(sessionKey string) (darwinTapStats, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	state := b.stateLocked(sessionKey)
	handle := b.handles[sessionKey]
	if handle == nil || handle.session == nil {
		return darwinTapStats{Attached: state.Attached}, wrapDarwinTapError("darwin tap stats", errors.New("target unavailable: tap session not attached"))
	}

	stats, err := handle.session.Stats()
	if err != nil {
		return darwinTapStats{}, wrapDarwinTapError("darwin tap stats", err)
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

func (b *defaultDarwinTapBridge) ensureOutputWatcher() {
	if !b.Supported() {
		return
	}
	b.outputListenerStart.Do(func() {
		listener, err := b.runtime.StartDefaultOutputListener()
		if err != nil {
			b.log("darwin_tap_output_listener_failed", runtimelog.Fields{"error": err})
			return
		}
		if listener == nil {
			return
		}
		b.mu.Lock()
		b.outputListener = listener
		b.mu.Unlock()
		go b.watchOutputDeviceChanges(listener)
	})
}

func (b *defaultDarwinTapBridge) watchOutputDeviceChanges(listener darwinTapOutputListener) {
	ticker := time.NewTicker(b.outputPollInterval)
	defer ticker.Stop()

	lastGeneration := listener.Generation()
	var changedAt time.Time
	for range ticker.C {
		current := listener.Generation()
		if current != lastGeneration {
			lastGeneration = current
			changedAt = time.Now()
			continue
		}
		if changedAt.IsZero() || time.Since(changedAt) < b.outputChangeDebounce {
			continue
		}
		if err := b.rebuildForOutputDeviceChange(); err != nil {
			b.log("darwin_tap_output_rebuild_failed", runtimelog.Fields{"error": err})
		}
		changedAt = time.Time{}
	}
}

func (b *defaultDarwinTapBridge) rebuildForOutputDeviceChange() error {
	targets := b.snapshotLiveTargets()
	b.log("darwin_tap_output_device_rebuild", runtimelog.Fields{
		"target_count": len(targets),
	})
	return b.Reconcile(targets)
}

func (b *defaultDarwinTapBridge) snapshotLiveTargets() []darwinTapTarget {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.liveTargets) == 0 {
		return nil
	}
	targets := make([]darwinTapTarget, 0, len(b.liveTargets))
	for _, target := range b.liveTargets {
		targets = append(targets, target)
	}
	return targets
}

func (b *defaultDarwinTapBridge) log(event string, fields runtimelog.Fields) {
	if fields == nil {
		fields = runtimelog.Fields{}
	}
	runtimelog.Log(event, fields)
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

func (b *defaultDarwinTapBridge) closeHandleLocked(sessionKey string, reason string) {
	handle := b.handles[sessionKey]
	if handle == nil {
		return
	}
	_ = handle.session.Close()
	b.log("darwin_tap_destroyed", runtimelog.Fields{
		"session_key":   sessionKey,
		"bundle_id":     handle.bundleID,
		"output_device": handle.outputDeviceID,
		"reason":        reason,
		"restore":       handle.restoreCapable,
	})
	delete(b.handles, sessionKey)
}

func cloneDarwinTapTargetsMap(in map[string]darwinTapTarget) map[string]darwinTapTarget {
	if len(in) == 0 {
		return map[string]darwinTapTarget{}
	}
	out := make(map[string]darwinTapTarget, len(in))
	for key, target := range in {
		out[key] = target
	}
	return out
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
		return DarwinTapProbeResult{}, wrapDarwinTapError("darwin tap probe", errors.New("unsupported: darwin process taps require macOS 14.2+"))
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
