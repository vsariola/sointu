#include <stdio.h>
#include <stdlib.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <stdbool.h>
#include <string.h>

#if defined (_WIN32)
#include <windows.h>
#else
#include <sys/types.h>
#include <sys/stat.h>
#endif

#include <math.h>
#include TEST_HEADER

int main(int argc, char* argv[]) {
    FILE* f;
    char filename[256];
    int n;
    int retval;
    char test_name[] = TEST_NAME;
    char expected_output_folder[] = "expected_output/";
    char actual_output_folder[] = "actual_output/";
    long fsize;
    long bufsize;
    float max_diff;
    float diff;
    SUsample* buf = NULL;
    SUsample* filebuf = NULL;
    SUsample v;
    bufsize = SU_BUFFER_LENGTH * sizeof(SUsample);
    buf = (SUsample*)malloc(bufsize);
    memset(buf, 0, bufsize);

    if (buf == NULL) {
        printf("Could not allocate buffer for 4klang rendering\n");
        return 1;
    }

    #ifdef SU_LOAD_GMDLS
    su_load_gmdls();
    #endif

    su_render_song(buf);

    snprintf(filename, sizeof filename, "%s%s%s", expected_output_folder, test_name, ".raw");

    f = fopen(filename, "rb");

    if (f == NULL) {
        printf("No expected waveform found!\n");
        goto fail;
    }

    fseek(f, 0, SEEK_END);
    fsize = ftell(f);
    fseek(f, 0, SEEK_SET);

    if (bufsize < fsize) {
        printf("4klang rendered shorter wave than expected\n");
        goto fail;
    }

    if (bufsize > fsize) {
        printf("4klang rendered longer wave than expected\n");
        goto fail;
    }

    filebuf = (SUsample*)malloc(bufsize);

    if (filebuf == NULL) {
        printf("Could not allocate buffer for file contents\n");
        goto fail;
    }

    fread((void*)filebuf, fsize, 1, f);

    max_diff = 0.0f;

    for (n = 0; n < SU_BUFFER_LENGTH; n++) {
        diff = fabs((float)(buf[n] - filebuf[n])/SU_SAMPLE_RANGE);
        if (diff > 1e-3f || isnan(diff)) {
            printf("4klang rendered different wave than expected\n");
            goto fail;
        }
        
        if (diff > max_diff) {
            max_diff = diff;
        }
    }

    if (max_diff > 1e-6) {
        printf("4klang rendered almost correct wave, but a small maximum error of %f\n",max_diff);
        goto fail;
    }

success:
    retval = 0;
    goto end;
fail:
    retval = 1;
end:

    if (f != 0) {
        fclose(f);
        f = 0;
    }

#if defined (_WIN32)
    CreateDirectory(actual_output_folder, NULL);
#else
    mkdir(actual_output_folder, 0777);
#endif

    snprintf(filename, sizeof filename, "%s%s%s", actual_output_folder, test_name, ".raw");
    f = fopen(filename, "wb");
    fwrite((void*)buf, 1, bufsize, f);
    fclose(f);

    if (buf != 0) {
        free(buf);
        buf = 0;
    }

    if (filebuf != 0) {
        free(filebuf);
        filebuf = 0;
    }
    return retval;
}
