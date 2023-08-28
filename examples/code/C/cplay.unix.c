#include <alsa/asoundlib.h>
#include <pthread.h>
#include <stdio.h>
#include <stdint.h>
#include "physics_girl_st.h"

static SUsample sound_buffer[SU_LENGTH_IN_SAMPLES * SU_CHANNEL_COUNT];
static snd_pcm_t *pcm_handle;
static pthread_t render_thread;
static uint32_t render_thread_handle;

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
	snd_pcm_writei(pcm_handle, sound_buffer, SU_LENGTH_IN_SAMPLES);

	return 0;
}
