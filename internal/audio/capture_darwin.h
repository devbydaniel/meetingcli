#ifndef CAPTURE_DARWIN_H
#define CAPTURE_DARWIN_H

// Start capturing system audio. Writes 16kHz mono 16-bit PCM WAV to the given path.
// Audio is streamed to disk continuously — no ring buffer.
// Returns 0 on success, -1 on error.
int capture_start(const char *output_path);

// Stop capturing. Finalizes the WAV header and closes the file.
// Returns 0 on success.
int capture_stop(void);

#endif
