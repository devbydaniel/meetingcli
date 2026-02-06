#import "capture_darwin.h"
#import <AVFoundation/AVFoundation.h>
#import <ScreenCaptureKit/ScreenCaptureKit.h>

@class AudioHandler;

static NSFileHandle *g_fileHandle = nil;
static uint32_t g_dataSize = 0;
static NSLock *g_lock = nil;
static SCStream *g_stream = nil;
static AudioHandler *g_handler = nil;

static void write_wav_header(NSFileHandle *fh, uint32_t dataSize) {
    uint32_t sampleRate = 16000;
    uint16_t channels = 1;
    uint16_t bitsPerSample = 16;
    uint32_t byteRate = sampleRate * channels * bitsPerSample / 8;
    uint16_t blockAlign = channels * bitsPerSample / 8;

    uint8_t header[44];
    memcpy(header + 0, "RIFF", 4);
    uint32_t fileSize = 36 + dataSize;
    memcpy(header + 4, &fileSize, 4);
    memcpy(header + 8, "WAVE", 4);
    memcpy(header + 12, "fmt ", 4);
    uint32_t subchunkSize = 16;
    memcpy(header + 16, &subchunkSize, 4);
    uint16_t format = 1;
    memcpy(header + 20, &format, 2);
    memcpy(header + 22, &channels, 2);
    memcpy(header + 24, &sampleRate, 4);
    memcpy(header + 28, &byteRate, 4);
    memcpy(header + 32, &blockAlign, 2);
    memcpy(header + 34, &bitsPerSample, 2);
    memcpy(header + 36, "data", 4);
    memcpy(header + 40, &dataSize, 4);

    [fh seekToFileOffset:0];
    [fh writeData:[NSData dataWithBytes:header length:44]];
}

// ScreenCaptureKit stream output handler
@interface AudioHandler : NSObject <SCStreamOutput>
@end

@implementation AudioHandler

- (void)stream:(SCStream *)stream
    didOutputSampleBuffer:(CMSampleBufferRef)sampleBuffer
               ofType:(SCStreamOutputType)type {
    if (type != SCStreamOutputTypeAudio) return;

    CMFormatDescriptionRef formatDesc = CMSampleBufferGetFormatDescription(sampleBuffer);
    if (!formatDesc) return;

    const AudioStreamBasicDescription *asbd =
        CMAudioFormatDescriptionGetStreamBasicDescription(formatDesc);
    if (!asbd) return;

    CMBlockBufferRef blockBuffer = CMSampleBufferGetDataBuffer(sampleBuffer);
    if (!blockBuffer) return;

    size_t length = 0;
    char *dataPointer = NULL;
    OSStatus status = CMBlockBufferGetDataPointer(blockBuffer, 0, NULL, &length, &dataPointer);
    if (status != noErr || !dataPointer) return;

    int channelCount = asbd->mChannelsPerFrame;
    int frameCount = (int)(length / (sizeof(float) * channelCount));
    float *src = (float *)dataPointer;

    // Downsample 48kHz→16kHz (factor 3), stereo→mono, float32→int16
    int outCapacity = (frameCount / 3 + 1);
    int16_t *outBuf = (int16_t *)malloc(outCapacity * sizeof(int16_t));
    int outCount = 0;

    for (int i = 0; i < frameCount; i += 3) {
        float sample;
        if (channelCount >= 2) {
            // Non-interleaved: ch0 data then ch1 data
            sample = (src[i] + src[frameCount + i]) * 0.5f;
        } else {
            sample = src[i];
        }
        if (sample > 1.0f) sample = 1.0f;
        if (sample < -1.0f) sample = -1.0f;
        outBuf[outCount++] = (int16_t)(sample * 32767.0f);
    }

    NSData *data = [NSData dataWithBytesNoCopy:outBuf
                                        length:outCount * sizeof(int16_t)
                                  freeWhenDone:YES];

    [g_lock lock];
    @try {
        [g_fileHandle writeData:data];
        g_dataSize += (uint32_t)(outCount * sizeof(int16_t));
    } @catch (NSException *e) {}
    [g_lock unlock];
}

@end

int capture_start(const char *output_path) {
    g_lock = [[NSLock alloc] init];
    g_dataSize = 0;

    NSString *path = [NSString stringWithUTF8String:output_path];
    [[NSFileManager defaultManager] createFileAtPath:path contents:nil attributes:nil];
    g_fileHandle = [NSFileHandle fileHandleForWritingAtPath:path];
    if (!g_fileHandle) return -1;

    uint8_t header[44] = {0};
    [g_fileHandle writeData:[NSData dataWithBytes:header length:44]];

    __block int result = -1;
    dispatch_semaphore_t sem = dispatch_semaphore_create(0);

    [SCShareableContent getShareableContentExcludingDesktopWindows:NO
                                              onScreenWindowsOnly:NO
                                                completionHandler:^(SCShareableContent *content, NSError *error) {
        if (error || content.displays.count == 0) {
            dispatch_semaphore_signal(sem);
            return;
        }

        SCDisplay *display = content.displays.firstObject;
        SCContentFilter *filter = [[SCContentFilter alloc] initWithDisplay:display excludingWindows:@[]];

        SCStreamConfiguration *config = [[SCStreamConfiguration alloc] init];
        config.capturesAudio = YES;
        config.sampleRate = 48000;
        config.channelCount = 2;
        config.width = 2;
        config.height = 2;
        CMTime interval;
        interval.value = 1;
        interval.timescale = 1;
        interval.flags = kCMTimeFlags_Valid;
        interval.epoch = 0;
        config.minimumFrameInterval = interval;

        g_handler = [[AudioHandler alloc] init];
        g_stream = [[SCStream alloc] initWithFilter:filter configuration:config delegate:nil];

        NSError *addError = nil;
        [g_stream addStreamOutput:g_handler
                             type:SCStreamOutputTypeAudio
                sampleHandlerQueue:dispatch_get_global_queue(QOS_CLASS_USER_INTERACTIVE, 0)
                            error:&addError];
        if (addError) {
            dispatch_semaphore_signal(sem);
            return;
        }

        [g_stream startCaptureWithCompletionHandler:^(NSError *startError) {
            if (!startError) result = 0;
            dispatch_semaphore_signal(sem);
        }];
    }];

    dispatch_time_t timeout = dispatch_time(DISPATCH_TIME_NOW, 10LL * NSEC_PER_SEC);
    if (dispatch_semaphore_wait(sem, timeout) != 0) {
        [g_fileHandle closeFile];
        g_fileHandle = nil;
        return -1;
    }

    if (result != 0) {
        [g_fileHandle closeFile];
        g_fileHandle = nil;
    }

    return result;
}

int capture_stop(void) {
    if (!g_stream) return 0;

    dispatch_semaphore_t sem = dispatch_semaphore_create(0);
    [g_stream stopCaptureWithCompletionHandler:^(NSError *error) {
        dispatch_semaphore_signal(sem);
    }];
    dispatch_time_t timeout = dispatch_time(DISPATCH_TIME_NOW, 5LL * NSEC_PER_SEC);
    dispatch_semaphore_wait(sem, timeout);

    g_stream = nil;
    g_handler = nil;

    [g_lock lock];
    if (g_fileHandle) {
        write_wav_header(g_fileHandle, g_dataSize);
        [g_fileHandle closeFile];
        g_fileHandle = nil;
    }
    [g_lock unlock];

    return 0;
}
