#include <stdio.h>
#include <stdint.h>
#include "physics_girl_st.h"
#define WIN32_LEAN_AND_MEAN
#define WIN32_EXTRA_LEAN
#include <Windows.h>
#include "mmsystem.h"
#include "mmreg.h"
#define CINTERFACE
#include <dsound.h>

#ifndef DSBCAPS_TRUEPLAYPOSITION // Not defined in MinGW dsound headers, so let's add it
#define DSBCAPS_TRUEPLAYPOSITION 0x00080000
#endif

SUsample sound_buffer[SU_LENGTH_IN_SAMPLES * SU_CHANNEL_COUNT];
WAVEFORMATEX wave_format = {
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
DSBUFFERDESC buffer_description = {
	sizeof(DSBUFFERDESC),
	DSBCAPS_GETCURRENTPOSITION2 | DSBCAPS_GLOBALFOCUS | DSBCAPS_TRUEPLAYPOSITION,
	SU_LENGTH_IN_SAMPLES * SU_SAMPLE_SIZE * SU_CHANNEL_COUNT,
	0,
	&wave_format,
	0
};

int main(int argc, char **args) {
	// Load gm.dls if necessary.
#ifdef SU_LOAD_GMDLS
	su_load_gmdls();
#endif // SU_LOAD_GMDLS

	HWND hWnd = GetForegroundWindow();
	if(hWnd == NULL) {
		hWnd = GetDesktopWindow();
	}

	LPDIRECTSOUND direct_sound;
	LPDIRECTSOUNDBUFFER direct_sound_buffer;
	DirectSoundCreate(0, &direct_sound, 0);
	IDirectSound_SetCooperativeLevel(direct_sound, hWnd, DSSCL_PRIORITY);
	IDirectSound_CreateSoundBuffer(direct_sound, &buffer_description, &direct_sound_buffer, NULL);
	
	LPVOID p1;
	DWORD l1;
	IDirectSoundBuffer_Lock(direct_sound_buffer, 0, SU_LENGTH_IN_SAMPLES * SU_CHANNEL_COUNT * SU_SAMPLE_SIZE, &p1, &l1, NULL, NULL, 0);
	CreateThread(0, 0, (LPTHREAD_START_ROUTINE)su_render_song, p1, 0, 0);
	IDirectSoundBuffer_Play(direct_sound_buffer, 0, 0, 0);

	// We need to handle windows messages properly while playing, as waveOutWrite is async.
	MSG msg = {0};
	DWORD last_play_cursor = 0;
	for(DWORD play_cursor = 0; play_cursor >= last_play_cursor; IDirectSoundBuffer_GetCurrentPosition(direct_sound_buffer, (DWORD*)&play_cursor, NULL)) {
		while (PeekMessageA(&msg, NULL, 0, 0, PM_REMOVE)) {
			TranslateMessage(&msg);
			DispatchMessageA(&msg);
		}

		last_play_cursor = play_cursor;
	}

	return 0;
}
