#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <sointu/sointu.h>
#include "test_render_samples.h"

void SU_CALLCONV su_render_song(float* buffer) {
    Synth* synth;
    const unsigned char commands[] = { su_envelope_id, // MONO
                                       su_envelope_id, // MONO
                                       su_out_id + 1,  // STEREO
                                       su_advance_id };// MONO
    const unsigned char values[] = { 64, 64, 64, 80, 128, // envelope 1
                                     95, 64, 64, 80, 128, // envelope 2
                                     128};
    int retval;
    int samples;
    int time;
    // initialize Synth
    synth = (Synth*)malloc(sizeof(Synth));    
    memset(synth, 0, sizeof(Synth));
    memcpy(synth->Commands, commands, sizeof(commands));
    memcpy(synth->Values, values, sizeof(values));
    synth->NumVoices = 1;
    synth->Polyphony = 0;
    synth->RandSeed = 1;
    // triger first voice    
    synth->SynthWrk.Voices[0].Note = 64;
    samples = SU_MAX_SAMPLES / 2;
    time = INT32_MAX;
    retval = su_render(synth, buffer, &samples, &time);
    synth->SynthWrk.Voices[0].Release++;
    buffer = buffer + SU_MAX_SAMPLES;
    samples = SU_MAX_SAMPLES / 2;
    time = INT32_MAX;
    retval = su_render(synth, buffer, &samples, &time);
    free(synth);
    return;
}
