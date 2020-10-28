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
    .synthwrk   resb    su_synthworkspace.size
    .delaywrks  resb    su_delayline_wrk.size * 64    
    .randseed   resd    1
    .globaltime resd    1       
endstruc

struc su_synth
    .commands   resb    32 * 64
    .values     resb    32 * 64 * 8
    .polyphony  resd    1
    .numvoices  resd    1
endstruc

SECT_TEXT(sursampl)

EXPORT MANGLE_FUNC(su_render_time,20)
%if BITS == 32  ; stdcall    
    pushad                       ; push registers
    mov     eax, [esp + 4 + 32]  ; eax = &synth
    mov     ecx, [esp + 8 + 32]  ; ecx = &synthState 
    mov     edx, [esp + 12 + 32] ; edx = &buffer
    mov     esi, [esp + 16 + 32] ; esi = &samples
    mov     ebx, [esp + 20 + 32] ; ebx = &time
%else
    %ifidn __OUTPUT_FORMAT__,win64
        ; win64 ABI parameter order: RCX, RDX, R8, R9; rest in stack (right-to-left, note shadow space!)
        ; here: rcx = &synth, rdx = &synthstate, r8 = &buffer, r9 = &bufsize, stack1 = &time        
        push_registers rdi, rsi, rbx, rbp ; win64 ABI: these registers are non-volatile
        mov     rbx, [rsp + 0x28 + PUSH_REG_SIZE(4)]  ; rbx = &time, note the shadow space so &time is at rsp + 0x28
        mov     rax, rcx
        mov     rcx, rdx
        mov     rdx, r8
        mov     rsi, r9        
    %else
        ; System V ABI parameter order: RDI, RSI, RDX, RCX, R8, R9; rest in stack (right-to-left)
        ; here: rdi = &synth, rsi = &synthstate, rdx = &buffer, rcx = &samples, r8 = &time
        push_registers rbx, rbp ; System V ABI: these registers are non-volatile   
        mov     rax, rdi
        mov     rbx, r8
        xchg    rcx, rsi        
    %endif
%endif
    ; _AX = &synth, _BX == &time, _CX == &synthstate, _DX == &buf, _SI == &samples,  
    push    _AX         ; push the pointer to synth to stack
    push    _SI         ; push the pointer to samples
    push    _BX         ; push the pointer to time
    xor     eax, eax    ; samplenumber starts at 0
    push    _AX         ; push samplenumber to stack
    mov     esi, [_SI]  ; zero extend dereferenced pointer
    push    _SI         ; push bufsize
    push    _DX         ; push bufptr
    push    _CX         ; this takes place of the voicetrack
    mov     eax, [_CX + su_synth_state.randseed]
    push    _AX                             ; randseed
    mov     eax, [_CX + su_synth_state.globaltime]
    push    _AX                        ; global tick time    
    mov     ebx, dword [_BX]           ; zero extend dereferenced pointer
    push    _BX                        ; the nominal rowlength should be time_in
    xor     eax, eax                   ; rowtick starts at 0
su_render_samples_loop:
        cmp     eax, [_SP]                    ; if rowtick >= maxtime
        jge     su_render_samples_time_finish ;   goto finish
        mov     ecx, [_SP + PTRSIZE*5]        ; ecx = buffer length in samples
        cmp     [_SP + PTRSIZE*6], ecx        ; if samples >= maxsamples
        jge     su_render_samples_time_finish ;   goto finish
        inc     eax                           ; time++
        inc     dword [_SP + PTRSIZE*6]       ; samples++
        mov     _CX, [_SP + PTRSIZE*3]        ; _CX = &synthstate
        mov     _BP, [_SP + PTRSIZE*9]        ; _BP = &synth
        push    _AX                        ; push rowtick
        mov     eax, [_BP + su_synth.polyphony]
        push    _AX                        ;polyphony
        mov     eax, [_BP + su_synth.numvoices]
        push    _AX                        ;numvoices
        lea     _DX, [_CX + su_synth_state.synthwrk] 
        lea     COM, [_BP + su_synth.commands] 
        lea     VAL, [_BP + su_synth.values] 
        lea     WRK, [_DX + su_synthworkspace.voices]  ; _BP and WRK are actually the same thing so here _BP gets overwritten
        lea     _CX, [_CX + su_synth_state.delaywrks - su_delayline_wrk.filtstate] 
        call    MANGLE_FUNC(su_run_vm,0)
        pop     _AX
        pop     _AX
        mov     _DI, [_SP + PTRSIZE*5] ; edi containts buffer ptr
        mov     _CX, [_SP + PTRSIZE*4]
        lea     _SI, [_CX + su_synth_state.synthwrk + su_synthworkspace.left]
        movsd   ; copy left channel to output buffer
        movsd   ; copy right channel to output buffer
        mov     [_SP + PTRSIZE*5], _DI ; save back the updated ptr
        lea     _DI, [_SI-8]
        xor     eax, eax
        stosd   ; clear left channel so the VM is ready to write them again
        stosd   ; clear right channel so the VM is ready to write them again
        pop     _AX
        inc     dword [_SP + PTRSIZE] ; increment global time, used by delays
        jmp     su_render_samples_loop
su_render_samples_time_finish:
    pop     _CX
    pop     _BX
    pop     _DX    
    pop     _CX
    mov     [_CX + su_synth_state.randseed], edx
    mov     [_CX + su_synth_state.globaltime], ebx            
    pop     _BX
    pop     _BX
    pop     _DX
    pop     _BX  ; pop the pointer to time
    pop     _SI  ; pop the pointer to samples
    mov     dword [_SI], edx  ; *samples = samples rendered
    mov     dword [_BX], eax  ; *time = time ticks rendered
    pop     _AX  ; pop the synth pointer
    xor     eax, eax ; TODO: set eax to possible error code, now just 0
%if BITS == 32  ; stdcall
    mov     [_SP + 28],eax ; we want to return eax, but popad pops everything, so put eax to stack for popad to pop 
    popad
    ret 20
%else
    %ifidn __OUTPUT_FORMAT__,win64
        pop_registers rdi, rsi, rbx, rbp ; win64 ABI: these registers are non-volatile
    %else
        pop_registers rbx, rbp ; System V ABI: these registers are non-volatile   
    %endif
    ret
%endif

EXPORT MANGLE_FUNC(su_render,16)
%if BITS == 32  ; stdcall        
    mov     eax, 0x7FFFFFFF ; don't care about time, just try to fill the buffer
    push    eax
    push    esp
    lea     eax, [esp + 24]
    push    eax    
    push    dword [esp + 24]
    push    dword [esp + 24]
    push    dword [esp + 24]    
%else
    %ifidn __OUTPUT_FORMAT__,win64
        ; win64 ABI parameter order: RCX, RDX, R8, R9; rest in stack (right-to-left, note shadow space!)
        ; here: rcx = &synth, rdx = &synthstate, r8 = &buffer, r9 = samples        
        ; put the values in shadow space and get pointers to them
        mov     [_SP+0x8],r9
        lea     r9, [_SP+0x8]
        mov     [_SP+0x10], dword 0x7FFFFFFF ; don't care about time, just try to fill the buffer
        lea     r10, [_SP+0x10]
        mov     [_SP+0x20], r10
    %else
        ; System V ABI parameter order: RDI, RSI, RDX, RCX, R8, R9; rest in stack (right-to-left)
        ; here: rdi = &synth, rsi = &synthstate, rdx = &buffer, rcx = samples
        push    rcx
        mov     rcx, _SP        ; pass a pointer to samples instead of direct value
        mov     r8, 0x7FFFFFFF ; don't care about time, just try to fill the buffer
        push    r8
        mov     r8, _SP        ; still, we have to pass a pointer to time, so pointer to stack        
    %endif
%endif    
    call    MANGLE_FUNC(su_render_time,20)        
%if BITS == 32  ; stdcall    
    pop     ecx
    ret     16
%else
    %ifnidn __OUTPUT_FORMAT__,win64 ; win64 ABI: rdx = bufsize, r8 = &buffer, rcx = &synthstate
        pop     rcx
        pop     r8
    %endif
    ret
%endif  
