#include <stdio.h>
#include <stdlib.h>

extern void __stdcall _4klang_render();
extern int test_max_samples;

int main(int argc, char* argv[]) {
	FILE* f;
	char filename[256];
	int n;
	int retval;
	char test_name[] = TEST_NAME;
	long fsize;
	long bufsize;
#ifndef GO4K_USE_16BIT_OUTPUT
	float* buf = NULL;
	float* filebuf = NULL;
	float v;
	bufsize = test_max_samples * 2 * sizeof(float);
	buf = (float*)malloc(bufsize);
#else
	short* buf = NULL;
	short* filebuf = NULL;
	short v;
	bufsize = test_max_samples * 2 * sizeof(short);
	buf = (short*)malloc(bufsize);
#endif	

	if (buf == NULL) {
		printf("Could not allocate buffer for 4klang rendering\n");
		return 1;
	}

	_4klang_render(buf);

	snprintf(filename, sizeof filename, "%s%s", test_name, "_expected.raw");

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

#ifndef GO4K_USE_16BIT_OUTPUT	
	filebuf = (float*)malloc(bufsize);
#else	
	filebuf = (short*)malloc(bufsize);
#endif	

	if (filebuf == NULL) {
		printf("Could not allocate buffer for file contents\n");
		goto fail;
	}

	fread((void*)filebuf, test_max_samples * 2, sizeof(*filebuf), f);

	for (n = 0; n < test_max_samples * 2; n++) {
		if (buf[n] != filebuf[n]) {
			printf("4klang rendered different wave than expected\n");
			goto fail;
		}
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
	
	snprintf(filename, sizeof filename, "%s%s", test_name, "_got.raw");
	f = fopen(filename, "wb");	
	fwrite((void*)buf, sizeof(*buf), 2 * test_max_samples, f);
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