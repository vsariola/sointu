#define PY_SSIZE_T_CLEAN
#include "Python.h"

#include TRACK_HEADER

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>

SUsample sound_buffer[SU_LENGTH_IN_SAMPLES * SU_CHANNEL_COUNT];
#ifdef WIN32
#define WIN32_LEAN_AND_MEAN
#define WIN32_EXTRA_LEAN
#include <Windows.h>
#include "mmsystem.h"
#include "mmreg.h"
#define CINTERFACE
#include <dsound.h>

static WAVEFORMATEX wave_format = {
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

static HWND hWnd;
static LPDIRECTSOUND direct_sound;
static LPDIRECTSOUNDBUFFER direct_sound_buffer;
static LPVOID p1;
static DWORD l1;
/*
 * Note: The DirectSound API design is annoyingly bad. The
 *       playback position obtained by `IDirectSoundBuffer_GetCurrentPosition`
 *       will wrap around after the track is finished, so we really
 *       only have shady means of checking whether or not playback has finished.
 */
static DWORD last_play_cursor = 0;
#endif /* WIN32 */

#ifdef UNIX
#include <alsa/asoundlib.h>
#include <pthread.h>

static SUsample sound_buffer[SU_LENGTH_IN_SAMPLES * SU_CHANNEL_COUNT];
static snd_pcm_t *pcm_handle;
static pthread_t render_thread;
static uint32_t render_thread_handle;
static pthread_t playback_thread;
static uint32_t playback_thread_handle;

static int _snd_pcm_writei(void *params) {
    (void) params;
    snd_pcm_writei(pcm_handle, sound_buffer, SU_LENGTH_IN_SAMPLES);
    return 0;
}
#endif /* UNIX */

static PyObject *sointuError;

static PyObject *sointu_play_song(PyObject *self, PyObject *args) {
#ifdef WIN32

#ifdef SU_LOAD_GMDLS
    su_load_gmdls();
#endif // SU_LOAD_GMDLS

    hWnd = GetForegroundWindow();
	if(hWnd == NULL) {
		hWnd = GetDesktopWindow();
	}

	DirectSoundCreate(0, &direct_sound, 0);
	IDirectSound_SetCooperativeLevel(direct_sound, hWnd, DSSCL_PRIORITY);
	IDirectSound_CreateSoundBuffer(direct_sound, &buffer_description, &direct_sound_buffer, NULL);
	IDirectSoundBuffer_Lock(direct_sound_buffer, 0, SU_LENGTH_IN_SAMPLES * SU_CHANNEL_COUNT * SU_SAMPLE_SIZE, &p1, &l1, NULL, NULL, 0);
	CreateThread(0, 0, (LPTHREAD_START_ROUTINE)su_render_song, p1, 0, 0);
	IDirectSoundBuffer_Play(direct_sound_buffer, 0, 0, 0);
    
#endif /* WIN32 */

#ifdef UNIX
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

    // Enable playback time querying.
    snd_pcm_sw_params_t *swparams;
    snd_pcm_sw_params_alloca(&swparams);
    snd_pcm_sw_params_current(pcm_handle, swparams);
    snd_pcm_sw_params_get_tstamp_mode(swparams, SND_PCM_TSTAMP_ENABLE);
    snd_pcm_sw_params_set_tstamp_type(pcm_handle, swparams, SND_PCM_TSTAMP_TYPE_GETTIMEOFDAY);
    snd_pcm_sw_params(pcm_handle, swparams);

    playback_thread_handle = pthread_create(&playback_thread, 0, (void *(*)(void *))_snd_pcm_writei, 0);
#endif /* UNIX */

    return PyLong_FromLong(0);
}

static PyObject *sointu_playback_position(PyObject *self, PyObject *args) {
#ifdef WIN32
    DWORD play_cursor = 0;
    IDirectSoundBuffer_GetCurrentPosition(direct_sound_buffer, (DWORD*)&play_cursor, NULL);
    return Py_BuildValue("i", play_cursor / SU_CHANNEL_COUNT / sizeof(SUsample));
#endif /* WIN32 */

#ifdef UNIX
    snd_htimestamp_t ts;
    snd_pcm_uframes_t avail;
    err = snd_pcm_htimestamp(pcm_handle, &avail, &ts);

    // TODO: return the correct timestamp
#endif /* UNIX */
}

static PyObject *sointu_playback_finished(PyObject *self, PyObject *args) {
    bool result;

#ifdef WIN32
    DWORD play_cursor = 0;
    IDirectSoundBuffer_GetCurrentPosition(direct_sound_buffer, (DWORD*)&play_cursor, NULL);
    result = play_cursor < last_play_cursor;
    last_play_cursor = play_cursor;
#endif /* WIN32 */

#ifdef UNIX
    // TODO: Return the correct check.
#endif /* UNIX */

    return PyBool_FromLong(result);
}

static PyObject *sointu_sample_rate(PyObject *self, PyObject *args) {
    return Py_BuildValue("i", SU_SAMPLE_RATE);
}

static PyObject *sointu_track_length(PyObject *self, PyObject *args) {
    return Py_BuildValue("i", SU_LENGTH_IN_SAMPLES);
}

static PyMethodDef sointuMethods[] = {
    {"play_song", sointu_play_song, METH_VARARGS, "Play sointu track."},
    {"playback_position", sointu_playback_position, METH_VARARGS, "Get playback position of sointu track currently playing."},
    {"playback_finished", sointu_playback_finished, METH_VARARGS, "Check if currently playing sointu track has finished playing."},
    {"sample_rate", sointu_sample_rate, METH_VARARGS, "Return the sample rate of the track compiled into this module."},
    {"track_length", sointu_track_length, METH_VARARGS, "Return the track length in samples."},
    {NULL, NULL, 0, NULL} /* Sentinel */
};

static struct PyModuleDef sointumodule = {
    PyModuleDef_HEAD_INIT,
    "sointu",
    NULL,
    -1,
    sointuMethods
};

PyMODINIT_FUNC PyInit_sointu(void) {
    PyObject *module = PyModule_Create(&sointumodule);
    if(module == NULL) {
        return NULL;
    }

    sointuError = PyErr_NewException("sointu.sointuError", NULL, NULL);
    Py_XINCREF(sointuError);
    
    if(PyModule_AddObject(module, "error", sointuError) < 0) {
        Py_XDECREF(sointuError);
        Py_CLEAR(sointuError);
        Py_DECREF(module);
        return NULL;
    }

    return module;
}

int main(int argc, char *argv[])
{
    wchar_t *program = Py_DecodeLocale(argv[0], NULL);
    if (program == NULL) {
        fprintf(stderr, "Fatal error: cannot decode argv[0]\n");
        exit(1);
    }

    /* Add a built-in module, before Py_Initialize */
    if (PyImport_AppendInittab("sointu", PyInit_sointu) == -1) {
        fprintf(stderr, "Error: could not extend in-built modules table\n");
        exit(1);
    }

    /* Pass argv[0] to the Python interpreter */
    Py_SetProgramName(program);

    /* Initialize the Python interpreter.  Required.
       If this step fails, it will be a fatal error. */
    Py_Initialize();

    /* Optionally import the module; alternatively,
       import can be deferred until the embedded script
       imports it. */
    PyObject *pmodule = PyImport_ImportModule("sointu");
    if (!pmodule) {
        PyErr_Print();
        fprintf(stderr, "Error: could not import module 'sointu'\n");
    }

    PyMem_RawFree(program);
    return 0;
}
