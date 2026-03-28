//go:build darwin && cgo

package audio

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework AudioToolbox -framework AudioUnit -framework CoreAudio -framework CoreFoundation -framework Foundation

#include <AudioUnit/AudioUnit.h>
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
	AudioDeviceIOProcID capture_proc_id;
	AudioUnit output_unit;
	AudioStreamBasicDescription tap_format;
	Float32* ring_samples;
	uint32_t ring_capacity_frames;
	_Atomic(uint64_t) ring_read_frames;
	_Atomic(uint64_t) ring_write_frames;
	_Atomic(float) gain;
	_Atomic(uint32_t) muted;
	_Atomic(uint64_t) capture_callbacks;
	_Atomic(uint64_t) capture_frames;
	_Atomic(uint64_t) render_callbacks;
	_Atomic(uint64_t) render_frames;
	_Atomic(uint64_t) underruns;
	_Atomic(uint64_t) overflows;
	uint32_t process_object_id;
	uint32_t restore_capable;
	char* bundle_id;
	char* output_device_uid;
	uint32_t output_stream_index;
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

static uint32_t mama_tap_channels(const mama_tap_session* session) {
	if (session == NULL || session->tap_format.mChannelsPerFrame == 0) {
		return 0;
	}
	return session->tap_format.mChannelsPerFrame;
}

static uint64_t mama_tap_min_u64(uint64_t a, uint64_t b) {
	return a < b ? a : b;
}

static void mama_tap_zero_output(AudioBufferList* ioData) {
	if (ioData == NULL) {
		return;
	}
	for (UInt32 i = 0; i < ioData->mNumberBuffers; i++) {
		AudioBuffer* outBuf = &ioData->mBuffers[i];
		if (outBuf->mData != NULL && outBuf->mDataByteSize > 0) {
			memset(outBuf->mData, 0, outBuf->mDataByteSize);
		}
	}
}

static uint64_t mama_tap_frames_for_buffer(const AudioBuffer* buffer, uint32_t bytesPerFrame) {
	if (buffer == NULL || buffer->mData == NULL || buffer->mDataByteSize == 0 || bytesPerFrame == 0) {
		return 0;
	}
	return buffer->mDataByteSize / bytesPerFrame;
}

static uint64_t mama_tap_output_frames(AudioBufferList* ioData, uint32_t bytesPerFrame) {
	if (ioData == NULL || ioData->mNumberBuffers == 0 || bytesPerFrame == 0) {
		return 0;
	}
	uint64_t frames = 0;
	for (UInt32 i = 0; i < ioData->mNumberBuffers; i++) {
		uint64_t candidate = mama_tap_frames_for_buffer(&ioData->mBuffers[i], bytesPerFrame);
		if (candidate == 0) {
			continue;
		}
		if (frames == 0 || candidate < frames) {
			frames = candidate;
		}
	}
	return frames;
}

static void mama_tap_copy_interleaved_output(mama_tap_session* session,
	const Float32* src,
	uint64_t frames,
	AudioBufferList* ioData) {
	if (session == NULL || src == NULL || ioData == NULL || frames == 0) {
		mama_tap_zero_output(ioData);
		return;
	}

	uint32_t channels = mama_tap_channels(session);
	if (channels == 0) {
		mama_tap_zero_output(ioData);
		return;
	}

	uint32_t muted = atomic_load_explicit(&session->muted, memory_order_relaxed);
	float gain = muted ? 0.0f : atomic_load_explicit(&session->gain, memory_order_relaxed);

	if (ioData->mNumberBuffers == 1) {
		AudioBuffer* outBuf = &ioData->mBuffers[0];
		if (outBuf->mData == NULL || outBuf->mDataByteSize == 0) {
			mama_tap_zero_output(ioData);
			return;
		}
		uint64_t totalSamples = frames * channels;
		Float32* outSamples = (Float32*)outBuf->mData;
		if (gain <= 0.0001f) {
			memset(outSamples, 0, totalSamples * sizeof(Float32));
		} else if (gain >= 0.9999f) {
			memcpy(outSamples, src, totalSamples * sizeof(Float32));
		} else {
			for (uint64_t i = 0; i < totalSamples; i++) {
				outSamples[i] = src[i] * gain;
			}
		}
		return;
	}

	for (UInt32 i = 0; i < ioData->mNumberBuffers; i++) {
		AudioBuffer* outBuf = &ioData->mBuffers[i];
		if (outBuf->mData == NULL || outBuf->mDataByteSize == 0) {
			continue;
		}
		if (i >= channels || outBuf->mNumberChannels != 1) {
			memset(outBuf->mData, 0, outBuf->mDataByteSize);
			continue;
		}
		Float32* outSamples = (Float32*)outBuf->mData;
		uint64_t frameCount = mama_tap_frames_for_buffer(outBuf, (uint32_t)sizeof(Float32));
		if (frameCount > frames) {
			frameCount = frames;
		}
		if (gain <= 0.0001f) {
			memset(outSamples, 0, frameCount * sizeof(Float32));
			continue;
		}
		for (uint64_t frame = 0; frame < frameCount; frame++) {
			Float32 sample = src[(frame * channels) + i];
			outSamples[frame] = gain >= 0.9999f ? sample : sample * gain;
		}
	}
}

static OSStatus mama_tap_capture_ioproc(AudioObjectID inDevice,
	const AudioTimeStamp* inNow,
	const AudioBufferList* inInputData,
	const AudioTimeStamp* inInputTime,
	AudioBufferList* outOutputData,
	const AudioTimeStamp* inOutputTime,
	void* inClientData) {
	(void)inDevice;
	(void)inNow;
	(void)inInputTime;
	(void)inOutputTime;
	mama_tap_session* session = (mama_tap_session*)inClientData;
	if (session == NULL) {
		return noErr;
	}
	atomic_fetch_add_explicit(&session->capture_callbacks, 1, memory_order_relaxed);
	if (session->tap_format.mBytesPerFrame == 0) {
		mama_tap_zero_output(outOutputData);
		return noErr;
	}

	uint64_t outputFrames = mama_tap_output_frames(outOutputData, session->tap_format.mBytesPerFrame);
	if (inInputData == NULL || inInputData->mNumberBuffers == 0) {
		mama_tap_zero_output(outOutputData);
		if (outputFrames > 0) {
			atomic_fetch_add_explicit(&session->underruns, outputFrames, memory_order_relaxed);
		}
		return noErr;
	}
	const AudioBuffer* inBuf = &inInputData->mBuffers[0];
	uint64_t inputFrames = mama_tap_frames_for_buffer(inBuf, session->tap_format.mBytesPerFrame);
	atomic_fetch_add_explicit(&session->capture_frames, inputFrames, memory_order_relaxed);
	atomic_fetch_add_explicit(&session->render_callbacks, 1, memory_order_relaxed);

	uint64_t frames = inputFrames;
	if (outputFrames > 0 && outputFrames < frames) {
		frames = outputFrames;
	}
	atomic_fetch_add_explicit(&session->render_frames, frames, memory_order_relaxed);
	if (frames == 0 || inBuf->mData == NULL) {
		mama_tap_zero_output(outOutputData);
		if (outputFrames > 0) {
			atomic_fetch_add_explicit(&session->underruns, outputFrames, memory_order_relaxed);
		}
		return noErr;
	}

	mama_tap_copy_interleaved_output(session, (const Float32*)inBuf->mData, frames, outOutputData);
	if (outputFrames > frames) {
		atomic_fetch_add_explicit(&session->underruns, outputFrames - frames, memory_order_relaxed);
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

static int mama_tap_first_output_stream_index(AudioObjectID deviceID, uint32_t* outIndex, char** outErr) {
	if (outIndex == NULL) {
		return 1;
	}
	*outIndex = 0;
	if (outErr != NULL) {
		*outErr = NULL;
	}

	AudioObjectPropertyAddress address = mama_tap_property_address(
		kAudioDevicePropertyStreams,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain);
	UInt32 size = 0;
	OSStatus st = AudioObjectGetPropertyDataSize(deviceID, &address, 0, NULL, &size);
	if (st != noErr || size < sizeof(AudioObjectID)) {
		mama_tap_set_status_error(outErr, "output device unavailable: enumerate output streams", st);
		return 1;
	}

	UInt32 streamCount = size / sizeof(AudioObjectID);
	AudioObjectID* streamIDs = (AudioObjectID*)malloc(size);
	if (streamIDs == NULL) {
		mama_tap_set_plain_error(outErr, "transient tap failure: allocate output stream list");
		return 1;
	}

	st = AudioObjectGetPropertyData(deviceID, &address, 0, NULL, &size, streamIDs);
	if (st != noErr) {
		free(streamIDs);
		mama_tap_set_status_error(outErr, "output device unavailable: read output streams", st);
		return 1;
	}

	int found = 0;
	for (UInt32 i = 0; i < streamCount; i++) {
		UInt32 direction = 0;
		UInt32 directionSize = (UInt32)sizeof(direction);
		AudioObjectPropertyAddress directionAddress = mama_tap_property_address(
			kAudioStreamPropertyDirection,
			kAudioObjectPropertyScopeGlobal,
			kAudioObjectPropertyElementMain);
		st = AudioObjectGetPropertyData(streamIDs[i], &directionAddress, 0, NULL, &directionSize, &direction);
		if (st != noErr) {
			continue;
		}
		if (direction == 0) {
			*outIndex = i;
			found = 1;
			break;
		}
	}
	free(streamIDs);

	if (!found) {
		mama_tap_set_plain_error(outErr, "output device unavailable: no output stream available on default output device");
		return 1;
	}
	return 0;
}

static int mama_tap_default_output_device_info(char** outUID, uint32_t* outStreamIndex, char** outErr) {
	if (outUID == NULL || outStreamIndex == NULL) {
		return 1;
	}
	*outUID = NULL;
	*outStreamIndex = 0;
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

	uint32_t streamIndex = 0;
	if (mama_tap_first_output_stream_index(deviceID, &streamIndex, outErr) != 0) {
		free(copied);
		return 1;
	}

	*outUID = copied;
	*outStreamIndex = streamIndex;
	return 0;
}

static int mama_tap_wait_for_io_streams(AudioObjectID deviceID, uint32_t timeoutMillis) {
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

static int mama_tap_validate_format(AudioStreamBasicDescription format, char** outErr) {
	if (format.mFormatID != kAudioFormatLinearPCM) {
		mama_tap_set_plain_error(outErr, "unsupported: tap format must be linear PCM");
		return 0;
	}
	if ((format.mFormatFlags & kAudioFormatFlagIsFloat) == 0) {
		mama_tap_set_plain_error(outErr, "unsupported: tap format must be float PCM");
		return 0;
	}
	if (format.mChannelsPerFrame == 0 || format.mBytesPerFrame == 0 || format.mBitsPerChannel != 32) {
		mama_tap_set_plain_error(outErr, "unsupported: tap format is missing PCM frame layout");
		return 0;
	}
	if (format.mBytesPerFrame != format.mChannelsPerFrame * sizeof(Float32)) {
		mama_tap_set_plain_error(outErr, "unsupported: tap format must be interleaved float PCM");
		return 0;
	}
	return 1;
}

static uint32_t mama_tap_default_ring_capacity_frames(AudioStreamBasicDescription format) {
	double sampleRate = format.mSampleRate > 0 ? format.mSampleRate : 48000.0;
	uint32_t frames = (uint32_t)(sampleRate * 2.0);
	if (frames < 4096) {
		frames = 4096;
	}
	if (frames > 262144) {
		frames = 262144;
	}
	return frames;
}

static void mama_tap_cleanup_session(mama_tap_session* session) {
	if (session == NULL) {
		return;
	}
	if (session->aggregate_device_id != kAudioObjectUnknown && session->capture_proc_id != NULL) {
		AudioDeviceStop(session->aggregate_device_id, session->capture_proc_id);
		AudioDeviceDestroyIOProcID(session->aggregate_device_id, session->capture_proc_id);
		session->capture_proc_id = NULL;
	}
	if (session->output_unit != NULL) {
		AudioOutputUnitStop(session->output_unit);
		AudioUnitUninitialize(session->output_unit);
		AudioComponentInstanceDispose(session->output_unit);
		session->output_unit = NULL;
	}
	if (session->aggregate_device_id != kAudioObjectUnknown) {
		AudioHardwareDestroyAggregateDevice(session->aggregate_device_id);
		session->aggregate_device_id = kAudioObjectUnknown;
	}
	if (session->tap_id != kAudioObjectUnknown) {
		AudioHardwareDestroyProcessTap(session->tap_id);
		session->tap_id = kAudioObjectUnknown;
	}
	if (session->ring_samples != NULL) {
		free(session->ring_samples);
		session->ring_samples = NULL;
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
	uint32_t outputStreamIndex,
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

	NSString* bundleIDString = nil;
	if (bundleID != NULL && bundleID[0] != '\0') {
		bundleIDString = [NSString stringWithUTF8String:bundleID];
	}
	NSString* outputDeviceUIDString = [NSString stringWithUTF8String:outputDeviceUID];
	if (outputDeviceUIDString == nil) {
		mama_tap_set_plain_error(outErr, "output device unavailable: invalid output device uid");
		return 1;
	}

	CATapDescription* tapDescription = [[CATapDescription alloc] initWithProcesses:@[@(processObjectID)] andDeviceUID:outputDeviceUIDString withStream:outputStreamIndex];
	if (tapDescription == nil) {
		mama_tap_set_plain_error(outErr, "transient tap failure: create process tap description");
		return 1;
	}
	tapDescription.privateTap = YES;
	tapDescription.muteBehavior = CATapMuted;
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

	AudioStreamBasicDescription tapFormat;
	memset(&tapFormat, 0, sizeof(tapFormat));
	UInt32 size = (UInt32)sizeof(tapFormat);
	AudioObjectPropertyAddress formatAddress = mama_tap_property_address(kAudioTapPropertyFormat, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain);
	st = AudioObjectGetPropertyData(tapID, &formatAddress, 0, NULL, &size, &tapFormat);
	if (st != noErr) {
		AudioHardwareDestroyProcessTap(tapID);
		mama_tap_set_status_error(outErr, "transient tap failure: read process tap format", st);
		return 1;
	}
	if (!mama_tap_validate_format(tapFormat, outErr)) {
		AudioHardwareDestroyProcessTap(tapID);
		return 1;
	}

	CFStringRef tapUIDRef = NULL;
	size = (UInt32)sizeof(tapUIDRef);
	AudioObjectPropertyAddress tapUIDAddress = mama_tap_property_address(kAudioTapPropertyUID, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain);
	st = AudioObjectGetPropertyData(tapID, &tapUIDAddress, 0, NULL, &size, &tapUIDRef);
	if (st != noErr || tapUIDRef == NULL) {
		AudioHardwareDestroyProcessTap(tapID);
		mama_tap_set_status_error(outErr, "transient tap failure: read process tap uid", st);
		return 1;
	}
	NSString* tapUIDString = CFBridgingRelease(tapUIDRef);

	NSDictionary* tapEntryDescription = @{
		@kAudioSubTapUIDKey: tapUIDString,
		@kAudioSubTapDriftCompensationKey: @YES,
	};
	NSDictionary* aggregateDescription = @{
		@kAudioAggregateDeviceUIDKey: [[NSUUID UUID] UUIDString],
		@kAudioAggregateDeviceNameKey: @"MAMA Tap Aggregate",
		@kAudioAggregateDeviceIsPrivateKey: @YES,
		@kAudioAggregateDeviceTapAutoStartKey: @YES,
	};

	AudioObjectID aggregateDeviceID = kAudioObjectUnknown;
	st = AudioHardwareCreateAggregateDevice((__bridge CFDictionaryRef)aggregateDescription, &aggregateDeviceID);
	if (st != noErr) {
		AudioHardwareDestroyProcessTap(tapID);
		mama_tap_set_status_error(outErr, "transient tap failure: create aggregate device", st);
		return 1;
	}

	CFStringRef outputDeviceUIDRef = (__bridge CFStringRef)outputDeviceUIDString;
	CFStringRef tapUIDCF = (__bridge CFStringRef)tapUIDString;
	CFArrayRef subDeviceList = CFArrayCreate(kCFAllocatorDefault, (const void**)&outputDeviceUIDRef, 1, &kCFTypeArrayCallBacks);
	CFArrayRef tapList = CFArrayCreate(kCFAllocatorDefault, (const void**)&tapUIDCF, 1, &kCFTypeArrayCallBacks);
	if (subDeviceList == NULL || tapList == NULL) {
		if (subDeviceList != NULL) {
			CFRelease(subDeviceList);
		}
		if (tapList != NULL) {
			CFRelease(tapList);
		}
		AudioHardwareDestroyAggregateDevice(aggregateDeviceID);
		AudioHardwareDestroyProcessTap(tapID);
		mama_tap_set_plain_error(outErr, "transient tap failure: build aggregate composition");
		return 1;
	}

	AudioObjectPropertyAddress mainSubDeviceAddress = mama_tap_property_address(
		kAudioAggregateDevicePropertyMainSubDevice,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain);
	st = AudioObjectSetPropertyData(aggregateDeviceID, &mainSubDeviceAddress, 0, NULL, (UInt32)sizeof(outputDeviceUIDRef), &outputDeviceUIDRef);
	if (st != noErr) {
		CFRelease(subDeviceList);
		CFRelease(tapList);
		AudioHardwareDestroyAggregateDevice(aggregateDeviceID);
		AudioHardwareDestroyProcessTap(tapID);
		mama_tap_set_status_error(outErr, "transient tap failure: set aggregate main subdevice", st);
		return 1;
	}

	AudioObjectPropertyAddress fullSubDeviceAddress = mama_tap_property_address(
		kAudioAggregateDevicePropertyFullSubDeviceList,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain);
	st = AudioObjectSetPropertyData(aggregateDeviceID, &fullSubDeviceAddress, 0, NULL, (UInt32)sizeof(subDeviceList), &subDeviceList);
	if (st != noErr) {
		CFRelease(subDeviceList);
		CFRelease(tapList);
		AudioHardwareDestroyAggregateDevice(aggregateDeviceID);
		AudioHardwareDestroyProcessTap(tapID);
		mama_tap_set_status_error(outErr, "transient tap failure: set aggregate subdevice list", st);
		return 1;
	}

	AudioObjectPropertyAddress tapListAddress = mama_tap_property_address(
		kAudioAggregateDevicePropertyTapList,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain);
	st = AudioObjectSetPropertyData(aggregateDeviceID, &tapListAddress, 0, NULL, (UInt32)sizeof(tapList), &tapList);
	CFRelease(subDeviceList);
	CFRelease(tapList);
	if (st != noErr) {
		AudioHardwareDestroyAggregateDevice(aggregateDeviceID);
		AudioHardwareDestroyProcessTap(tapID);
		mama_tap_set_status_error(outErr, "transient tap failure: set aggregate tap list", st);
		return 1;
	}

	if (!mama_tap_wait_for_io_streams(aggregateDeviceID, 5000)) {
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
	session->tap_format = tapFormat;
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
	session->output_stream_index = outputStreamIndex;

	st = AudioDeviceCreateIOProcID(aggregateDeviceID, mama_tap_capture_ioproc, session, &session->capture_proc_id);
	if (st != noErr) {
		mama_tap_cleanup_session(session);
		mama_tap_set_status_error(outErr, "transient tap failure: create aggregate io proc", st);
		return 1;
	}

	st = AudioDeviceStart(aggregateDeviceID, session->capture_proc_id);
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

static void mama_tap_session_get_stats(mama_tap_session* session,
	uint64_t* outCaptureCallbacks,
	uint64_t* outCaptureFrames,
	uint64_t* outRenderCallbacks,
	uint64_t* outRenderFrames,
	uint64_t* outUnderruns,
	uint64_t* outOverflows,
	double* outSampleRate,
	uint32_t* outChannels,
	uint32_t* outBitsPerChannel,
	uint32_t* outBytesPerFrame) {
	if (outCaptureCallbacks != NULL) {
		*outCaptureCallbacks = session != NULL ? atomic_load_explicit(&session->capture_callbacks, memory_order_relaxed) : 0;
	}
	if (outCaptureFrames != NULL) {
		*outCaptureFrames = session != NULL ? atomic_load_explicit(&session->capture_frames, memory_order_relaxed) : 0;
	}
	if (outRenderCallbacks != NULL) {
		*outRenderCallbacks = session != NULL ? atomic_load_explicit(&session->render_callbacks, memory_order_relaxed) : 0;
	}
	if (outRenderFrames != NULL) {
		*outRenderFrames = session != NULL ? atomic_load_explicit(&session->render_frames, memory_order_relaxed) : 0;
	}
	if (outUnderruns != NULL) {
		*outUnderruns = session != NULL ? atomic_load_explicit(&session->underruns, memory_order_relaxed) : 0;
	}
	if (outOverflows != NULL) {
		*outOverflows = session != NULL ? atomic_load_explicit(&session->overflows, memory_order_relaxed) : 0;
	}
	if (outSampleRate != NULL) {
		*outSampleRate = session != NULL ? session->tap_format.mSampleRate : 0.0;
	}
	if (outChannels != NULL) {
		*outChannels = session != NULL ? session->tap_format.mChannelsPerFrame : 0;
	}
	if (outBitsPerChannel != NULL) {
		*outBitsPerChannel = session != NULL ? session->tap_format.mBitsPerChannel : 0;
	}
	if (outBytesPerFrame != NULL) {
		*outBytesPerFrame = session != NULL ? session->tap_format.mBytesPerFrame : 0;
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
	(void)inObjectID;
	(void)inNumberAddresses;
	(void)inAddresses;
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
	StateKey      string
	RuntimeKey    string
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
	Callbacks       uint64
	Frames          uint64
	RenderCallbacks uint64
	RenderFrames    uint64
	Underruns       uint64
	Overflows       uint64
	Attached        bool
	OutputDeviceID  string
	RestoreCapable  bool
	MatchCount      int
	SampleRate      float64
	Channels        int
	BitsPerChannel  int
	BytesPerFrame   int
}

type darwinTapOutputRoute struct {
	DeviceUID   string
	StreamIndex int
}

type darwinTapBridge interface {
	Supported() bool
	EnsureTap(target darwinTapTarget) error
	ReleaseTap(stateKey string) error
	SetGain(stateKey string, volume int) error
	SetMuted(stateKey string, muted bool) error
	ReadVirtualState(stateKey string) darwinTapState
	Reconcile(targets []darwinTapTarget) error
	Stats(stateKey string) (darwinTapStats, error)
}

type darwinTapRuntime interface {
	Supported() bool
	DefaultOutputRoute() (darwinTapOutputRoute, error)
	CreateSession(target darwinTapTarget, outputRoute darwinTapOutputRoute, volume int, muted bool) (darwinTapRuntimeSession, error)
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
	stateKey       string
	runtimeKey     string
	processObject  uint32
	bundleID       string
	outputRoute    darwinTapOutputRoute
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

func (defaultDarwinTapRuntime) DefaultOutputRoute() (darwinTapOutputRoute, error) {
	var cUID *C.char
	var cStreamIndex C.uint32_t
	var cErr *C.char
	result := C.mama_tap_default_output_device_info(&cUID, &cStreamIndex, &cErr)
	if cErr != nil {
		defer C.mama_tap_free_error(cErr)
	}
	if cUID != nil {
		defer C.free(unsafe.Pointer(cUID))
	}
	if result != 0 {
		if cErr != nil {
			return darwinTapOutputRoute{}, wrapDarwinTapError("darwin tap default output device", errors.New(C.GoString(cErr)))
		}
		return darwinTapOutputRoute{}, wrapDarwinTapError("darwin tap default output device", errors.New("output device unavailable"))
	}
	return darwinTapOutputRoute{
		DeviceUID:   strings.TrimSpace(C.GoString(cUID)),
		StreamIndex: int(cStreamIndex),
	}, nil
}

func (defaultDarwinTapRuntime) CreateSession(target darwinTapTarget, outputRoute darwinTapOutputRoute, volume int, muted bool) (darwinTapRuntimeSession, error) {
	if target.ProcessObject == 0 {
		return nil, wrapDarwinTapError("darwin tap session create", errors.New("target unavailable: missing process object"))
	}
	if strings.TrimSpace(outputRoute.DeviceUID) == "" {
		return nil, wrapDarwinTapError("darwin tap session create", errors.New("output device unavailable: missing output device uid"))
	}
	if outputRoute.StreamIndex < 0 {
		return nil, wrapDarwinTapError("darwin tap session create", errors.New("output device unavailable: invalid output stream index"))
	}

	cOutputUID := C.CString(outputRoute.DeviceUID)
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
		C.uint32_t(outputRoute.StreamIndex),
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
	var captureCallbacks C.uint64_t
	var captureFrames C.uint64_t
	var renderCallbacks C.uint64_t
	var renderFrames C.uint64_t
	var underruns C.uint64_t
	var overflows C.uint64_t
	var sampleRate C.double
	var channels C.uint32_t
	var bitsPerChannel C.uint32_t
	var bytesPerFrame C.uint32_t
	C.mama_tap_session_get_stats(
		s.session,
		&captureCallbacks,
		&captureFrames,
		&renderCallbacks,
		&renderFrames,
		&underruns,
		&overflows,
		&sampleRate,
		&channels,
		&bitsPerChannel,
		&bytesPerFrame,
	)
	return darwinTapStats{
		Callbacks:       uint64(captureCallbacks),
		Frames:          uint64(captureFrames),
		RenderCallbacks: uint64(renderCallbacks),
		RenderFrames:    uint64(renderFrames),
		Underruns:       uint64(underruns),
		Overflows:       uint64(overflows),
		RestoreCapable:  s.restoreCapable,
		SampleRate:      float64(sampleRate),
		Channels:        int(channels),
		BitsPerChannel:  int(bitsPerChannel),
		BytesPerFrame:   int(bytesPerFrame),
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
		strings.Contains(message, "not permitted"):
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
	if strings.TrimSpace(target.StateKey) == "" || strings.TrimSpace(target.RuntimeKey) == "" || target.ProcessObject == 0 {
		return wrapDarwinTapError("darwin tap ensure", errors.New("target unavailable: missing tap target identity"))
	}
	b.ensureOutputWatcher()

	outputRoute, err := b.runtime.DefaultOutputRoute()
	if err != nil {
		err = wrapDarwinTapError("darwin tap default output device", err)
		b.log("darwin_tap_attach_failed", runtimelog.Fields{
			"error":          err,
			"state_key":      target.StateKey,
			"runtime_key":    target.RuntimeKey,
			"bundle_id":      target.BundleID,
			"process_object": target.ProcessObject,
			"reason":         "default_output_lookup",
		})
		return err
	}

	b.mu.Lock()
	state := b.rawStateLocked(target.StateKey)
	state.Wanted = true
	b.states[target.StateKey] = state
	b.liveTargets[target.RuntimeKey] = target
	if b.handleMatchesLocked(target, outputRoute) {
		b.mu.Unlock()
		return nil
	}
	if orphanKey, orphanHandle := b.findRestoreHandleLocked(target, outputRoute); orphanHandle != nil {
		delete(b.handles, orphanKey)
		orphanHandle.runtimeKey = target.RuntimeKey
		orphanHandle.stateKey = target.StateKey
		orphanHandle.processObject = target.ProcessObject
		orphanHandle.bundleID = strings.TrimSpace(target.BundleID)
		orphanHandle.outputRoute = outputRoute
		b.handles[target.RuntimeKey] = orphanHandle
		b.mu.Unlock()
		b.log("darwin_tap_restart_recovered", runtimelog.Fields{
			"state_key":      target.StateKey,
			"runtime_key":    target.RuntimeKey,
			"bundle_id":      target.BundleID,
			"output_device":  outputRoute.DeviceUID,
			"output_stream":  outputRoute.StreamIndex,
			"process_object": target.ProcessObject,
			"restore":        orphanHandle.restoreCapable,
		})
		return nil
	}
	b.closeHandleLocked(target.RuntimeKey, "replace_attach")
	desired := b.rawStateLocked(target.StateKey)
	b.mu.Unlock()

	session, err := b.runtime.CreateSession(target, outputRoute, desired.Volume, desired.Muted)
	if err != nil {
		err = wrapDarwinTapError("darwin tap session create", err)
		b.mu.Lock()
		state = b.rawStateLocked(target.StateKey)
		state.Attached = b.stateAttachedLocked(target.StateKey)
		b.states[target.StateKey] = state
		b.mu.Unlock()
		b.log("darwin_tap_attach_failed", runtimelog.Fields{
			"error":          err,
			"state_key":      target.StateKey,
			"runtime_key":    target.RuntimeKey,
			"bundle_id":      target.BundleID,
			"output_device":  outputRoute.DeviceUID,
			"output_stream":  outputRoute.StreamIndex,
			"process_object": target.ProcessObject,
		})
		return err
	}

	handle := &darwinTapHandle{
		session:        session,
		stateKey:       target.StateKey,
		runtimeKey:     target.RuntimeKey,
		processObject:  target.ProcessObject,
		bundleID:       strings.TrimSpace(target.BundleID),
		outputRoute:    outputRoute,
		restoreCapable: session.RestoreCapable(),
	}

	b.mu.Lock()
	if existing := b.handles[target.RuntimeKey]; existing != nil {
		b.closeHandleLocked(target.RuntimeKey, "replace_attach")
	}
	b.handles[target.RuntimeKey] = handle
	b.mu.Unlock()

	b.log("darwin_tap_created", runtimelog.Fields{
		"state_key":      target.StateKey,
		"runtime_key":    target.RuntimeKey,
		"bundle_id":      target.BundleID,
		"output_device":  outputRoute.DeviceUID,
		"output_stream":  outputRoute.StreamIndex,
		"process_object": target.ProcessObject,
		"restore":        handle.restoreCapable,
	})
	b.log("darwin_tap_renderer_created", runtimelog.Fields{
		"state_key":      target.StateKey,
		"runtime_key":    target.RuntimeKey,
		"output_device":  outputRoute.DeviceUID,
		"output_stream":  outputRoute.StreamIndex,
		"process_object": target.ProcessObject,
	})
	b.log("darwin_tap_capture_started", runtimelog.Fields{
		"state_key":      target.StateKey,
		"runtime_key":    target.RuntimeKey,
		"output_device":  outputRoute.DeviceUID,
		"output_stream":  outputRoute.StreamIndex,
		"process_object": target.ProcessObject,
	})
	return nil
}

func (b *defaultDarwinTapBridge) ReleaseTap(stateKey string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for runtimeKey, target := range b.liveTargets {
		if target.StateKey == stateKey {
			delete(b.liveTargets, runtimeKey)
		}
	}
	for runtimeKey, handle := range b.handles {
		if handle.stateKey == stateKey {
			b.closeHandleLocked(runtimeKey, "release")
		}
	}
	state := b.rawStateLocked(stateKey)
	state.Wanted = false
	state.Attached = false
	b.states[stateKey] = state
	return nil
}

func (b *defaultDarwinTapBridge) SetGain(stateKey string, volume int) error {
	return b.applyToState(stateKey, "set_gain", func(state darwinTapState, handle *darwinTapHandle) error {
		return handle.session.SetGain(state.Volume)
	}, func(state darwinTapState) darwinTapState {
		state.Volume = clampPercent(volume)
		return state
	})
}

func (b *defaultDarwinTapBridge) SetMuted(stateKey string, muted bool) error {
	return b.applyToState(stateKey, "set_mute", func(state darwinTapState, handle *darwinTapHandle) error {
		return handle.session.SetMuted(state.Muted)
	}, func(state darwinTapState) darwinTapState {
		state.Muted = muted
		return state
	})
}

func (b *defaultDarwinTapBridge) ReadVirtualState(stateKey string) darwinTapState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.stateLocked(stateKey)
}

func (b *defaultDarwinTapBridge) Reconcile(targets []darwinTapTarget) error {
	if !b.Supported() {
		return nil
	}
	b.ensureOutputWatcher()

	outputRoute, err := b.runtime.DefaultOutputRoute()
	if err != nil {
		return wrapDarwinTapError("darwin tap default output device", err)
	}

	targetsByRuntimeKey := make(map[string]darwinTapTarget, len(targets))
	for _, target := range targets {
		if strings.TrimSpace(target.StateKey) == "" || strings.TrimSpace(target.RuntimeKey) == "" {
			continue
		}
		targetsByRuntimeKey[target.RuntimeKey] = target
	}

	b.mu.Lock()
	b.liveTargets = cloneDarwinTapTargetsMap(targetsByRuntimeKey)

	for runtimeKey, handle := range b.handles {
		target, ok := targetsByRuntimeKey[runtimeKey]
		if !ok {
			if handle.restoreCapable && handle.outputRoute == outputRoute && strings.TrimSpace(handle.bundleID) != "" {
				continue
			}
			b.closeHandleLocked(runtimeKey, "target_gone")
			continue
		}
		if b.handleMatchesForTargetLocked(handle, target, outputRoute) {
			continue
		}
		b.closeHandleLocked(runtimeKey, "reconcile_mismatch")
	}

	type createRecord struct {
		target darwinTapTarget
		handle *darwinTapHandle
	}
	var toCreate []darwinTapTarget
	var recovered []createRecord
	for _, target := range targetsByRuntimeKey {
		if b.handleMatchesLocked(target, outputRoute) {
			continue
		}
		if orphanKey, orphanHandle := b.findRestoreHandleLocked(target, outputRoute); orphanHandle != nil {
			delete(b.handles, orphanKey)
			orphanHandle.runtimeKey = target.RuntimeKey
			orphanHandle.stateKey = target.StateKey
			orphanHandle.processObject = target.ProcessObject
			orphanHandle.bundleID = strings.TrimSpace(target.BundleID)
			orphanHandle.outputRoute = outputRoute
			b.handles[target.RuntimeKey] = orphanHandle
			recovered = append(recovered, createRecord{target: target, handle: orphanHandle})
			continue
		}
		if !b.rawStateLocked(target.StateKey).Wanted {
			continue
		}
		toCreate = append(toCreate, target)
	}
	b.mu.Unlock()

	for _, record := range recovered {
		b.log("darwin_tap_restart_recovered", runtimelog.Fields{
			"state_key":      record.target.StateKey,
			"runtime_key":    record.target.RuntimeKey,
			"bundle_id":      record.target.BundleID,
			"output_device":  record.handle.outputRoute.DeviceUID,
			"output_stream":  record.handle.outputRoute.StreamIndex,
			"process_object": record.target.ProcessObject,
			"restore":        record.handle.restoreCapable,
		})
	}

	var firstErr error
	for _, target := range toCreate {
		if err := b.EnsureTap(target); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		b.log("darwin_tap_attach_failed", runtimelog.Fields{
			"error":  firstErr,
			"reason": "reconcile_create",
		})
	}
	return firstErr
}

func (b *defaultDarwinTapBridge) Stats(stateKey string) (darwinTapStats, error) {
	b.mu.Lock()
	handles := b.handlesForStateLocked(stateKey)
	matchCount := len(b.liveTargetsForStateLocked(stateKey))
	b.mu.Unlock()

	if len(handles) == 0 {
		state := b.ReadVirtualState(stateKey)
		return darwinTapStats{Attached: state.Attached}, wrapDarwinTapError("darwin tap stats", errors.New("target unavailable: tap session not attached"))
	}

	var combined darwinTapStats
	for _, handle := range handles {
		stats, err := handle.session.Stats()
		if err != nil {
			return darwinTapStats{}, wrapDarwinTapError("darwin tap stats", err)
		}
		combined.Callbacks += stats.Callbacks
		combined.Frames += stats.Frames
		combined.RenderCallbacks += stats.RenderCallbacks
		combined.RenderFrames += stats.RenderFrames
		combined.Underruns += stats.Underruns
		combined.Overflows += stats.Overflows
		if combined.SampleRate == 0 {
			combined.SampleRate = stats.SampleRate
		}
		if combined.Channels == 0 {
			combined.Channels = stats.Channels
		}
		if combined.BitsPerChannel == 0 {
			combined.BitsPerChannel = stats.BitsPerChannel
		}
		if combined.BytesPerFrame == 0 {
			combined.BytesPerFrame = stats.BytesPerFrame
		}
		if combined.OutputDeviceID == "" {
			combined.OutputDeviceID = handle.outputRoute.DeviceUID
		}
		combined.RestoreCapable = combined.RestoreCapable || handle.restoreCapable
	}
	combined.Attached = b.ReadVirtualState(stateKey).Attached
	combined.MatchCount = matchCount
	return combined, nil
}

func (b *defaultDarwinTapBridge) applyToState(stateKey string, reason string, fn func(darwinTapState, *darwinTapHandle) error, mutate func(darwinTapState) darwinTapState) error {
	b.mu.Lock()
	state := b.rawStateLocked(stateKey)
	state = mutate(state)
	state.Wanted = true
	b.states[stateKey] = state
	handles := b.handlesForStateLocked(stateKey)
	b.mu.Unlock()

	if len(handles) == 0 {
		return wrapDarwinTapError("darwin tap "+reason, errors.New("target unavailable: tap session not attached"))
	}

	var firstErr error
	var failed []*darwinTapHandle
	for _, handle := range handles {
		if err := fn(state, handle); err != nil {
			if firstErr == nil {
				firstErr = wrapDarwinTapError("darwin tap "+reason, err)
			}
			failed = append(failed, handle)
		}
	}

	if len(failed) == 0 {
		return nil
	}

	b.mu.Lock()
	for _, handle := range failed {
		current := b.handles[handle.runtimeKey]
		if current == handle {
			b.closeHandleLocked(handle.runtimeKey, reason)
		}
	}
	b.mu.Unlock()

	if firstErr != nil {
		b.log("darwin_tap_attach_failed", runtimelog.Fields{
			"error":     firstErr,
			"state_key": stateKey,
			"reason":    reason,
		})
	}
	return firstErr
}

func (b *defaultDarwinTapBridge) rawStateLocked(stateKey string) darwinTapState {
	if stateKey == "" {
		return darwinTapState{Volume: 100}
	}
	if state, ok := b.states[stateKey]; ok {
		state.Volume = clampPercent(state.Volume)
		return state
	}
	return darwinTapState{Volume: 100}
}

func (b *defaultDarwinTapBridge) stateLocked(stateKey string) darwinTapState {
	state := b.rawStateLocked(stateKey)
	state.Attached = b.stateAttachedLocked(stateKey)
	return state
}

func (b *defaultDarwinTapBridge) stateAttachedLocked(stateKey string) bool {
	liveTargets := b.liveTargetsForStateLocked(stateKey)
	if len(liveTargets) == 0 {
		return false
	}
	for _, target := range liveTargets {
		handle := b.handles[target.RuntimeKey]
		if handle == nil || !b.handleMatchesForTargetLocked(handle, target, handle.outputRoute) {
			return false
		}
	}
	return true
}

func (b *defaultDarwinTapBridge) liveTargetsForStateLocked(stateKey string) []darwinTapTarget {
	if stateKey == "" {
		return nil
	}
	targets := make([]darwinTapTarget, 0, len(b.liveTargets))
	for _, target := range b.liveTargets {
		if target.StateKey == stateKey {
			targets = append(targets, target)
		}
	}
	return targets
}

func (b *defaultDarwinTapBridge) handlesForStateLocked(stateKey string) []*darwinTapHandle {
	if stateKey == "" {
		return nil
	}
	handles := make([]*darwinTapHandle, 0, len(b.handles))
	for _, handle := range b.handles {
		if handle != nil && handle.stateKey == stateKey && handle.session != nil {
			handles = append(handles, handle)
		}
	}
	return handles
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

func (b *defaultDarwinTapBridge) handleMatchesLocked(target darwinTapTarget, outputRoute darwinTapOutputRoute) bool {
	handle := b.handles[target.RuntimeKey]
	return b.handleMatchesForTargetLocked(handle, target, outputRoute)
}

func (b *defaultDarwinTapBridge) handleMatchesForTargetLocked(handle *darwinTapHandle, target darwinTapTarget, outputRoute darwinTapOutputRoute) bool {
	if handle == nil || handle.session == nil {
		return false
	}
	if handle.stateKey != target.StateKey || handle.outputRoute != outputRoute {
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

func (b *defaultDarwinTapBridge) findRestoreHandleLocked(target darwinTapTarget, outputRoute darwinTapOutputRoute) (string, *darwinTapHandle) {
	targetBundleID := strings.TrimSpace(target.BundleID)
	if targetBundleID == "" {
		return "", nil
	}
	for runtimeKey, handle := range b.handles {
		if handle == nil || handle.session == nil || !handle.restoreCapable {
			continue
		}
		if handle.stateKey != target.StateKey || handle.outputRoute != outputRoute {
			continue
		}
		if strings.EqualFold(handle.bundleID, targetBundleID) {
			return runtimeKey, handle
		}
	}
	return "", nil
}

func (b *defaultDarwinTapBridge) closeHandleLocked(runtimeKey string, reason string) {
	handle := b.handles[runtimeKey]
	if handle == nil {
		return
	}
	stats, statsErr := handle.session.Stats()
	if statsErr == nil && (stats.Underruns > 0 || stats.Overflows > 0) {
		b.log("darwin_tap_buffer_summary", runtimelog.Fields{
			"state_key":       handle.stateKey,
			"runtime_key":     runtimeKey,
			"underruns":       stats.Underruns,
			"overflows":       stats.Overflows,
			"capture_frames":  stats.Frames,
			"render_frames":   stats.RenderFrames,
			"capture_calls":   stats.Callbacks,
			"render_calls":    stats.RenderCallbacks,
			"output_device":   handle.outputRoute.DeviceUID,
			"output_stream":   handle.outputRoute.StreamIndex,
			"process_object":  handle.processObject,
			"restore_capable": handle.restoreCapable,
		})
	}
	_ = handle.session.Close()
	b.log("darwin_tap_capture_stopped", runtimelog.Fields{
		"state_key":      handle.stateKey,
		"runtime_key":    runtimeKey,
		"output_device":  handle.outputRoute.DeviceUID,
		"output_stream":  handle.outputRoute.StreamIndex,
		"process_object": handle.processObject,
		"reason":         reason,
	})
	b.log("darwin_tap_renderer_destroyed", runtimelog.Fields{
		"state_key":      handle.stateKey,
		"runtime_key":    runtimeKey,
		"output_device":  handle.outputRoute.DeviceUID,
		"output_stream":  handle.outputRoute.StreamIndex,
		"process_object": handle.processObject,
		"reason":         reason,
	})
	b.log("darwin_tap_destroyed", runtimelog.Fields{
		"state_key":     handle.stateKey,
		"runtime_key":   runtimeKey,
		"bundle_id":     handle.bundleID,
		"output_device": handle.outputRoute.DeviceUID,
		"output_stream": handle.outputRoute.StreamIndex,
		"reason":        reason,
		"restore":       handle.restoreCapable,
	})
	delete(b.handles, runtimeKey)
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
	BundleID              string
	Selector              string
	StateKey              string
	MatchedProcessObjects []uint32
	Volume                int
	Muted                 bool
	Callbacks             uint64
	Frames                uint64
	RenderCallbacks       uint64
	RenderFrames          uint64
	Underruns             uint64
	Overflows             uint64
	OutputDeviceUID       string
	RestoreCapable        bool
	SampleRate            float64
	Channels              int
	BitsPerChannel        int
	BytesPerFrame         int
	Duration              time.Duration
}

func RunDarwinTapProbe(bundleID string, duration time.Duration, volume int, muted bool) (DarwinTapProbeResult, error) {
	return RunDarwinTapProbeSelector("", bundleID, duration, volume, muted)
}

func RunDarwinTapProbeSelector(selectorToken, bundleID string, duration time.Duration, volume int, muted bool) (DarwinTapProbeResult, error) {
	if duration <= 0 {
		duration = 5 * time.Second
	}

	discovery := newCoreAudioBridge()
	processes, err := discovery.ListProcesses()
	if err != nil {
		return DarwinTapProbeResult{}, err
	}

	sessions := make([]darwinAppSession, 0, len(processes))
	for _, process := range processes {
		session, ok := buildDarwinSession(process)
		if !ok {
			continue
		}
		sessions = append(sessions, session)
	}

	logicalApps := buildDarwinLogicalApps(sessions, nil)
	target, err := resolveDarwinTapProbeTarget(logicalApps, selectorToken, bundleID)
	if err != nil {
		return DarwinTapProbeResult{}, err
	}

	tapBridge, ok := newDarwinTapBridge().(*defaultDarwinTapBridge)
	if !ok || !tapBridge.Supported() {
		return DarwinTapProbeResult{}, wrapDarwinTapError("darwin tap probe", errors.New("unsupported: darwin process taps require macOS 14.2+"))
	}
	defer func() {
		_ = tapBridge.ReleaseTap(target.stateKey)
	}()

	for _, session := range target.sessions {
		if err := tapBridge.EnsureTap(session.tapTarget()); err != nil {
			return DarwinTapProbeResult{}, err
		}
	}
	if err := tapBridge.SetGain(target.stateKey, volume); err != nil {
		return DarwinTapProbeResult{}, err
	}
	if err := tapBridge.SetMuted(target.stateKey, muted); err != nil {
		return DarwinTapProbeResult{}, err
	}

	time.Sleep(duration)

	state := tapBridge.ReadVirtualState(target.stateKey)
	stats, err := tapBridge.Stats(target.stateKey)
	if err != nil {
		return DarwinTapProbeResult{}, err
	}
	if stats.Frames == 0 {
		return DarwinTapProbeResult{}, fmt.Errorf("darwin tap probe observed no captured audio frames for %q", target.name)
	}
	if stats.RenderFrames == 0 {
		return DarwinTapProbeResult{}, fmt.Errorf("darwin tap probe observed no rendered audio frames for %q", target.name)
	}

	processObjects := make([]uint32, 0, len(target.sessions))
	for _, session := range target.sessions {
		processObjects = append(processObjects, session.processObject)
	}

	return DarwinTapProbeResult{
		BundleID:              target.bundleID,
		Selector:              selectorToken,
		StateKey:              target.stateKey,
		MatchedProcessObjects: processObjects,
		Volume:                state.Volume,
		Muted:                 state.Muted,
		Callbacks:             stats.Callbacks,
		Frames:                stats.Frames,
		RenderCallbacks:       stats.RenderCallbacks,
		RenderFrames:          stats.RenderFrames,
		Underruns:             stats.Underruns,
		Overflows:             stats.Overflows,
		OutputDeviceUID:       stats.OutputDeviceID,
		RestoreCapable:        stats.RestoreCapable,
		SampleRate:            stats.SampleRate,
		Channels:              stats.Channels,
		BitsPerChannel:        stats.BitsPerChannel,
		BytesPerFrame:         stats.BytesPerFrame,
		Duration:              duration,
	}, nil
}

func resolveDarwinTapProbeTarget(apps []darwinLogicalApp, selectorToken, bundleID string) (darwinLogicalApp, error) {
	if selector := strings.TrimSpace(selectorToken); selector != "" {
		parsed := parseSelectorToken(selector)
		for _, app := range apps {
			if darwinLogicalAppMatchesSelector(app, parsed) {
				return app, nil
			}
		}
		return darwinLogicalApp{}, fmt.Errorf("%w: no running process found for selector %q", ErrTargetUnavailable, selector)
	}
	bundleID = strings.TrimSpace(bundleID)
	for _, app := range apps {
		if strings.EqualFold(strings.TrimSpace(app.bundleID), bundleID) {
			return app, nil
		}
	}
	return darwinLogicalApp{}, fmt.Errorf("%w: no running process found for bundle %q", ErrTargetUnavailable, bundleID)
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
