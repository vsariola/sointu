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
#ifndef GO4K_USE_16BIT_OUTPUT
	float* buf;
	float v;
	buf = (float*)malloc(test_max_samples * 2 * sizeof(float));
#else
	short* buf;
	short v;
	buf = (short*)malloc(test_max_samples * 2 * sizeof(short));
#endif

	

	if (buf == NULL) {
		printf("Could not allocate buffer\n");
		return 1;
	}

	_4klang_render(buf);

	snprintf(filename, sizeof filename, "%s%s", test_name, "_expected.raw");

	f = fopen(filename, "rb");

	if (f == NULL) {
		printf("No expected waveform found!\n");
		retval = 1;
		goto end;
	}

	n = 0;
	while (1) {		
		fread((void*)(&v), sizeof(v), 1, f);		
		if (feof(f)) {
			if (n == test_max_samples * 2) {
				retval = 0;
			}
			else {
				printf("4klang rendered longer wave than expected\n");
				retval = 1;
			}
			break;
		}
		if (n >= test_max_samples * 2) {
			printf("4klang rendered shorter wave than expected\n");
			retval = 1;
			break;
		}
		if (buf[n] != v) {
			printf("4klang rendered different wave than expected\n");
			retval = 1;
			break;
		}
		++n;		
	}	
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
	return retval;
}