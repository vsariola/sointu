#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include "../include/sointu.h"

#define BPM 100
#define SAMPLE_RATE 44100
#define TOTAL_ROWS 16
#define SAMPLES_PER_ROW SAMPLE_RATE * 4 * 60 / (BPM * 16)
const int su_max_samples = SAMPLES_PER_ROW * TOTAL_ROWS;

int main(int argc, char* argv[]) {
    SynthState* synthState;
    float* buffer;
    const unsigned char commands[] = { su_envelope_id, // MONO
                                       su_envelope_id, // MONO
                                       su_out_id + 1,  // STEREO
                                       su_advance_id };// MONO
    const unsigned char values[] = { 64, 64, 64, 80, 128, // envelope 1
                                     95, 64, 64, 80, 128, // envelope 2
                                     128 };
    int remaining, remainingOut;
    int retval;
    synthState = (SynthState*)malloc(sizeof(SynthState));
    buffer = (float*)malloc(2 * sizeof(float) * su_max_samples);
    memset(synthState, 0, sizeof(SynthState));
    memcpy(synthState->Commands, commands, sizeof(commands));
    memcpy(synthState->Values, values, sizeof(values));
    synthState->RandSeed = 1;
    synthState->NumVoices = 1;
    synthState->Synth.Voices[0].Note = 64;
    remaining = su_max_samples;
    // First check that when RowLen = 0, we render nothing and remaining does not change
    synthState->SamplesPerRow = 0;
    if (su_render_samples(synthState, remaining, buffer) != remaining)
    {
        printf("su_render_samples rendered samples despite number of samples per row being 0");
        goto fail;
    }
    // Then check that each time we call render, only SAMPLES_PER_ROW
    // number of samples are rendered
    synthState->SamplesPerRow = SAMPLES_PER_ROW;
    for (int i = 0; i < 16; i++) {
        // Simulate "small buffers" i.e. render a buffer with 1 sample
        // check that buffer full
        remainingOut = su_render_samples(synthState, 1, &buffer[2 * (su_max_samples - remaining)]);
        if (remainingOut != -1)
        {
            printf("su_render_samples should have return -1, as it should have believed buffer is full");
            goto fail;
        }
        if (synthState->RowTick != 1)
        {
            printf("su_render_samples RowTick should be at 1 after rendering 1 tick of a row");
            goto fail;
        }
        remaining--; // we rendered just one sample
        remainingOut = su_render_samples(synthState, remaining, &buffer[2 * (su_max_samples - remaining)]);
        if (remainingOut != remaining - SAMPLES_PER_ROW + 1)
        {
            printf("su_render_samples did not render SAMPLES_PER_ROW, despite rowLen being SAMPLES_PER_ROW");
            goto fail;
        }
        if (synthState->RowTick != 0)
        {
            printf("The row should be have been reseted");
            goto fail;
        }
        remaining = remainingOut;
        if (i == 8)
            synthState->Synth.Voices[0].Release++;
    }
    if (remaining != 0) {
        printf("The buffer should be full and row finished");
        goto fail;
    }
    // Finally, now that there is no more buffer remaining, should return -1
    if (su_render_samples(synthState, remaining, &buffer[2 * (su_max_samples - remaining)]) != -1)
    {
        printf("su_render_samples should have ran out of buffer and thus return -1");
        goto fail;
    }
    retval = 0;
finish:
    free(synthState);
    free(buffer);
    return retval;
fail:
    retval = 1;
    goto finish;
}
