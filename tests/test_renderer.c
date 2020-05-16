#include <stdio.h>
#include <stdlib.h>
#include <sys/types.h>
#include <sys/stat.h>

#if defined (_WIN32)
#include <windows.h>
#else
#include <sys/types.h>
#include <sys/stat.h>
#endif

#include <math.h>

extern void __stdcall su_render();
extern int su_max_samples;

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
	boolean small_difference;
	double diff;
#ifndef SU_USE_16BIT_OUTPUT
	float* buf = NULL;
	float* filebuf = NULL;
	float v;
	bufsize = su_max_samples * 2 * sizeof(float);
	buf = (float*)malloc(bufsize);
#else
	short* buf = NULL;
	short* filebuf = NULL;
	short v;
	bufsize = su_max_samples * 2 * sizeof(short);
	buf = (short*)malloc(bufsize);
#endif	

	if (buf == NULL) {
		printf("Could not allocate buffer for 4klang rendering\n");
		return 1;
	}

	su_render(buf);

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

#ifndef SU_USE_16BIT_OUTPUT	
	filebuf = (float*)malloc(bufsize);
#else	
	filebuf = (short*)malloc(bufsize);
#endif	

	if (filebuf == NULL) {
		printf("Could not allocate buffer for file contents\n");
		goto fail;
	}

	fread((void*)filebuf, su_max_samples * 2, sizeof(*filebuf), f);

	small_difference = FALSE;

	for (n = 0; n < su_max_samples * 2; n++) {
		diff = (double)(buf[n]) - (double)(filebuf[n]);
#ifdef SU_USE_16BIT_OUTPUT	
		diff = diff / 32768.0f;
#endif
		diff = fabs(diff);
		if (diff > 1e-3f || isnan(diff)) {
			printf("4klang rendered different wave than expected\n");
			goto fail;
		}
		else if (diff > 0.0f) {
			small_difference = TRUE;
		}
	}

	if (small_difference) {
		printf("4klang rendered almost correct wave, but with small errors (< 1e-3)\n");
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
	fwrite((void*)buf, sizeof(*buf), 2 * su_max_samples, f);
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