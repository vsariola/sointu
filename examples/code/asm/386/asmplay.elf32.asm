%include TRACK_INCLUDE

%define SND_PCM_FORMAT_S16_LE 0x2
%define SND_PCM_FORMAT_FLOAT 0xE
%define SND_PCM_ACCESS_RW_INTERLEAVED 0x3
%define SND_PCM_STREAM_PLAYBACK 0x0

section .bss
sound_buffer:
	resb SU_LENGTH_IN_SAMPLES * SU_SAMPLE_SIZE * SU_CHANNEL_COUNT

render_thread:
	resd 1

pcm_handle:
	resd 1

section .data
default_device:
	db "default", 0

section .text
symbols:
	extern pthread_create
	extern sleep
	extern snd_pcm_open
	extern snd_pcm_set_params
	extern snd_pcm_writei

	global main
main:
	; elf32 uses the cdecl calling convention. This is more readable imo ;)

	; Prologue
	push	ebp
	mov	 ebp, esp
	sub	 esp, 0x10

	; Unix does not have gm.dls, no need to ifdef and setup here.

	; We render in the background while playing already.
	push sound_buffer
	lea eax, su_render_song
	push eax
	push 0
	push render_thread
	call pthread_create

	; We can't start playing too early or the missing samples will be audible.
	push 0x2
	call sleep

	; Play the track.
	push 0x0
	push SND_PCM_STREAM_PLAYBACK
	push default_device
	push pcm_handle
	call snd_pcm_open

	push SU_LENGTH_IN_SAMPLES
	push 0
	push SU_SAMPLE_RATE
	push SU_CHANNEL_COUNT
	push SND_PCM_ACCESS_RW_INTERLEAVED
%ifdef SU_SAMPLE_FLOAT
	push SND_PCM_FORMAT_FLOAT
%else ; SU_SAMPLE_FLOAT
	push SND_PCM_FORMAT_S16_LE
%endif ; SU_SAMPLE_FLOAT
	push dword [pcm_handle]
	call snd_pcm_set_params

	push SU_LENGTH_IN_SAMPLES
	push sound_buffer
	push dword [pcm_handle]
	call snd_pcm_writei

exit:
	; At least we can skip the epilogue :)
	leave
	ret
