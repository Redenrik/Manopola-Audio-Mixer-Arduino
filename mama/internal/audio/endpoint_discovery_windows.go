//go:build windows

package audio

import (
	"fmt"
	"strings"

	"github.com/moutend/go-wca"
	"mama/internal/config"
)

type windowsEndpointInfo struct {
	id        string
	name      string
	dataFlow  uint32
	isDefault bool
}

func listWindowsEndpointTargets() ([]DiscoveredTarget, error) {
	renderEndpoints, renderErr := listEndpointInfos(wca.ERender)
	captureEndpoints, captureErr := listEndpointInfos(wca.ECapture)
	if renderErr != nil && captureErr != nil {
		return nil, renderErr
	}

	targets := make([]DiscoveredTarget, 0, len(renderEndpoints)+(len(captureEndpoints)*2))
	for _, endpoint := range renderEndpoints {
		targets = append(targets, DiscoveredTarget{
			ID:       "endpoint:render:" + endpoint.id,
			Type:     config.TargetMasterOut,
			Name:     endpoint.name,
			Selector: "id:" + endpoint.id,
			Aliases:  []string{endpoint.name},
		})
	}
	for _, endpoint := range captureEndpoints {
		selector := "id:" + endpoint.id
		targets = append(targets, DiscoveredTarget{
			ID:       "endpoint:mic:" + endpoint.id,
			Type:     config.TargetMicIn,
			Name:     endpoint.name,
			Selector: selector,
			Aliases:  []string{endpoint.name},
		})
		targets = append(targets, DiscoveredTarget{
			ID:       "endpoint:line:" + endpoint.id,
			Type:     config.TargetLineIn,
			Name:     endpoint.name,
			Selector: selector,
			Aliases:  []string{endpoint.name},
		})
	}
	return targets, nil
}

func listEndpointInfos(dataFlow uint32) ([]windowsEndpointInfo, error) {
	var endpoints []windowsEndpointInfo
	err := withCOMApartment(func() error {
		var mmde *wca.IMMDeviceEnumerator
		if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
			return err
		}
		defer mmde.Release()

		infos, err := listEndpointInfosWithEnumerator(mmde, dataFlow)
		if err != nil {
			return err
		}
		endpoints = infos
		return nil
	})
	return endpoints, err
}

func listEndpointInfosWithEnumerator(mmde *wca.IMMDeviceEnumerator, dataFlow uint32) ([]windowsEndpointInfo, error) {
	var collection *wca.IMMDeviceCollection
	if err := mmde.EnumAudioEndpoints(dataFlow, wca.DEVICE_STATE_ACTIVE, &collection); err != nil {
		return nil, err
	}
	defer collection.Release()

	defaultIDs := defaultEndpointIDSet(mmde, dataFlow)

	var count uint32
	if err := collection.GetCount(&count); err != nil {
		return nil, err
	}

	endpoints := make([]windowsEndpointInfo, 0, count)
	for i := uint32(0); i < count; i++ {
		var device *wca.IMMDevice
		if err := collection.Item(i, &device); err != nil {
			continue
		}

		info, err := endpointInfoFromDevice(device, dataFlow)
		device.Release()
		if err != nil {
			continue
		}
		_, info.isDefault = defaultIDs[strings.ToLower(info.id)]
		endpoints = append(endpoints, info)
	}
	return endpoints, nil
}

func defaultEndpointIDSet(mmde *wca.IMMDeviceEnumerator, dataFlow uint32) map[string]struct{} {
	ids := make(map[string]struct{}, 3)
	roles := []uint32{wca.EConsole, wca.EMultimedia, wca.ECommunications}
	for _, role := range roles {
		var device *wca.IMMDevice
		if err := mmde.GetDefaultAudioEndpoint(dataFlow, role, &device); err != nil {
			continue
		}
		id, err := endpointID(device)
		device.Release()
		if err != nil {
			continue
		}
		ids[strings.ToLower(id)] = struct{}{}
	}
	return ids
}

func endpointInfoFromDevice(device *wca.IMMDevice, dataFlow uint32) (windowsEndpointInfo, error) {
	id, err := endpointID(device)
	if err != nil {
		return windowsEndpointInfo{}, err
	}
	name := endpointFriendlyName(device)
	if name == "" {
		name = id
	}
	return windowsEndpointInfo{
		id:       id,
		name:     name,
		dataFlow: dataFlow,
	}, nil
}

func endpointID(device *wca.IMMDevice) (string, error) {
	id, err := getIMMDeviceID(device)
	if err != nil {
		return "", err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("empty endpoint id")
	}
	return id, nil
}

func endpointFriendlyName(device *wca.IMMDevice) string {
	var store *wca.IPropertyStore
	if err := device.OpenPropertyStore(wca.STGM_READ, &store); err != nil {
		return ""
	}
	defer store.Release()

	var value wca.PROPVARIANT
	if err := store.GetValue(&wca.PKEY_Device_FriendlyName, &value); err != nil {
		return ""
	}
	return strings.TrimSpace(value.String())
}

func selectEndpointDevice(mmde *wca.IMMDeviceEnumerator, dataFlow uint32, selectorToken string) (*wca.IMMDevice, error) {
	selectorToken = strings.TrimSpace(selectorToken)
	if selectorToken == "" {
		return selectDefaultEndpointDevice(mmde, dataFlow)
	}

	selector, matchByID := parseEndpointSelectorToken(selectorToken)

	var collection *wca.IMMDeviceCollection
	if err := mmde.EnumAudioEndpoints(dataFlow, wca.DEVICE_STATE_ACTIVE, &collection); err != nil {
		return nil, err
	}
	defer collection.Release()

	var count uint32
	if err := collection.GetCount(&count); err != nil {
		return nil, err
	}

	for i := uint32(0); i < count; i++ {
		var device *wca.IMMDevice
		if err := collection.Item(i, &device); err != nil {
			continue
		}
		info, err := endpointInfoFromDevice(device, dataFlow)
		if err != nil {
			device.Release()
			continue
		}
		if endpointMatchesSelector(info, selector, matchByID) {
			return device, nil
		}
		device.Release()
	}

	return nil, fmt.Errorf("%w: no %s endpoint matched selector %q", ErrTargetUnavailable, endpointDataFlowLabel(dataFlow), selectorToken)
}

func selectDefaultEndpointDevice(mmde *wca.IMMDeviceEnumerator, dataFlow uint32) (*wca.IMMDevice, error) {
	roles := []uint32{wca.EConsole, wca.EMultimedia, wca.ECommunications}
	for _, role := range roles {
		var device *wca.IMMDevice
		if err := mmde.GetDefaultAudioEndpoint(dataFlow, role, &device); err == nil && device != nil {
			return device, nil
		}
	}
	return nil, fmt.Errorf("default %s endpoint unavailable", endpointDataFlowLabel(dataFlow))
}

func parseEndpointSelectorToken(token string) (config.Selector, bool) {
	token = strings.TrimSpace(token)
	if left, right, ok := strings.Cut(token, ":"); ok {
		if strings.EqualFold(strings.TrimSpace(left), "id") {
			value := strings.TrimSpace(right)
			return config.Selector{Kind: config.SelectorExact, Value: value}, true
		}
	}

	selector := parseSelectorToken(token)
	if selector.Kind == config.SelectorExe {
		selector.Kind = config.SelectorExact
	}
	return selector, false
}

func endpointMatchesSelector(endpoint windowsEndpointInfo, selector config.Selector, matchByID bool) bool {
	if matchByID {
		return strings.EqualFold(strings.TrimSpace(endpoint.id), strings.TrimSpace(selector.Value))
	}
	return selectorMatchesAnyValue(selector, normalizeMatchValues(endpoint.name, endpoint.id))
}

func endpointDataFlowLabel(dataFlow uint32) string {
	switch dataFlow {
	case wca.ERender:
		return "output"
	case wca.ECapture:
		return "input"
	default:
		return "audio"
	}
}
