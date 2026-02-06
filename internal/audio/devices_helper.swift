// Audio device helper for meetingcli.
// Communicates with Go via JSON on stdout.
// Usage: swift devices_helper.swift <command> [args...]
//
// Commands:
//   list-devices                    — list all audio devices
//   current-output                  — get current default output device UID
//   create-devices <blackhole_uid>  — create multi-output + aggregate devices, print JSON with IDs
//   destroy-devices <multi_id> <agg_id> — destroy created devices
//   switch-output <device_uid>      — set default output device by UID
//   find-blackhole                  — find BlackHole 2ch device UID

import CoreAudio
import Foundation

// MARK: - Helpers

func getDeviceUID(_ deviceID: AudioDeviceID) -> String? {
    var size = UInt32(MemoryLayout<CFString>.size)
    var address = AudioObjectPropertyAddress(
        mSelector: kAudioDevicePropertyDeviceUID,
        mScope: kAudioObjectPropertyScopeGlobal,
        mElement: kAudioObjectPropertyElementMain
    )
    var uid: Unmanaged<CFString>?
    let status = withUnsafeMutablePointer(to: &uid) { ptr in
        AudioObjectGetPropertyData(deviceID, &address, 0, nil, &size, ptr)
    }
    guard status == noErr, let cfStr = uid?.takeRetainedValue() else { return nil }
    return cfStr as String
}

func getDeviceName(_ deviceID: AudioDeviceID) -> String? {
    var size = UInt32(MemoryLayout<CFString>.size)
    var address = AudioObjectPropertyAddress(
        mSelector: kAudioObjectPropertyName,
        mScope: kAudioObjectPropertyScopeGlobal,
        mElement: kAudioObjectPropertyElementMain
    )
    var name: Unmanaged<CFString>?
    let status = withUnsafeMutablePointer(to: &name) { ptr in
        AudioObjectGetPropertyData(deviceID, &address, 0, nil, &size, ptr)
    }
    guard status == noErr, let cfStr = name?.takeRetainedValue() else { return nil }
    return cfStr as String
}

func hasStreams(_ deviceID: AudioDeviceID, scope: AudioObjectPropertyScope) -> Bool {
    var address = AudioObjectPropertyAddress(
        mSelector: kAudioDevicePropertyStreams,
        mScope: scope,
        mElement: kAudioObjectPropertyElementMain
    )
    var size: UInt32 = 0
    let status = AudioObjectGetPropertyDataSize(deviceID, &address, 0, nil, &size)
    return status == noErr && size > 0
}

func getAllDevices() -> [AudioDeviceID] {
    var size: UInt32 = 0
    var address = AudioObjectPropertyAddress(
        mSelector: kAudioHardwarePropertyDevices,
        mScope: kAudioObjectPropertyScopeGlobal,
        mElement: kAudioObjectPropertyElementMain
    )
    AudioObjectGetPropertyDataSize(AudioObjectID(kAudioObjectSystemObject), &address, 0, nil, &size)
    let count = Int(size) / MemoryLayout<AudioDeviceID>.size
    var devices = [AudioDeviceID](repeating: 0, count: count)
    AudioObjectGetPropertyData(AudioObjectID(kAudioObjectSystemObject), &address, 0, nil, &size, &devices)
    return devices
}

func getCurrentOutputDeviceID() -> AudioDeviceID? {
    var size = UInt32(MemoryLayout<AudioDeviceID>.size)
    var address = AudioObjectPropertyAddress(
        mSelector: kAudioHardwarePropertyDefaultOutputDevice,
        mScope: kAudioObjectPropertyScopeGlobal,
        mElement: kAudioObjectPropertyElementMain
    )
    var deviceID: AudioDeviceID = 0
    let status = AudioObjectGetPropertyData(AudioObjectID(kAudioObjectSystemObject), &address, 0, nil, &size, &deviceID)
    guard status == noErr else { return nil }
    return deviceID
}

func setDefaultOutputDevice(uid: String) -> Bool {
    let devices = getAllDevices()
    for device in devices {
        if getDeviceUID(device) == uid {
            var address = AudioObjectPropertyAddress(
                mSelector: kAudioHardwarePropertyDefaultOutputDevice,
                mScope: kAudioObjectPropertyScopeGlobal,
                mElement: kAudioObjectPropertyElementMain
            )
            var deviceID = device
            let status = AudioObjectSetPropertyData(
                AudioObjectID(kAudioObjectSystemObject), &address, 0, nil,
                UInt32(MemoryLayout<AudioDeviceID>.size), &deviceID
            )
            return status == noErr
        }
    }
    return false
}

func findDeviceByUID(_ uid: String) -> AudioDeviceID? {
    for device in getAllDevices() {
        if getDeviceUID(device) == uid {
            return device
        }
    }
    return nil
}

func jsonOutput(_ dict: [String: Any]) {
    if let data = try? JSONSerialization.data(withJSONObject: dict, options: []),
       let str = String(data: data, encoding: .utf8) {
        print(str)
    }
}

func errorOutput(_ msg: String) {
    jsonOutput(["error": msg])
}

// MARK: - Commands

func listDevices() {
    let devices = getAllDevices()
    var result: [[String: Any]] = []
    for device in devices {
        guard let uid = getDeviceUID(device), let name = getDeviceName(device) else { continue }
        let isInput = hasStreams(device, scope: kAudioObjectPropertyScopeInput)
        let isOutput = hasStreams(device, scope: kAudioObjectPropertyScopeOutput)
        result.append([
            "id": device,
            "uid": uid,
            "name": name,
            "is_input": isInput,
            "is_output": isOutput
        ])
    }
    jsonOutput(["devices": result])
}

func currentOutput() {
    guard let deviceID = getCurrentOutputDeviceID(),
          let uid = getDeviceUID(deviceID),
          let name = getDeviceName(deviceID) else {
        errorOutput("could not get current output device")
        return
    }
    jsonOutput(["uid": uid, "name": name, "id": deviceID])
}

func findBlackhole() {
    let devices = getAllDevices()
    for device in devices {
        guard let uid = getDeviceUID(device), let name = getDeviceName(device) else { continue }
        // BlackHole 2ch typically has "BlackHole" in the name
        if name.contains("BlackHole") && name.contains("2ch") {
            jsonOutput(["uid": uid, "name": name, "id": device])
            return
        }
    }
    errorOutput("BlackHole 2ch not found. Install with: brew install blackhole-2ch")
}

func findBuiltInMic() -> String? {
    let devices = getAllDevices()
    for device in devices {
        guard let uid = getDeviceUID(device) else { continue }
        if uid == "BuiltInMicrophoneDevice" {
            return uid
        }
    }
    // Fallback: find any input device
    for device in devices {
        guard let uid = getDeviceUID(device) else { continue }
        if hasStreams(device, scope: kAudioObjectPropertyScopeInput) {
            return uid
        }
    }
    return nil
}

func destroyStaleDevices() {
    // Clean up any leftover devices from a previous crashed session
    let staleUIDs = ["com.meetingcli.aggregate", "com.meetingcli.multioutput"]
    for device in getAllDevices() {
        if let uid = getDeviceUID(device), staleUIDs.contains(uid) {
            AudioHardwareDestroyAggregateDevice(device)
        }
    }
}

func createDevices(blackholeUID: String) {
    // Clean up stale devices from previous crashed sessions
    destroyStaleDevices()

    // Get current output device to include in multi-output
    guard let currentOutputID = getCurrentOutputDeviceID(),
          let currentOutputUID = getDeviceUID(currentOutputID) else {
        errorOutput("could not get current output device")
        return
    }

    // Find a mic
    guard let micUID = findBuiltInMic() else {
        errorOutput("could not find microphone device")
        return
    }

    // 1. Create Multi-Output Device (current speakers + BlackHole)
    let multiOutputSubs: [[String: Any]] = [
        [kAudioSubDeviceUIDKey as String: currentOutputUID],
        [kAudioSubDeviceUIDKey as String: blackholeUID]
    ]
    let multiOutputDesc: [String: Any] = [
        kAudioAggregateDeviceNameKey as String: "MeetingCLI-MultiOutput",
        kAudioAggregateDeviceUIDKey as String: "com.meetingcli.multioutput",
        kAudioAggregateDeviceIsPrivateKey as String: 1,
        kAudioAggregateDeviceIsStackedKey as String: 1,
        kAudioAggregateDeviceSubDeviceListKey as String: multiOutputSubs
    ]

    var multiOutputID: AudioDeviceID = 0
    var status = AudioHardwareCreateAggregateDevice(multiOutputDesc as CFDictionary, &multiOutputID)
    if status != noErr {
        errorOutput("failed to create multi-output device: OSStatus \(status)")
        return
    }

    // 2. Create Aggregate Device (BlackHole + mic for recording)
    let aggregateSubs: [[String: Any]] = [
        [kAudioSubDeviceUIDKey as String: micUID],
        [kAudioSubDeviceUIDKey as String: blackholeUID]
    ]
    let aggregateDesc: [String: Any] = [
        kAudioAggregateDeviceNameKey as String: "MeetingCLI-Aggregate",
        kAudioAggregateDeviceUIDKey as String: "com.meetingcli.aggregate",
        kAudioAggregateDeviceSubDeviceListKey as String: aggregateSubs
    ]

    var aggregateID: AudioDeviceID = 0
    status = AudioHardwareCreateAggregateDevice(aggregateDesc as CFDictionary, &aggregateID)
    if status != noErr {
        // Clean up multi-output
        AudioHardwareDestroyAggregateDevice(multiOutputID)
        errorOutput("failed to create aggregate device: OSStatus \(status)")
        return
    }

    // 3. Switch output to multi-output device
    var outputAddr = AudioObjectPropertyAddress(
        mSelector: kAudioHardwarePropertyDefaultOutputDevice,
        mScope: kAudioObjectPropertyScopeGlobal,
        mElement: kAudioObjectPropertyElementMain
    )
    var newOutputID = multiOutputID
    status = AudioObjectSetPropertyData(
        AudioObjectID(kAudioObjectSystemObject), &outputAddr, 0, nil,
        UInt32(MemoryLayout<AudioDeviceID>.size), &newOutputID
    )
    if status != noErr {
        AudioHardwareDestroyAggregateDevice(aggregateID)
        AudioHardwareDestroyAggregateDevice(multiOutputID)
        errorOutput("failed to switch output: OSStatus \(status)")
        return
    }

    jsonOutput([
        "multi_output_id": multiOutputID,
        "aggregate_id": aggregateID,
        "aggregate_uid": "com.meetingcli.aggregate",
        "aggregate_name": "MeetingCLI-Aggregate",
        "original_output_uid": currentOutputUID,
        "mic_uid": micUID
    ])
}

func destroyDevices(multiOutputID: UInt32, aggregateID: UInt32) {
    var errors: [String] = []

    let s1 = AudioHardwareDestroyAggregateDevice(AudioDeviceID(aggregateID))
    if s1 != noErr {
        errors.append("failed to destroy aggregate: OSStatus \(s1)")
    }

    let s2 = AudioHardwareDestroyAggregateDevice(AudioDeviceID(multiOutputID))
    if s2 != noErr {
        errors.append("failed to destroy multi-output: OSStatus \(s2)")
    }

    if errors.isEmpty {
        jsonOutput(["ok": true])
    } else {
        errorOutput(errors.joined(separator: "; "))
    }
}

func switchOutput(uid: String) {
    if setDefaultOutputDevice(uid: uid) {
        jsonOutput(["ok": true])
    } else {
        errorOutput("failed to switch output to \(uid)")
    }
}

// MARK: - Main

let args = CommandLine.arguments
guard args.count >= 2 else {
    errorOutput("usage: devices_helper.swift <command> [args...]")
    exit(1)
}

switch args[1] {
case "list-devices":
    listDevices()
case "current-output":
    currentOutput()
case "find-blackhole":
    findBlackhole()
case "create-devices":
    guard args.count >= 3 else {
        errorOutput("usage: create-devices <blackhole_uid>")
        exit(1)
    }
    createDevices(blackholeUID: args[2])
case "destroy-devices":
    guard args.count >= 4,
          let multiID = UInt32(args[2]),
          let aggID = UInt32(args[3]) else {
        errorOutput("usage: destroy-devices <multi_output_id> <aggregate_id>")
        exit(1)
    }
    destroyDevices(multiOutputID: multiID, aggregateID: aggID)
case "switch-output":
    guard args.count >= 3 else {
        errorOutput("usage: switch-output <device_uid>")
        exit(1)
    }
    switchOutput(uid: args[2])
default:
    errorOutput("unknown command: \(args[1])")
    exit(1)
}
