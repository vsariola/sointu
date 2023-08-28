#include <stdio.h>
#include <stdint.h>
#include "physics_girl_st.h"

#define WAVE_FORMAT_PCM 0x1
#define WAVE_FORMAT_IEEE_FLOAT 0x3

static SUsample sound_buffer[SU_LENGTH_IN_SAMPLES * SU_CHANNEL_COUNT];

#pragma pack(push, 1)
typedef struct {
	char riff[4];
	uint32_t file_size;
	char wavefmt[8];
} riff_header_t;

typedef struct {
	char data[4];
	uint32_t data_size;
} data_header_t;

typedef struct {
	riff_header_t riff_header;
	uint32_t riff_header_size;
	uint16_t sample_type;
	uint16_t channel_count;
	uint32_t sample_rate;
	uint32_t bytes_per_second;
	uint16_t bytes_per_channel;
	uint16_t bits_per_sample;
	data_header_t data_header;
} wave_header_t;
#pragma pack(pop)

int main(int argc, char **args) {
	wave_header_t wave_header = {
		.riff_header = (riff_header_t) {
			.riff = "RIFF",
			.file_size = sizeof(wave_header_t) + SU_LENGTH_IN_SAMPLES * SU_SAMPLE_SIZE * SU_CHANNEL_COUNT,
			.wavefmt = "WAVEfmt ",
		},
		.riff_header_size = sizeof(riff_header_t),
	#ifdef SU_SAMPLE_FLOAT
		.sample_type = WAVE_FORMAT_IEEE_FLOAT,
	#else // SU_SAMPLE_FLOAT
		.sample_type = WAVE_FORMAT_PCM,
	#endif // SU_SAMPLE_FLOAT
		.channel_count = SU_CHANNEL_COUNT,
		.sample_rate = SU_SAMPLE_RATE,
		.bytes_per_second = SU_SAMPLE_SIZE * SU_SAMPLE_RATE * SU_CHANNEL_COUNT,
		.bytes_per_channel = SU_SAMPLE_SIZE * SU_CHANNEL_COUNT,
		.bits_per_sample = SU_SAMPLE_SIZE * 8,
		.data_header = (data_header_t) {
			.data = "data",
			.data_size = sizeof(data_header_t) + SU_LENGTH_IN_SAMPLES * SU_SAMPLE_SIZE * SU_CHANNEL_COUNT
		}
	};

	// Load gm.dls if necessary.
#ifdef SU_LOAD_GMDLS
    su_load_gmdls();
#endif // SU_LOAD_GMDLS

	su_render_song(sound_buffer);

	FILE *file = fopen("physics_girl_st.wav", "wb");
	fwrite(&wave_header, sizeof(wave_header_t), 1, file);
	fwrite((uint8_t *)sound_buffer, 1, SU_LENGTH_IN_SAMPLES * SU_SAMPLE_SIZE * SU_CHANNEL_COUNT, file);
	fclose(file);

	return 0;
}
