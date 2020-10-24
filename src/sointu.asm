; source file for compiling sointu as a library
%define SU_DISABLE_PLAYER

%include "sointu_header.inc"

; TODO: make sure compile everything in

USE_ENVELOPE
USE_OSCILLAT
USE_MULP
USE_PAN
USE_OUT

%define INCLUDE_TRISAW
%define INCLUDE_SINE
%define INCLUDE_PULSE
%define INCLUDE_GATE
%define INCLUDE_STEREO_OSCILLAT
%define INCLUDE_STEREO_ENVELOPE
%define INCLUDE_STEREO_OUT
%define INCLUDE_POLYPHONY
%define INCLUDE_MULTIVOICE_TRACKS

%include "sointu_footer.inc"

section .text

struc su_synth_state
    .synth      resb    su_synth.size
    .delaywrks  resb    su_delayline_wrk.size * 64
    .commands   resb    32 * 64
    .values     resb    32 * 64 * 8
    .polyphony  resd    1
    .numvoices  resd    1
    .randseed   resd    1
    .globaltime resd    1   
    .rowtick    resd    1
    .rowlen     resd    1
endstruc

SECT_TEXT(sursampl)

EXPORT MANGLE_FUNC(su_render_samples,12)
%if BITS == 32  ; stdcall    
    pushad                  ; push registers
    mov     ecx, [esp + 4 + 32] ; ecx = &synthState
    mov     esi, [esp + 8 + 32]  ; esi = bufsize
    mov     edx, [esp + 12 + 32]  ; edx = &buffer 
%else
    %ifidn __OUTPUT_FORMAT__,win64 ; win64 ABI: rdx = bufsize, r8 = &buffer, rcx = &synthstate
        push_registers rdi, rsi, rbx, rbp ; win64 ABI: these registers are non-volatile
        mov     rsi, rdx ; rsi = bufsize
        mov     rdx, r8 ; rdx = &buffer
    %else ; System V ABI: rsi = bufsize, rdx = &buffer, rdi = &synthstate
        push_registers rbx, rbp ; System V ABI: these registers are non-volatile   
        mov     rcx, rdi ; rcx = &Synthstate
    %endif
%endif
    push    _SI  ; push bufsize
    push    _DX  ; push bufptr
    push    _CX  ; this takes place of the voicetrack
    mov     eax, [_CX + su_synth_state.randseed]
    push    _AX                             ; randseed
    mov     eax, [_CX + su_synth_state.globaltime]
    push    _AX                        ; global tick time
    mov     eax, [_CX + su_synth_state.rowlen] 
    push    _AX                        ; push the rowlength to stack so we can easily compare to it, normally this would be row
    mov     eax, [_CX + su_synth_state.rowtick]
su_render_samples_loop:
        cmp     eax, [_SP] ; compare current tick to rowlength
        jge     su_render_samples_row_advance
        sub     dword [_SP + PTRSIZE*5], 1 ; compare current tick to rowlength
        jb      su_render_samples_buffer_full
        mov     _CX, [_SP + PTRSIZE*3]
        push    _AX                        ; push rowtick
        mov     eax, [_CX + su_synth_state.polyphony]
        push    _AX                        ;polyphony
        mov     eax, [_CX + su_synth_state.numvoices]
        push    _AX                        ;numvoices
        lea     _DX, [_CX+ su_synth_state.synth] 
        lea     COM, [_CX+ su_synth_state.commands] 
        lea     VAL, [_CX+ su_synth_state.values] 
        lea     WRK, [_DX + su_synth.voices]  
        lea     _CX, [_CX+ su_synth_state.delaywrks - su_delayline_wrk.filtstate] 
        call    MANGLE_FUNC(su_run_vm,0)
        pop     _AX
        pop     _AX
        mov     _DI, [_SP + PTRSIZE*5] ; edi containts buffer ptr
        mov     _CX, [_SP + PTRSIZE*4]
        lea     _SI, [_CX + su_synth_state.synth + su_synth.left]
        movsd   ; copy left channel to output buffer
        movsd   ; copy right channel to output buffer
        mov     [_SP + PTRSIZE*5], _DI ; save back the updated ptr
        lea     _DI, [_SI-8]
        xor     eax, eax
        stosd   ; clear left channel so the VM is ready to write them again
        stosd   ; clear right channel so the VM is ready to write them again
        pop     _AX
        inc     dword [_SP + PTRSIZE] ; increment global time, used by delays
        inc     eax
        jmp     su_render_samples_loop
su_render_samples_row_advance:
    xor     eax, eax ; row has finished, so clear the rowtick for next round
su_render_samples_buffer_full:
    pop     _CX
    pop     _BX
    pop     _DX    
    pop     _CX
    mov     [_CX + su_synth_state.randseed], edx
    mov     [_CX + su_synth_state.globaltime], ebx        
    mov     [_CX + su_synth_state.rowtick], eax
    pop     _AX
    pop     _AX
%if BITS == 32  ; stdcall
    mov     [_SP + 28],eax ; we want to return eax, but popad pops everything, so put eax to stack for popad to pop 
    popad
    ret 12
%else
    %ifidn __OUTPUT_FORMAT__,win64
        pop_registers rdi, rsi, rbx, rbp ; win64 ABI: these registers are non-volatile
    %else
        pop_registers rbx, rbp ; System V ABI: these registers are non-volatile   
    %endif
    ret
%endif
