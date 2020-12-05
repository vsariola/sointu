#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <sointu/sointu.h>

#define BPM 100
#define SAMPLE_RATE 44100
#define TOTAL_ROWS 16
#define SAMPLES_PER_ROW SAMPLE_RATE * 4 * 60 / (BPM * 16)
const int su_max_samples = SAMPLES_PER_ROW * TOTAL_ROWS;

int main(int argc, char* argv[]) {
    Synth* synth;
    float* buffer;
    const unsigned char commands[] = { su_envelope_id, // MONO
                                       su_envelope_id, // MONO
                                       su_out_id + 1,  // STEREO
                                       su_advance_id };// MONO
    const unsigned char values[] = { 64, 64, 64, 80, 128, // envelope 1
                                     95, 64, 64, 80, 128, // envelope 2
                                     128 };    
    int errcode;
    int time;
    int samples;
    int totalrendered;
    int retval;    
    // initialize Synth
    synth = (Synth*)malloc(sizeof(Synth));    
    memcpy(synth->Commands, commands, sizeof(commands));
    memcpy(synth->Values, values, sizeof(values));
    synth->NumVoices = 1;
    synth->Polyphony = 0;
    synth->RandSeed = 1;
    // initialize Buffer
    buffer = (float*)malloc(2 * sizeof(float) * su_max_samples);
    // triger first voice    
    synth->SynthWrk.Voices[0].Note = 64;
    totalrendered = 0;
    // First check that when we render using su_render with 0 time
    // we get nothing done    
    samples = su_max_samples;
    time = 0;
    errcode = su_render(synth, buffer, &samples, &time);
    if (errcode != 0)
        goto fail;
    if (samples > 0)
    {
        printf("su_render rendered samples, despite it should not\n");
        goto fail;
    }    
    if (time > 0)
    {
        printf("su_render advanced time, despite it should not\n");
        goto fail;
    }
    // Then check that when we render using su_render with 0 samples,
    // we get nothing done    
    samples = 0;
    time = INT32_MAX;
    errcode = su_render(synth, buffer, &samples, &time);
    if (errcode != 0)         
        goto fail;
    if (samples > 0)
    {
        printf("su_render rendered samples, despite it should not\n");
        goto fail;
    }
    if (time > 0)
    {
        printf("su_render advanced time, despite it should not\n");
        goto fail;
    }
    // Then check that each time we call render, only SAMPLES_PER_ROW
    // number of samples are rendered
    for (int i = 0; i < 16; i++) {
        // Simulate "small buffers" i.e. render a buffer with 1 sample
        // check that buffer full
        samples = 1;
        time = INT32_MAX;
        errcode = su_render(synth, &buffer[totalrendered*2], &samples, &time);
        if (errcode != 0)
            goto fail;
        totalrendered += samples;
        if (samples != 1)
        {
            printf("su_render should have return 1, as it should have believed buffer is full");
            goto fail;
        }
        if (time != 1)
        {
            printf("su_render should have advanced the time also by one");
            goto fail;
        }        
        samples = SAMPLES_PER_ROW - 1;
        time = INT32_MAX;
        errcode = su_render(synth, &buffer[totalrendered * 2], &samples, &time);
        if (errcode != 0)
            goto fail;
        totalrendered += samples;
        if (samples != SAMPLES_PER_ROW - 1)
        {
            printf("su_render should have return SAMPLES_PER_ROW - 1, as it should have believed buffer is full");
            goto fail;
        }
        if (time != SAMPLES_PER_ROW - 1)
        {
            printf("su_render should have advanced the time also by SAMPLES_PER_ROW - 1");
            goto fail;
        }
        if (i == 8)
            synth->SynthWrk.Voices[0].Release++;
    }
    if (totalrendered != su_max_samples)
    {
        printf("su_render should have rendered a total of su_max_samples");
        goto fail;
    }    
    retval = 0;
finish:
    free(synth);
    free(buffer);
    return retval;
fail:
    if (errcode > 0) {
        if ((errcode & 0xFF00) != 0)
            printf("FPU stack was not empty on exit\n");
        if ((errcode & 0x04) != 0)
            printf("FPU zero divide\n");
        if ((errcode & 0x01) != 0)
            printf("FPU invalid operation\n");
        if ((errcode & 0x40) != 0)
            printf("FPU stack error\n");
    }
    retval = 1;
    goto finish;
}
