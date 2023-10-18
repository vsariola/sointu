#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <sointu.h>
#include "test_render_samples.h"

void SU_CALLCONV su_render_song(float *buffer)
{
    Synth *synth;
    const unsigned char opcodes[] = {SU_ENVELOPE_ID,       // MONO
                                     SU_ENVELOPE_ID,       // MONO
                                     SU_OUT_ID + 1,        // STEREO
                                     SU_ADVANCE_ID};       // MONO
    const unsigned char operands[] = {64, 64, 64, 80, 128, // envelope 1
                                      95, 64, 64, 80, 128, // envelope 2
                                      128};
    int retval;
    int samples;
    int time;
    // initialize Synth
    synth = (Synth *)malloc(sizeof(Synth));
    memset(synth, 0, sizeof(Synth));
    memcpy(synth->Opcodes, opcodes, sizeof(opcodes));
    memcpy(synth->Operands, operands, sizeof(operands));
    synth->NumVoices = 1;
    synth->Polyphony = 0;
    synth->RandSeed = 1;
    // triger first voice
    synth->SynthWrk.Voices[0].Note = 64;
    synth->SynthWrk.Voices[0].Sustain = 1;
    samples = SU_LENGTH_IN_SAMPLES / 2;
    time = INT32_MAX;
    retval = su_render(synth, buffer, &samples, &time);
    synth->SynthWrk.Voices[0].Sustain = 0;
    buffer = buffer + SU_LENGTH_IN_SAMPLES;
    samples = SU_LENGTH_IN_SAMPLES / 2;
    time = INT32_MAX;
    retval = su_render(synth, buffer, &samples, &time);
    free(synth);
    return;
}
