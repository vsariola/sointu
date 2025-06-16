#include <alsa/asoundlib.h>
#include <pthread.h>
#include <stdio.h>
#include <stdint.h>
#include <time.h>
#include "physics_girl_st.h"

static SUsample sound_buffer[SU_LENGTH_IN_SAMPLES * SU_CHANNEL_COUNT];
static snd_pcm_t *pcm_handle;
static pthread_t render_thread, playback_thread;
static uint32_t render_thread_handle, playback_thread_handle;

void play() {
	snd_pcm_writei(pcm_handle, sound_buffer, SU_LENGTH_IN_SAMPLES);
}

int main(int argc, char **args) {
	// Unix does not have gm.dls, no need to ifdef and setup here.

	// We render in the background while playing already.
	render_thread_handle = pthread_create(&render_thread, 0, (void * (*)(void *))su_render_song, sound_buffer);

	// We can't start playing too early or the missing samples will be audible.
	sleep(2.);

	// Play the track.
	snd_pcm_open(&pcm_handle, "default", SND_PCM_STREAM_PLAYBACK, 0);
	snd_pcm_set_params(
		pcm_handle,
#ifdef SU_SAMPLE_FLOAT
		SND_PCM_FORMAT_FLOAT,
#else // SU_SAMPLE_FLOAT
		SND_PCM_FORMAT_S16_LE,
#endif // SU_SAMPLE_FLOAT
		SND_PCM_ACCESS_RW_INTERLEAVED,
		SU_CHANNEL_COUNT,
		SU_SAMPLE_RATE,
		0,
		SU_LENGTH_IN_SAMPLES
	);
	playback_thread_handle = pthread_create(&playback_thread, 0, (void *(*)(void *))play, 0);

	// This is for obtaining the playback time.
	snd_pcm_status_t *status;
	snd_pcm_status_malloc(&status);
	snd_htimestamp_t htime, htstart;
	snd_pcm_status(pcm_handle, status);
	snd_pcm_status_get_htstamp(status, &htstart);
	for(int sample; sample < SU_LENGTH_IN_SAMPLES; sample = (int)(((float)htime.tv_sec + (float)htime.tv_nsec * 1.e-9 - (float)htstart.tv_sec - (float)htstart.tv_nsec * 1.e-9) * SU_SAMPLE_RATE)) {
		snd_pcm_status(pcm_handle, status);
		snd_pcm_status_get_htstamp(status, &htime);
		printf("Sample: %d\n", sample);
		usleep(1000000 / 30);
	}
	snd_pcm_status_free(status);

	return 0;
}
