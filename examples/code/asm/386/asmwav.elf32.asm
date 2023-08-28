%include TRACK_INCLUDE

%define WAVE_FORMAT_PCM 0x1
%define WAVE_FORMAT_IEEE_FLOAT 0x3

section .bss
sound_buffer:
	resb SU_LENGTH_IN_SAMPLES * SU_SAMPLE_SIZE * SU_CHANNEL_COUNT

file:
	resd 1

section .data
; Change the filename over -DFILENAME="yourfilename.wav"
filename:
	db FILENAME, 0

format:
	db "wb", 0

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
	extern fopen
	extern fwrite
	extern fclose

	global main
main:
	; elf32 uses the cdecl calling convention. This is more readable imo ;)

	; Prologue
	push	ebp
	mov	 ebp, esp
	sub	 esp, 0x10

	; Unix does not have gm.dls, no need to ifdef and setup here.

	; We render the complete track here.
	push sound_buffer
	call su_render_song

	; Now we open the file and save the track.
	push format
	push filename
	call fopen
	mov dword [file], eax

	; Write header
	push dword [file]
	push 0x1
	push wave_file_end - wave_file
	push wave_file
	call fwrite

	; write data
	push dword [file]
	push 0x1
	push SU_LENGTH_IN_SAMPLES * SU_SAMPLE_SIZE * SU_CHANNEL_COUNT
	push sound_buffer
	call fwrite

	push dword [file]
	call fclose

exit:
	; At least we can skip the epilogue :)
	leave
	ret
