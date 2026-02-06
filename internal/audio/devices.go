package audio

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ScreenCaptureKit -framework CoreMedia -framework AVFoundation -framework Foundation
#include <stdlib.h>
#include "capture_darwin.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// SystemAudioCapturer captures system audio via ScreenCaptureKit (in-process via cgo).
type SystemAudioCapturer struct{}

func NewSystemAudioCapturer() (*SystemAudioCapturer, error) {
	return &SystemAudioCapturer{}, nil
}

// StartCapture begins capturing system audio, streaming 16kHz mono WAV to outputPath.
func (c *SystemAudioCapturer) StartCapture(outputPath string) error {
	cPath := C.CString(outputPath)
	defer C.free(unsafe.Pointer(cPath))

	if C.capture_start(cPath) != 0 {
		return fmt.Errorf("failed to start system audio capture — check screen recording permission in System Settings > Privacy & Security")
	}
	return nil
}

// StopCapture stops capturing and finalizes the WAV file.
func (c *SystemAudioCapturer) StopCapture() {
	C.capture_stop()
}
