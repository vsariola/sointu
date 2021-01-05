#if defined (_WIN32)
#define _CRT_SECURE_NO_DEPRECATE
#include <windows.h>
#else
#include <sys/types.h>
#include <sys/stat.h>
#endif
#include <stdlib.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <stdbool.h>
#include <string.h>
#include <math.h>
#include <stdio.h>

#include TEST_HEADER
SUsample buf[SU_BUFFER_LENGTH];
SUsample filebuf[SU_BUFFER_LENGTH];

int main(int argc, char* argv[]) {
    FILE* f;
    char filename[256];
    int n;
    char test_name[] = TEST_NAME;
    char expected_output_folder[] = "expected_output/";
    char actual_output_folder[] = "actual_output/";
    long fsize;
    float max_diff;
    float diff;

    if (argc < 2) {
        fprintf(stderr, "usage: [test] path/to/expected_wave.raw");
    }

    #ifdef SU_LOAD_GMDLS
    su_load_gmdls();
    #endif

    su_render_song(buf);

#if defined (_WIN32)
    CreateDirectory(actual_output_folder, NULL);
#else
    mkdir(actual_output_folder, 0777);
#endif

    snprintf(filename, sizeof filename, "%s%s%s", actual_output_folder, test_name, ".raw");
    f = fopen(filename, "wb");
    fwrite((void*)buf, sizeof(SUsample), SU_BUFFER_LENGTH, f);
    fclose(f);

    f = fopen(argv[1], "rb");

    if (f == NULL) {
        printf("No expected waveform found!\n");
        goto fail;
    }

    fseek(f, 0, SEEK_END);
    fsize = ftell(f);
    fseek(f, 0, SEEK_SET);

    if (SU_BUFFER_LENGTH * sizeof(SUsample) < fsize) {
        printf("Sointu rendered shorter wave than expected\n");
        goto fail;
    }

    if (SU_BUFFER_LENGTH * sizeof(SUsample) > fsize) {
        printf("Sointu rendered longer wave than expected\n");
        goto fail;
    }

    fread((void*)filebuf, fsize, 1, f);
    fclose(f);
    f = NULL;

    max_diff = 0.0f;

    for (n = 0; n < SU_BUFFER_LENGTH; n++) {
        diff = (float)fabs((float)(buf[n] - filebuf[n])/SU_SAMPLE_RANGE);
        if (diff > 1e-3f || isnan(diff)) {
            printf("Sointu rendered different wave than expected\n");
            goto fail;
        }

        if (diff > max_diff) {
            max_diff = diff;
        }
    }

    if (max_diff > 1e-6) {
        printf("Warning: Sointu rendered almost correct wave, but a small maximum error of %f\n",max_diff);        
    }

    return 0;

fail:
    if (f != NULL) {
        fclose(f);
        f = NULL;
    }
    return 1;
}
