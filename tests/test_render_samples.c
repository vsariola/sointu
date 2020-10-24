#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include "../include/sointu.h"

#if UINTPTR_MAX == 0xffffffff // are we 32-bit?
#if defined(__clang__) || defined(__GNUC__)
#define CALLCONV __attribute__ ((stdcall))
#elif defined(_WIN32)
#define CALLCONV __stdcall // on 32-bit platforms, we just use stdcall, as all know it
#endif
#else // 64-bit
#define CALLCONV  // the asm will use honor honor correct x64 ABI on all 64-bit platforms
#endif

#define BPM 100
#define SAMPLE_RATE 44100
#define TOTAL_ROWS 16
#define SAMPLES_PER_ROW SAMPLE_RATE * 4 * 60 / (BPM * 16)
const int su_max_samples = SAMPLES_PER_ROW * TOTAL_ROWS;

void CALLCONV su_render(float* buffer) {
    SynthState* synthState;  
    const unsigned char commands[] = { su_envelope_id, // MONO
                                       su_envelope_id, // MONO
                                       su_out_id + 1,  // STEREO
                                       su_advance_id };// MONO
    const unsigned char values[] = { 64, 64, 64, 80, 128, // envelope 1
                                     95, 64, 64, 80, 128, // envelope 2
                                     128};
    int retval;
    synthState = malloc(sizeof(SynthState));
    memset(synthState, 0, sizeof(SynthState));
    memcpy(synthState->Commands, commands, sizeof(commands));
    memcpy(synthState->Values, values, sizeof(values));
    synthState->RandSeed = 1;
    synthState->SamplesPerRow = INT32_MAX;
    synthState->NumVoices = 1;
    synthState->Synth.Voices[0].Note = 64;
    retval = su_render_samples(synthState, su_max_samples / 2, buffer);
    synthState->Synth.Voices[0].Release++;
    buffer = buffer + su_max_samples;
    retval = su_render_samples(synthState, su_max_samples / 2, buffer);
    free(synthState);
	return;
}
