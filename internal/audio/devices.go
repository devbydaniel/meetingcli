package audio

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

//go:embed devices_helper.swift
var helperFS embed.FS

// DeviceInfo represents an audio device.
type DeviceInfo struct {
	ID       uint32 `json:"id"`
	UID      string `json:"uid"`
	Name     string `json:"name"`
	IsInput  bool   `json:"is_input"`
	IsOutput bool   `json:"is_output"`
}

// CreatedDevices holds IDs of programmatically created audio devices.
type CreatedDevices struct {
	MultiOutputID     uint32 `json:"multi_output_id"`
	AggregateID       uint32 `json:"aggregate_id"`
	AggregateUID      string `json:"aggregate_uid"`
	OriginalOutputUID string `json:"original_output_uid"`
	MicUID            string `json:"mic_uid"`
}

// DeviceManager manages CoreAudio devices via the Swift helper.
type DeviceManager struct {
	helperPath string
}

// NewDeviceManager creates a DeviceManager, extracting the Swift helper to a temp location.
func NewDeviceManager() (*DeviceManager, error) {
	content, err := helperFS.ReadFile("devices_helper.swift")
	if err != nil {
		return nil, fmt.Errorf("reading embedded helper: %w", err)
	}

	tmpDir := filepath.Join(os.TempDir(), "meetingcli")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return nil, err
	}

	helperPath := filepath.Join(tmpDir, "devices_helper.swift")
	if err := os.WriteFile(helperPath, content, 0o644); err != nil {
		return nil, err
	}

	return &DeviceManager{helperPath: helperPath}, nil
}

func (dm *DeviceManager) run(args ...string) (map[string]any, error) {
	cmdArgs := append([]string{dm.helperPath}, args...)
	cmd := exec.Command("swift", cmdArgs...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("swift helper failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("running swift helper: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parsing helper output: %w (raw: %s)", err, string(out))
	}

	if errMsg, ok := result["error"].(string); ok {
		return nil, fmt.Errorf("%s", errMsg)
	}

	return result, nil
}

func getString(m map[string]any, key string) (string, error) {
	v, ok := m[key].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid field %q in helper response", key)
	}
	return v, nil
}

func getFloat64(m map[string]any, key string) (float64, error) {
	v, ok := m[key].(float64)
	if !ok {
		return 0, fmt.Errorf("missing or invalid field %q in helper response", key)
	}
	return v, nil
}

// FindBlackhole locates the BlackHole 2ch device.
func (dm *DeviceManager) FindBlackhole() (*DeviceInfo, error) {
	result, err := dm.run("find-blackhole")
	if err != nil {
		return nil, err
	}

	uid, err := getString(result, "uid")
	if err != nil {
		return nil, err
	}
	name, err := getString(result, "name")
	if err != nil {
		return nil, err
	}

	return &DeviceInfo{UID: uid, Name: name}, nil
}

// GetCurrentOutput returns the current default output device UID.
func (dm *DeviceManager) GetCurrentOutput() (string, error) {
	result, err := dm.run("current-output")
	if err != nil {
		return "", err
	}
	return getString(result, "uid")
}

// CreateDevices creates the multi-output and aggregate devices for recording.
func (dm *DeviceManager) CreateDevices(blackholeUID string) (*CreatedDevices, error) {
	result, err := dm.run("create-devices", blackholeUID)
	if err != nil {
		return nil, err
	}

	multiOutputID, err := getFloat64(result, "multi_output_id")
	if err != nil {
		return nil, err
	}
	aggregateID, err := getFloat64(result, "aggregate_id")
	if err != nil {
		return nil, err
	}
	aggregateUID, err := getString(result, "aggregate_uid")
	if err != nil {
		return nil, err
	}
	originalOutputUID, err := getString(result, "original_output_uid")
	if err != nil {
		return nil, err
	}
	micUID, err := getString(result, "mic_uid")
	if err != nil {
		return nil, err
	}

	return &CreatedDevices{
		MultiOutputID:     uint32(multiOutputID),
		AggregateID:       uint32(aggregateID),
		AggregateUID:      aggregateUID,
		OriginalOutputUID: originalOutputUID,
		MicUID:            micUID,
	}, nil
}

// DestroyDevices removes the programmatically created audio devices.
func (dm *DeviceManager) DestroyDevices(multiOutputID, aggregateID uint32) error {
	_, err := dm.run("destroy-devices",
		strconv.FormatUint(uint64(multiOutputID), 10),
		strconv.FormatUint(uint64(aggregateID), 10),
	)
	return err
}

// SwitchOutput sets the default output device by UID.
func (dm *DeviceManager) SwitchOutput(uid string) error {
	_, err := dm.run("switch-output", uid)
	return err
}

// ListDevices returns all audio devices.
func (dm *DeviceManager) ListDevices() ([]DeviceInfo, error) {
	result, err := dm.run("list-devices")
	if err != nil {
		return nil, err
	}

	devicesRaw, ok := result["devices"].([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected devices format")
	}

	var devices []DeviceInfo
	for _, d := range devicesRaw {
		m, ok := d.(map[string]any)
		if !ok {
			continue
		}

		id, err := getFloat64(m, "id")
		if err != nil {
			continue
		}
		uid, err := getString(m, "uid")
		if err != nil {
			continue
		}
		name, err := getString(m, "name")
		if err != nil {
			continue
		}

		isInput, _ := m["is_input"].(bool)
		isOutput, _ := m["is_output"].(bool)

		devices = append(devices, DeviceInfo{
			ID:       uint32(id),
			UID:      uid,
			Name:     name,
			IsInput:  isInput,
			IsOutput: isOutput,
		})
	}
	return devices, nil
}
