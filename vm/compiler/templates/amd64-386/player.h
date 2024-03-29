// auto-generated by Sointu, editing not recommended
#ifndef SU_RENDER_H
#define SU_RENDER_H

#define SU_CHANNEL_COUNT        2
#define SU_LENGTH_IN_SAMPLES    {{.MaxSamples}}
#define SU_BUFFER_LENGTH        (SU_LENGTH_IN_SAMPLES*SU_CHANNEL_COUNT)

#define SU_SAMPLE_RATE          44100
#define SU_BPM                  {{.Song.BPM}}
#define SU_ROWS_PER_BEAT        {{.Song.RowsPerBeat}}
#define SU_ROWS_PER_PATTERN     {{.Song.Score.RowsPerPattern}}
#define SU_LENGTH_IN_PATTERNS   {{.Song.Score.Length}}
#define SU_LENGTH_IN_ROWS       (SU_LENGTH_IN_PATTERNS*SU_PATTERN_SIZE)
#define SU_SAMPLES_PER_ROW      (SU_SAMPLE_RATE*60/(SU_BPM*SU_ROWS_PER_BEAT))

{{- if or .RowSync (.HasOp "sync")}}
{{- if .RowSync}}
#define SU_NUMSYNCS             {{add1 .Song.Patch.NumSyncs}}
{{- else}}
#define SU_NUMSYNCS             {{.Song.Patch.NumSyncs}}
{{- end}}
#define SU_SYNCBUFFER_LENGTH    ((SU_LENGTH_IN_SAMPLES+255)>>8)*SU_NUMSYNCS
{{- end}}

#include <stdint.h>
#if UINTPTR_MAX == 0xffffffff
	#if defined(__clang__) || defined(__GNUC__)
		#define SU_CALLCONV __attribute__ ((stdcall))
	#elif defined(_WIN32)
		#define SU_CALLCONV __stdcall
	#endif
#else
	#define SU_CALLCONV
#endif

{{- if .Output16Bit}}
typedef short SUsample;
#define SU_SAMPLE_RANGE 32767.0
#define SU_SAMPLE_PCM16
#define SU_SAMPLE_SIZE 2
{{- else}}
typedef float SUsample;
#define SU_SAMPLE_RANGE 1.0
#define SU_SAMPLE_FLOAT
#define SU_SAMPLE_SIZE 4
{{- end}}


#ifdef __cplusplus
extern "C" {
#endif

{{- if or .RowSync (.HasOp "sync")}}
#define SU_SYNC
{{- end}}
void SU_CALLCONV su_render_song(SUsample *buffer);

{{- if gt (.SampleOffsets | len) 0}}
void SU_CALLCONV su_load_gmdls();
#define SU_LOAD_GMDLS
{{- end}}


#ifdef __cplusplus
}
#endif

#endif