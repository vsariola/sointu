%include TRACK_INCLUDE

%define SND_PCM_FORMAT_S16_LE 0x2
%define SND_PCM_FORMAT_FLOAT 0xE
%define SND_PCM_ACCESS_RW_INTERLEAVED 0x3
%define SND_PCM_STREAM_PLAYBACK 0x0

section .bss
sound_buffer:
	resb SU_LENGTH_IN_SAMPLES * SU_SAMPLE_SIZE * SU_CHANNEL_COUNT

render_thread:
	resq 1

pcm_handle:
	resq 1

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
	; Prologue
	push	rbp
	mov	 rbp, rsp
	sub	 rsp, 0x10

	; Unix does not have gm.dls, no need to ifdef and setup here.

	; We render in the background while playing already.
	mov rcx, sound_buffer
	lea rdx, su_render_song
	mov rsi, 0x0
	mov rdi, render_thread
	call pthread_create

	; We can't start playing too early or the missing samples will be audible.
	mov edi, 0x2
	call sleep

	; Play the track.
	mov rdi, pcm_handle
	mov rsi, default_device
	mov rdx, SND_PCM_STREAM_PLAYBACK
	mov rcx, 0x0
	call snd_pcm_open

	; This is unfortunate. amd64 ABI calling convention kicks in.
	; now we have to maintain the stack pointer :/
	mov rdi, qword [pcm_handle]
	sub rsp, 0x8
	push SU_LENGTH_IN_SAMPLES
%ifdef SU_SAMPLE_FLOAT
	mov rsi, SND_PCM_FORMAT_FLOAT
%else ; SU_SAMPLE_FLOAT
	mov rsi, SND_PCM_FORMAT_S16_LE
%endif ; SU_SAMPLE_FLOAT
	mov rdx, SND_PCM_ACCESS_RW_INTERLEAVED
	mov rcx, SU_CHANNEL_COUNT
	mov r8d, SU_SAMPLE_RATE
	mov r9d, 0x0
	call snd_pcm_set_params

	mov rdi, qword [pcm_handle]
	mov rsi, sound_buffer
	mov rdx, SU_LENGTH_IN_SAMPLES
	call snd_pcm_writei

exit:
	; At least we can skip the epilogue :)
	leave
	ret
