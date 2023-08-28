#include <stdio.h>
#include <stdint.h>
#include "physics_girl_st.h"
#define WIN32_LEAN_AND_MEAN
#define WIN32_EXTRA_LEAN
#include <Windows.h>
#include "mmsystem.h"
#include "mmreg.h"

SUsample sound_buffer[SU_LENGTH_IN_SAMPLES * SU_CHANNEL_COUNT];
HWAVEOUT	wave_out_handle;
WAVEFORMATEX WaveFMT = {
#ifdef SU_SAMPLE_FLOAT
	WAVE_FORMAT_IEEE_FLOAT,
#else
	WAVE_FORMAT_PCM,
#endif
	SU_CHANNEL_COUNT,
	SU_SAMPLE_RATE,
	SU_SAMPLE_RATE * SU_SAMPLE_SIZE * SU_CHANNEL_COUNT,
	SU_SAMPLE_SIZE * SU_CHANNEL_COUNT,
	SU_SAMPLE_SIZE*8,
	0
};
WAVEHDR WaveHDR = {
	(LPSTR)sound_buffer, 
	SU_LENGTH_IN_SAMPLES * SU_SAMPLE_SIZE * SU_CHANNEL_COUNT,
	0,
	0,
	WHDR_PREPARED,
	0,
	0,
	0
};
MMTIME MMTime = {
	TIME_SAMPLES,
	0
};

int main(int argc, char **args) {
	// Load gm.dls if necessary.
#ifdef SU_LOAD_GMDLS
	su_load_gmdls();
#endif // SU_LOAD_GMDLS

	CreateThread(0, 0, (LPTHREAD_START_ROUTINE)su_render_song, sound_buffer, 0, 0);

	// We render in the background while playing already. Fortunately,
	// Windows is slow with the calls below, so we're not worried that
	// we don't have enough samples ready before the track starts.
	waveOutOpen(&wave_out_handle, WAVE_MAPPER, &WaveFMT, 0, 0, CALLBACK_NULL);
	waveOutWrite(wave_out_handle, &WaveHDR, sizeof(WaveHDR));

	// We need to handle windows messages properly while playing, as waveOutWrite is async.
	for(MSG msg = {0}; MMTime.u.sample != SU_LENGTH_IN_SAMPLES; waveOutGetPosition(wave_out_handle, &MMTime, sizeof(MMTIME))) {
		while (PeekMessageA(&msg, NULL, 0, 0, PM_REMOVE)) {
			TranslateMessage(&msg);
			DispatchMessageA(&msg);
		}
	}

	return 0;
}
