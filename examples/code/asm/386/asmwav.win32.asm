%define MANGLED
%include TRACK_INCLUDE

%define WAVE_FORMAT_PCM 0x1
%define WAVE_FORMAT_IEEE_FLOAT 0x3
%define FILE_ATTRIBUTE_NORMAL 0x00000080
%define CREATE_ALWAYS 2
%define GENERIC_WRITE 0x40000000

section .bss
sound_buffer:
	resb SU_LENGTH_IN_SAMPLES * SU_SAMPLE_SIZE * SU_CHANNEL_COUNT

file:
	resd 1

bytes_written:
	resd 1

section .data
; Change the filename over -DFILENAME="yourfilename.wav"
filename:
	db FILENAME, 0

; This is the wave file header.
wave_file:
	db "RIFF"
	dd wave_file_end + SU_LENGTH_IN_SAMPLES * SU_SAMPLE_SIZE * SU_CHANNEL_COUNT - wave_file
	db "WAVE"
	db "fmt "
wave_format_end:
	dd wave_format_end - wave_file
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
wave_header_end:
	db "data"
	dd wave_file_end + SU_LENGTH_IN_SAMPLES * SU_SAMPLE_SIZE * SU_CHANNEL_COUNT - wave_header_end
wave_file_end:

section .text
symbols:
	extern _CreateFileA@28
	extern _WriteFile@20
	extern _CloseHandle@4

	global _mainCRTStartup
_mainCRTStartup:
	; Prologue
	push	ebp
	mov	 ebp, esp
	sub	 esp, 0x10

%ifdef SU_LOAD_GMDLS
	call _su_load_gmdls
%endif ; SU_LOAD_GMDLS

	; We render the complete track here.
	push sound_buffer
	call _su_render_song@4

	; Now we open the file and save the track.
	push 0x0
	push FILE_ATTRIBUTE_NORMAL
	push CREATE_ALWAYS
	push 0x0
	push 0x0
	push GENERIC_WRITE
	push filename
	call _CreateFileA@28
	mov dword [file], eax

	; This is the WAV header
	push 0x0
	push bytes_written
	push wave_file_end - wave_file
	push wave_file
	push dword [file]
	call _WriteFile@20
	
	; There we write the actual samples
	push 0x0
	push bytes_written
	push SU_LENGTH_IN_SAMPLES * SU_CHANNEL_COUNT * SU_SAMPLE_SIZE
	push sound_buffer
	push dword [file]
	call _WriteFile@20
	
	push dword [file]
	call _CloseHandle@4

exit:
	; At least we can skip the epilogue :)
	leave
	ret
