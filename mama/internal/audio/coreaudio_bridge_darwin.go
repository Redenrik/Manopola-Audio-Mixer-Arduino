//go:build darwin && cgo

package audio

/*
#cgo darwin LDFLAGS: -framework CoreAudio -framework CoreFoundation
#include <CoreAudio/CoreAudio.h>
#include <CoreFoundation/CoreFoundation.h>
#include <libproc.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

typedef struct {
	uint32_t object_id;
	int32_t pid;
	uint8_t running_output;
	char* bundle_id;
	char* executable_path;
} mama_coreaudio_process_info;

enum {
	kMAMAAudioPropertyVirtualMainVolume = 'vmvc',
	kMAMAAudioPropertyProcessIsAudible = 'pmut',
};

typedef struct {
	AudioObjectPropertyAddress items[32];
	uint32_t count;
} mama_property_address_list;

typedef struct {
	AudioObjectID* items;
	uint32_t count;
} mama_object_id_list;

static void mama_set_error(char** outErr, const char* msg, OSStatus st) {
	if (outErr == NULL) {
		return;
	}
	char buffer[256];
	snprintf(buffer, sizeof(buffer), "%s (osstatus=%d)", msg, (int)st);
	*outErr = strdup(buffer);
}

static char* mama_copy_cfstring(CFStringRef value) {
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

static int mama_get_process_pid(AudioObjectID processObjectID, pid_t* outPID) {
	if (outPID == NULL) {
		return 0;
	}
	UInt32 pid = 0;
	UInt32 size = (UInt32)sizeof(pid);
	AudioObjectPropertyAddress addr = {
		kAudioProcessPropertyPID,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};
	OSStatus st = AudioObjectGetPropertyData(processObjectID, &addr, 0, NULL, &size, &pid);
	if (st != noErr) {
		return 0;
	}
	*outPID = (pid_t)pid;
	return 1;
}

static uint8_t mama_is_process_running_output(AudioObjectID processObjectID) {
	UInt32 running = 0;
	UInt32 size = (UInt32)sizeof(running);
	AudioObjectPropertyAddress addr = {
		kAudioProcessPropertyIsRunningOutput,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};
	OSStatus st = AudioObjectGetPropertyData(processObjectID, &addr, 0, NULL, &size, &running);
	if (st != noErr) {
		return 0;
	}
	return running ? 1 : 0;
}

static char* mama_get_process_bundle_id(AudioObjectID processObjectID) {
	CFStringRef bundleID = NULL;
	UInt32 size = (UInt32)sizeof(bundleID);
	AudioObjectPropertyAddress addr = {
		kAudioProcessPropertyBundleID,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};
	OSStatus st = AudioObjectGetPropertyData(processObjectID, &addr, 0, NULL, &size, &bundleID);
	if (st != noErr || bundleID == NULL) {
		return NULL;
	}
	char* value = mama_copy_cfstring(bundleID);
	CFRelease(bundleID);
	return value;
}

static char* mama_get_executable_path(pid_t pid) {
	char path[PROC_PIDPATHINFO_MAXSIZE];
	memset(path, 0, sizeof(path));
	int length = proc_pidpath(pid, path, sizeof(path));
	if (length <= 0) {
		return NULL;
	}
	return strdup(path);
}

int mama_coreaudio_list_processes(mama_coreaudio_process_info** outProcesses, uint32_t* outCount, char** outErr) {
	if (outProcesses == NULL || outCount == NULL) {
		return 1;
	}
	*outProcesses = NULL;
	*outCount = 0;
	if (outErr != NULL) {
		*outErr = NULL;
	}

	AudioObjectPropertyAddress addr = {
		kAudioHardwarePropertyProcessObjectList,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};

	UInt32 dataSize = 0;
	OSStatus st = AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &addr, 0, NULL, &dataSize);
	if (st != noErr) {
		if (st == kAudioHardwareUnknownPropertyError) {
			return 0;
		}
		mama_set_error(outErr, "read process object list size", st);
		return 1;
	}
	if (dataSize == 0) {
		return 0;
	}

	uint32_t objectCount = dataSize / sizeof(AudioObjectID);
	AudioObjectID* objectIDs = (AudioObjectID*)malloc(dataSize);
	if (objectIDs == NULL) {
		if (outErr != NULL) {
			*outErr = strdup("allocate process object list buffer");
		}
		return 1;
	}

	st = AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &dataSize, objectIDs);
	if (st != noErr) {
		free(objectIDs);
		mama_set_error(outErr, "read process object list", st);
		return 1;
	}

	mama_coreaudio_process_info* processes = (mama_coreaudio_process_info*)calloc(objectCount, sizeof(mama_coreaudio_process_info));
	if (processes == NULL) {
		free(objectIDs);
		if (outErr != NULL) {
			*outErr = strdup("allocate process info buffer");
		}
		return 1;
	}

	uint32_t n = 0;
	for (uint32_t i = 0; i < objectCount; i++) {
		AudioObjectID processObjectID = objectIDs[i];
		pid_t pid = 0;
		if (!mama_get_process_pid(processObjectID, &pid) || pid <= 0) {
			continue;
		}
		processes[n].object_id = (uint32_t)processObjectID;
		processes[n].pid = (int32_t)pid;
		processes[n].running_output = mama_is_process_running_output(processObjectID);
		processes[n].bundle_id = mama_get_process_bundle_id(processObjectID);
		processes[n].executable_path = mama_get_executable_path(pid);
		n++;
	}
	free(objectIDs);

	if (n == 0) {
		free(processes);
		return 0;
	}

	*outProcesses = processes;
	*outCount = n;
	return 0;
}

void mama_coreaudio_free_processes(mama_coreaudio_process_info* processes, uint32_t count) {
	if (processes == NULL) {
		return;
	}
	for (uint32_t i = 0; i < count; i++) {
		if (processes[i].bundle_id != NULL) {
			free(processes[i].bundle_id);
		}
		if (processes[i].executable_path != NULL) {
			free(processes[i].executable_path);
		}
	}
	free(processes);
}

void mama_coreaudio_free_error(char* err) {
	if (err != NULL) {
		free(err);
	}
}

static void mama_collect_property_addresses(AudioObjectID objectID, AudioObjectPropertySelector selector, mama_property_address_list* outList) {
	if (outList == NULL) {
		return;
	}
	outList->count = 0;
	AudioObjectPropertyScope scopes[] = {
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyScopeOutput,
	};
	AudioObjectPropertyElement elements[] = {
		kAudioObjectPropertyElementMain,
		1, 2, 3, 4, 5, 6, 7, 8,
	};
	for (uint32_t si = 0; si < sizeof(scopes)/sizeof(scopes[0]); si++) {
		for (uint32_t ei = 0; ei < sizeof(elements)/sizeof(elements[0]); ei++) {
			if (outList->count >= sizeof(outList->items)/sizeof(outList->items[0])) {
				return;
			}
			AudioObjectPropertyAddress addr = { selector, scopes[si], elements[ei] };
			if (!AudioObjectHasProperty(objectID, &addr)) {
				continue;
			}
			int duplicate = 0;
			for (uint32_t i = 0; i < outList->count; i++) {
				AudioObjectPropertyAddress existing = outList->items[i];
				if (existing.mSelector == addr.mSelector && existing.mScope == addr.mScope && existing.mElement == addr.mElement) {
					duplicate = 1;
					break;
				}
			}
			if (duplicate) {
				continue;
			}
			outList->items[outList->count++] = addr;
		}
	}
}

static int mama_get_object_class(AudioObjectID objectID, AudioClassID* outClass) {
	if (outClass == NULL) {
		return 0;
	}
	AudioClassID classID = 0;
	UInt32 size = (UInt32)sizeof(classID);
	AudioObjectPropertyAddress addr = {
		kAudioObjectPropertyClass,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};
	OSStatus st = AudioObjectGetPropertyData(objectID, &addr, 0, NULL, &size, &classID);
	if (st != noErr) {
		return 0;
	}
	*outClass = classID;
	return 1;
}

static int mama_class_is_or_inherits(AudioObjectID objectID, AudioClassID targetClass) {
	AudioClassID classID = 0;
	if (!mama_get_object_class(objectID, &classID)) {
		return 0;
	}
	for (uint32_t depth = 0; depth < 16; depth++) {
		if (classID == targetClass) {
			return 1;
		}
		if (classID == 0 || classID == kAudioObjectClassID) {
			return 0;
		}

		AudioClassID baseClass = 0;
		UInt32 size = (UInt32)sizeof(baseClass);
		AudioObjectPropertyAddress addr = {
			kAudioObjectPropertyBaseClass,
			kAudioObjectPropertyScopeGlobal,
			kAudioObjectPropertyElementMain,
		};
		OSStatus st = AudioObjectGetPropertyData(objectID, &addr, 0, NULL, &size, &baseClass);
		if (st != noErr || baseClass == 0 || baseClass == classID) {
			return 0;
		}
		classID = baseClass;
	}
	return 0;
}

static int mama_get_control_scope(AudioObjectID controlID, AudioObjectPropertyScope* outScope) {
	if (outScope == NULL) {
		return 0;
	}
	AudioObjectPropertyScope scope = kAudioObjectPropertyScopeGlobal;
	UInt32 size = (UInt32)sizeof(scope);
	AudioObjectPropertyAddress addr = {
		kAudioControlPropertyScope,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};
	OSStatus st = AudioObjectGetPropertyData(controlID, &addr, 0, NULL, &size, &scope);
	if (st != noErr) {
		return 0;
	}
	*outScope = scope;
	return 1;
}

static int mama_get_owned_objects(AudioObjectID objectID, mama_object_id_list* outList) {
	if (outList == NULL) {
		return 0;
	}
	outList->items = NULL;
	outList->count = 0;

	AudioObjectPropertyAddress addr = {
		kAudioObjectPropertyOwnedObjects,
		kAudioObjectPropertyScopeGlobal,
		kAudioObjectPropertyElementMain,
	};
	UInt32 dataSize = 0;
	OSStatus st = AudioObjectGetPropertyDataSize(objectID, &addr, 0, NULL, &dataSize);
	if (st != noErr || dataSize == 0) {
		return 0;
	}

	AudioObjectID* owned = (AudioObjectID*)malloc(dataSize);
	if (owned == NULL) {
		return 0;
	}
	st = AudioObjectGetPropertyData(objectID, &addr, 0, NULL, &dataSize, owned);
	if (st != noErr) {
		free(owned);
		return 0;
	}

	outList->items = owned;
	outList->count = dataSize / sizeof(AudioObjectID);
	return outList->count > 0;
}

static void mama_free_object_id_list(mama_object_id_list* list) {
	if (list == NULL) {
		return;
	}
	if (list->items != NULL) {
		free(list->items);
	}
	list->items = NULL;
	list->count = 0;
}

static int mama_collect_process_controls(AudioObjectID processObjectID, AudioClassID targetClass, mama_object_id_list* outControls) {
	if (outControls == NULL) {
		return 0;
	}
	outControls->items = NULL;
	outControls->count = 0;

	mama_object_id_list owned;
	if (!mama_get_owned_objects(processObjectID, &owned)) {
		return 0;
	}

	AudioObjectID* matches = (AudioObjectID*)calloc(owned.count, sizeof(AudioObjectID));
	if (matches == NULL) {
		mama_free_object_id_list(&owned);
		return 0;
	}

	uint32_t n = 0;
	for (uint32_t i = 0; i < owned.count; i++) {
		AudioObjectID candidate = owned.items[i];
		if (candidate == 0) {
			continue;
		}
		if (!mama_class_is_or_inherits(candidate, targetClass)) {
			continue;
		}
		AudioObjectPropertyScope scope = kAudioObjectPropertyScopeGlobal;
		if (mama_get_control_scope(candidate, &scope)) {
			if (scope != kAudioObjectPropertyScopeGlobal && scope != kAudioObjectPropertyScopeOutput) {
				continue;
			}
		}
		matches[n++] = candidate;
	}

	mama_free_object_id_list(&owned);
	if (n == 0) {
		free(matches);
		return 0;
	}

	outControls->items = matches;
	outControls->count = n;
	return 1;
}

static int mama_get_first_scalar_property(AudioObjectID objectID, AudioObjectPropertySelector selector, Float32* outValue) {
	if (outValue == NULL) {
		return 0;
	}
	mama_property_address_list addresses;
	mama_collect_property_addresses(objectID, selector, &addresses);
	for (uint32_t i = 0; i < addresses.count; i++) {
		Float32 value = 0.0f;
		UInt32 size = (UInt32)sizeof(value);
		OSStatus st = AudioObjectGetPropertyData(objectID, &addresses.items[i], 0, NULL, &size, &value);
		if (st == noErr) {
			*outValue = value;
			return 1;
		}
	}
	return 0;
}

static int mama_set_any_scalar_property(AudioObjectID objectID, AudioObjectPropertySelector selector, Float32 value) {
	mama_property_address_list addresses;
	mama_collect_property_addresses(objectID, selector, &addresses);
	UInt32 size = (UInt32)sizeof(value);
	int wroteAny = 0;
	for (uint32_t i = 0; i < addresses.count; i++) {
		OSStatus st = AudioObjectSetPropertyData(objectID, &addresses.items[i], 0, NULL, size, &value);
		if (st == noErr) {
			wroteAny = 1;
		}
	}
	return wroteAny;
}

static int mama_get_first_u32_property(AudioObjectID objectID, AudioObjectPropertySelector selector, UInt32* outValue) {
	if (outValue == NULL) {
		return 0;
	}
	mama_property_address_list addresses;
	mama_collect_property_addresses(objectID, selector, &addresses);
	for (uint32_t i = 0; i < addresses.count; i++) {
		UInt32 value = 0;
		UInt32 size = (UInt32)sizeof(value);
		OSStatus st = AudioObjectGetPropertyData(objectID, &addresses.items[i], 0, NULL, &size, &value);
		if (st == noErr) {
			*outValue = value;
			return 1;
		}
	}
	return 0;
}

static int mama_set_any_u32_property(AudioObjectID objectID, AudioObjectPropertySelector selector, UInt32 value) {
	mama_property_address_list addresses;
	mama_collect_property_addresses(objectID, selector, &addresses);
	UInt32 size = (UInt32)sizeof(value);
	int wroteAny = 0;
	for (uint32_t i = 0; i < addresses.count; i++) {
		OSStatus st = AudioObjectSetPropertyData(objectID, &addresses.items[i], 0, NULL, size, &value);
		if (st == noErr) {
			wroteAny = 1;
		}
	}
	return wroteAny;
}

int mama_coreaudio_get_process_volume(uint32_t processObjectID, float* outVolume, int* outSupported, int* outStatus) {
	if (outSupported != NULL) {
		*outSupported = 0;
	}
	if (outStatus != NULL) {
		*outStatus = 0;
	}
	if (outVolume == NULL) {
		return 1;
	}

	AudioObjectID objectID = (AudioObjectID)processObjectID;
	mama_property_address_list addresses;
	mama_collect_property_addresses(objectID, kAudioDevicePropertyVolumeScalar, &addresses);
	if (addresses.count == 0) {
		mama_collect_property_addresses(objectID, kMAMAAudioPropertyVirtualMainVolume, &addresses);
	}

	double sum = 0.0;
	int successCount = 0;
	OSStatus lastErr = noErr;
	if (addresses.count > 0) {
		if (outSupported != NULL) {
			*outSupported = 1;
		}
		for (uint32_t i = 0; i < addresses.count; i++) {
			Float32 value = 0.0f;
			UInt32 size = (UInt32)sizeof(value);
			OSStatus st = AudioObjectGetPropertyData(objectID, &addresses.items[i], 0, NULL, &size, &value);
			if (st != noErr) {
				lastErr = st;
				continue;
			}
			if (value < 0.0f) {
				value = 0.0f;
			}
			if (value > 1.0f) {
				value = 1.0f;
			}
			sum += value;
			successCount++;
		}
	}

	if (successCount == 0) {
		mama_object_id_list controls;
		if (!mama_collect_process_controls(objectID, kAudioVolumeControlClassID, &controls)) {
			mama_collect_process_controls(objectID, kAudioLevelControlClassID, &controls);
		}
		if (controls.count > 0) {
			if (outSupported != NULL) {
				*outSupported = 1;
			}
			for (uint32_t i = 0; i < controls.count; i++) {
				Float32 value = 0.0f;
				if (!mama_get_first_scalar_property(controls.items[i], kAudioLevelControlPropertyScalarValue, &value)) {
					continue;
				}
				if (value < 0.0f) {
					value = 0.0f;
				}
				if (value > 1.0f) {
					value = 1.0f;
				}
				sum += value;
				successCount++;
			}
			mama_free_object_id_list(&controls);
		}
	}

	if (successCount == 0) {
		if (outSupported != NULL && *outSupported != 0) {
			if (outStatus != NULL) {
				*outStatus = (int)lastErr;
			}
			return 1;
		}
		return 0;
	}

	Float32 avg = (Float32)(sum / (double)successCount);
	if (avg < 0.0f) {
		avg = 0.0f;
	}
	if (avg > 1.0f) {
		avg = 1.0f;
	}
	*outVolume = avg;
	return 0;
}

int mama_coreaudio_set_process_volume(uint32_t processObjectID, float volume, int* outSupported, int* outStatus) {
	if (outSupported != NULL) {
		*outSupported = 0;
	}
	if (outStatus != NULL) {
		*outStatus = 0;
	}
	if (volume < 0.0f) {
		volume = 0.0f;
	}
	if (volume > 1.0f) {
		volume = 1.0f;
	}

	AudioObjectID objectID = (AudioObjectID)processObjectID;
	mama_property_address_list addresses;
	mama_collect_property_addresses(objectID, kAudioDevicePropertyVolumeScalar, &addresses);
	if (addresses.count == 0) {
		mama_collect_property_addresses(objectID, kMAMAAudioPropertyVirtualMainVolume, &addresses);
	}
	if (addresses.count == 0) {
		return 0;
	}
	if (outSupported != NULL) {
		*outSupported = 1;
	}

	Float32 value = volume;
	UInt32 size = (UInt32)sizeof(value);
	int wroteAny = 0;
	OSStatus lastErr = noErr;
	for (uint32_t i = 0; i < addresses.count; i++) {
		OSStatus st = AudioObjectSetPropertyData(objectID, &addresses.items[i], 0, NULL, size, &value);
		if (st == noErr) {
			wroteAny = 1;
			continue;
		}
		lastErr = st;
	}
	if (!wroteAny) {
		if (outStatus != NULL) {
			*outStatus = (int)lastErr;
		}
		return 1;
	}
	return 0;
}

int mama_coreaudio_get_process_mute(uint32_t processObjectID, uint32_t* outMuted, int* outSupported, int* outStatus) {
	if (outSupported != NULL) {
		*outSupported = 0;
	}
	if (outStatus != NULL) {
		*outStatus = 0;
	}
	if (outMuted == NULL) {
		return 1;
	}

	AudioObjectID objectID = (AudioObjectID)processObjectID;
	mama_property_address_list addresses;
	mama_collect_property_addresses(objectID, kAudioDevicePropertyMute, &addresses);
	if (addresses.count > 0) {
		if (outSupported != NULL) {
			*outSupported = 1;
		}

		UInt32 value = 0;
		UInt32 size = (UInt32)sizeof(value);
		OSStatus st = AudioObjectGetPropertyData(objectID, &addresses.items[0], 0, NULL, &size, &value);
		if (st != noErr) {
			if (outStatus != NULL) {
				*outStatus = (int)st;
			}
			return 1;
		}
		*outMuted = value ? 1 : 0;
		return 0;
	}

	mama_collect_property_addresses(objectID, kMAMAAudioPropertyProcessIsAudible, &addresses);
	if (addresses.count == 0) {
		return 0;
	}
	if (outSupported != NULL) {
		*outSupported = 1;
	}

	UInt32 audible = 1;
	UInt32 size = (UInt32)sizeof(audible);
	OSStatus st = AudioObjectGetPropertyData(objectID, &addresses.items[0], 0, NULL, &size, &audible);
	if (st != noErr) {
		if (outStatus != NULL) {
			*outStatus = (int)st;
		}
		return 1;
	}
	*outMuted = audible ? 0 : 1;
	return 0;
}

int mama_coreaudio_set_process_mute(uint32_t processObjectID, uint32_t muted, int* outSupported, int* outStatus) {
	if (outSupported != NULL) {
		*outSupported = 0;
	}
	if (outStatus != NULL) {
		*outStatus = 0;
	}

	AudioObjectID objectID = (AudioObjectID)processObjectID;
	mama_property_address_list addresses;
	mama_collect_property_addresses(objectID, kAudioDevicePropertyMute, &addresses);
	if (addresses.count > 0) {
		if (outSupported != NULL) {
			*outSupported = 1;
		}

		UInt32 value = muted ? 1 : 0;
		UInt32 size = (UInt32)sizeof(value);
		int wroteAny = 0;
		OSStatus lastErr = noErr;
		for (uint32_t i = 0; i < addresses.count; i++) {
			OSStatus st = AudioObjectSetPropertyData(objectID, &addresses.items[i], 0, NULL, size, &value);
			if (st == noErr) {
				wroteAny = 1;
				continue;
			}
			lastErr = st;
		}
		if (!wroteAny) {
			if (outStatus != NULL) {
				*outStatus = (int)lastErr;
			}
			return 1;
		}
		return 0;
	}

	mama_collect_property_addresses(objectID, kMAMAAudioPropertyProcessIsAudible, &addresses);
	if (addresses.count == 0) {
		return 0;
	}
	if (outSupported != NULL) {
		*outSupported = 1;
	}

	UInt32 audible = muted ? 0 : 1;
	UInt32 size = (UInt32)sizeof(audible);
	OSStatus st = AudioObjectSetPropertyData(objectID, &addresses.items[0], 0, NULL, size, &audible);
	if (st != noErr) {
		if (outStatus != NULL) {
			*outStatus = (int)st;
		}
		return 1;
	}
	return 0;
}
*/
import "C"

import (
	"fmt"
	"math"
	"unsafe"
)

type coreAudioProcessInfo struct {
	ObjectID       uint32
	PID            int
	RunningOutput  bool
	BundleID       string
	ExecutablePath string
}

type coreAudioBridge interface {
	ListProcesses() ([]coreAudioProcessInfo, error)
	GetProcessVolume(processObjectID uint32) (volume int, supported bool, err error)
	SetProcessVolume(processObjectID uint32, volume int) (supported bool, err error)
	GetProcessMute(processObjectID uint32) (muted bool, supported bool, err error)
	SetProcessMute(processObjectID uint32, muted bool) (supported bool, err error)
}

type defaultCoreAudioBridge struct{}

func newCoreAudioBridge() coreAudioBridge {
	return defaultCoreAudioBridge{}
}

func (defaultCoreAudioBridge) ListProcesses() ([]coreAudioProcessInfo, error) {
	var cProcesses *C.mama_coreaudio_process_info
	var cCount C.uint32_t
	var cErr *C.char

	result := C.mama_coreaudio_list_processes(&cProcesses, &cCount, &cErr)
	if cErr != nil {
		defer C.mama_coreaudio_free_error(cErr)
	}
	if result != 0 {
		if cErr != nil {
			return nil, fmt.Errorf("coreaudio list processes: %s", C.GoString(cErr))
		}
		return nil, fmt.Errorf("coreaudio list processes failed")
	}
	if cProcesses == nil || cCount == 0 {
		return nil, nil
	}
	defer C.mama_coreaudio_free_processes(cProcesses, cCount)

	count := int(cCount)
	rows := unsafe.Slice(cProcesses, count)
	processes := make([]coreAudioProcessInfo, 0, count)
	for i := 0; i < count; i++ {
		row := rows[i]
		proc := coreAudioProcessInfo{
			ObjectID:      uint32(row.object_id),
			PID:           int(row.pid),
			RunningOutput: row.running_output != 0,
		}
		if row.bundle_id != nil {
			proc.BundleID = C.GoString(row.bundle_id)
		}
		if row.executable_path != nil {
			proc.ExecutablePath = C.GoString(row.executable_path)
		}
		processes = append(processes, proc)
	}
	return processes, nil
}

func (defaultCoreAudioBridge) GetProcessVolume(processObjectID uint32) (int, bool, error) {
	var cVolume C.float
	var cSupported C.int
	var cStatus C.int
	result := C.mama_coreaudio_get_process_volume(C.uint32_t(processObjectID), &cVolume, &cSupported, &cStatus)
	if result != 0 {
		return 0, cSupported != 0, fmt.Errorf("coreaudio get process volume (id=%d): osstatus=%d", processObjectID, int(cStatus))
	}
	if cSupported == 0 {
		return 0, false, nil
	}
	v := int(math.Round(float64(cVolume) * 100.0))
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	return v, true, nil
}

func (defaultCoreAudioBridge) SetProcessVolume(processObjectID uint32, volume int) (bool, error) {
	if volume < 0 {
		volume = 0
	}
	if volume > 100 {
		volume = 100
	}
	var cSupported C.int
	var cStatus C.int
	result := C.mama_coreaudio_set_process_volume(C.uint32_t(processObjectID), C.float(float32(volume)/100.0), &cSupported, &cStatus)
	if result != 0 {
		return cSupported != 0, fmt.Errorf("coreaudio set process volume (id=%d): osstatus=%d", processObjectID, int(cStatus))
	}
	return cSupported != 0, nil
}

func (defaultCoreAudioBridge) GetProcessMute(processObjectID uint32) (bool, bool, error) {
	var cMuted C.uint32_t
	var cSupported C.int
	var cStatus C.int
	result := C.mama_coreaudio_get_process_mute(C.uint32_t(processObjectID), &cMuted, &cSupported, &cStatus)
	if result != 0 {
		return false, cSupported != 0, fmt.Errorf("coreaudio get process mute (id=%d): osstatus=%d", processObjectID, int(cStatus))
	}
	if cSupported == 0 {
		return false, false, nil
	}
	return cMuted != 0, true, nil
}

func (defaultCoreAudioBridge) SetProcessMute(processObjectID uint32, muted bool) (bool, error) {
	value := C.uint32_t(0)
	if muted {
		value = 1
	}
	var cSupported C.int
	var cStatus C.int
	result := C.mama_coreaudio_set_process_mute(C.uint32_t(processObjectID), value, &cSupported, &cStatus)
	if result != 0 {
		return cSupported != 0, fmt.Errorf("coreaudio set process mute (id=%d): osstatus=%d", processObjectID, int(cStatus))
	}
	return cSupported != 0, nil
}
