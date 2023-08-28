%define MANGLED
%include TRACK_INCLUDE

%define WAVE_FORMAT_PCM 0x1
%define WAVE_FORMAT_IEEE_FLOAT 0x3
%define WHDR_PREPARED 0x2
%define WAVE_MAPPER 0xFFFFFFFF
%define TIME_SAMPLES 0x2
%define PM_REMOVE 0x1

section .bss
sound_buffer:
	resb SU_LENGTH_IN_SAMPLES * SU_SAMPLE_SIZE * SU_CHANNEL_COUNT

wave_out_handle:
	resd 1

msg:
	resd 1
message:
	resd 7

section .data
wave_format:
%ifdef SU_SAMPLE_FLOAT
	dw WAVE_FORMAT_IEEE_FLOAT
%else ; SU_SAMPLE_FLOAT
	dw WAVE_FORMAT_PCM
%endif ; SU_SAMPLE_FLOAT
	dw SU_CHANNEL_COUNT
	dd SU_SAMPLE_RATE 
	dd SU_SAMPLE_SIZE * SU_SAMPLE_RATE * SU_CHANNEL_COUNT
	dw SU_SAMPLE_SIZE * SU_CHANNEL_COUNT
	dw SU_SAMPLE_SIZE * 8
	dw 0

wave_header:
	dd sound_buffer
	dd SU_LENGTH_IN_SAMPLES * SU_SAMPLE_SIZE * SU_CHANNEL_COUNT
	times 2 dd 0
	dd WHDR_PREPARED
	times 4 dd 0
wave_header_end:

mmtime:
	dd TIME_SAMPLES
sample:
	times 2 dd 0
mmtime_end:

section .text
symbols:
	extern _CreateThread@24
	extern _waveOutOpen@24
	extern _waveOutWrite@12
	extern _waveOutGetPosition@12
	extern _PeekMessageA@20
	extern _TranslateMessage@4
	extern _DispatchMessageA@4

	global _mainCRTStartup
_mainCRTStartup:
	; win32 uses the cdecl calling convention. This is more readable imo ;)
	; We can also skip the prologue; Windows doesn't mind.

%ifdef SU_LOAD_GMDLS
	call _su_load_gmdls
%endif ; SU_LOAD_GMDLS

	times 2 push 0
	push sound_buffer
	lea eax, _su_render_song@4
	push eax
	times 2 push 0
	call _CreateThread@24

	; We render in the background while playing already. Fortunately,
	; Windows is slow with the calls below, so we're not worried that
	; we don't have enough samples ready before the track starts.
	times 3 push 0
	push wave_format
	push WAVE_MAPPER
	push wave_out_handle
	call _waveOutOpen@24

	push wave_header_end - wave_header
	push wave_header
	push dword [wave_out_handle]
	call _waveOutWrite@12

	; We need to handle windows messages properly while playing, as waveOutWrite is async.
mainloop:
	dispatchloop:
		push PM_REMOVE
		times 3 push 0
		push msg
		call _PeekMessageA@20
		jz dispatchloop_end

		push msg
		call _TranslateMessage@4

		push msg
		call _DispatchMessageA@4

		jmp dispatchloop
	dispatchloop_end:

	push mmtime_end - mmtime
	push mmtime
	push dword [wave_out_handle]
	call _waveOutGetPosition@12

	cmp dword [sample], SU_LENGTH_IN_SAMPLES
	jne mainloop

exit:
	; At least we can skip the epilogue :)
	leave
	ret
