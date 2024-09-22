#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <sointu.h>

#define BPM 100
#define SAMPLE_RATE 44100
#define LENGTH_IN_ROWS 16
#define SAMPLES_PER_ROW SAMPLE_RATE * 4 * 60 / (BPM * 16)
const int su_max_samples = SAMPLES_PER_ROW * LENGTH_IN_ROWS;

int main(int argc, char* argv[])
{
    Synth* synth;
    float* buffer;
    // The patch is invalid and overflows the stack. This should still exit cleanly, but used to hard crash.
    // See: https://github.com/vsariola/sointu/issues/149
    const unsigned char opcodes[] = { SU_OSCILLATOR_ID + 1, // STEREO                                     
                                     SU_ADVANCE_ID };
    const unsigned char operands[] = { 69, 74, 0, 0, 82, 128, 128 };
    int errcode;
    int time;
    int samples;
    int totalrendered;
    int retval;
    // initialize Synth
    synth = (Synth*)malloc(sizeof(Synth));
    memset(synth, 0, sizeof(Synth));
    memcpy(synth->Opcodes, opcodes, sizeof(opcodes));
    memcpy(synth->Operands, operands, sizeof(operands));
    synth->NumVoices = 3;
    synth->Polyphony = 6;
    synth->RandSeed = 1;
    synth->SampleOffsets[0].Start = 91507;
    synth->SampleOffsets[0].LoopStart = 5448;
    synth->SampleOffsets[0].LoopLength = 563;
    // initialize Buffer
    buffer = (float*)malloc(2 * sizeof(float) * su_max_samples);
    // triger first voice
    synth->SynthWrk.Voices[0].Note = 64;
    synth->SynthWrk.Voices[0].Sustain = 1;
    totalrendered = 0;
    samples = su_max_samples;
    time = INT32_MAX;
    retval = 0;    
    errcode = su_render(synth, buffer, &samples, &time);
    if (errcode != 0x1041) {
        retval = 1;
        printf("su_render should have return errcode 0x1401, got 0x%08x\n", errcode);
    }       
    free(synth);
    free(buffer);
    return retval;
}
