%if BITS == 64
    %define WRK rbp ; alias for unit workspace
    %define VAL rsi ; alias for unit values (transformed/untransformed)
    %define COM rbx ; alias for instrument opcodes
    %define INP rdx ; alias for transformed inputs
    %define _AX rax ; push and offsets have to be r* on 64-bit and e* on 32-bit
    %define _BX rbx
    %define _CX rcx
    %define _DX rdx
    %define _SP rsp
    %define _SI rsi
    %define _DI rdi
    %define _BP rbp
    %define PTRSIZE 8
    %define PTRWORD qword
    %define RESPTR resq
    %define DPTR dq

    %macro do 2
        mov r9, qword %2
        %1 r9
    %endmacro

    %macro do 3
        mov r9, qword %2
        %1 r9 %3
    %endmacro

    %macro do 4
        mov r9, qword %2
        %1 r9+%3 %4
    %endmacro

    %macro do 5
        mov r9, qword %2
        lea r9, [r9+%3]
        %1 r9+%4 %5
    %endmacro

    %macro  push_registers 1-*
        %rep  %0
            push    %1
            %rotate 1
        %endrep
    %endmacro

    %macro  pop_registers 1-*
        %rep %0
            %rotate -1
            pop     %1
        %endrep
    %endmacro

    %define PUSH_REG_SIZE(n) (n*8)

    %ifidn __OUTPUT_FORMAT__,win64
        %define render_prologue push_registers rcx,rdi,rsi,rbx,rbp  ; rcx = ptr to buf. rdi,rsi,rbx,rbp  nonvolatile
        %macro render_epilogue 0
            pop_registers rcx,rdi,rsi,rbx,rbp
            ret
        %endmacro
    %else ; 64 bit mac & linux
        %define render_prologue push_registers rdi,rbx,rbp ; rdi = ptr to buf. rbx & rbp nonvolatile
        %macro render_epilogue 0
            pop_registers rdi,rbx,rbp
            ret
        %endmacro
    %endif
%else
    %define WRK ebp ; alias for unit workspace
    %define VAL esi ; alias for unit values (transformed/untransformed)
    %define COM ebx ; alias for instrument opcodes
    %define INP edx ; alias for transformed inputs
    %define _AX eax
    %define _BX ebx
    %define _CX ecx
    %define _DX edx
    %define _SP esp
    %define _SI esi
    %define _DI edi
    %define _BP ebp
    %define PTRSIZE 4
    %define PTRWORD dword
    %define RESPTR resd
    %define DPTR dd

    %macro do 2
        %1 %2
    %endmacro

    %macro do 3
        %1 %2 %3
    %endmacro

    %macro do 4
        %1 %2+%3 %4
    %endmacro

    %macro do 5
        %1 %2+%3+%4 %5
    %endmacro

    %macro  push_registers 1-*
        pushad ; in 32-bit mode, this is the easiest way to store all the registers
    %endmacro

    %macro  pop_registers 1-*
        popad
    %endmacro

    %define PUSH_REG_SIZE(n) 32

    %define render_prologue pushad ; stdcall & everything nonvolatile except eax, ecx, edx

    %macro render_epilogue 0
        popad
        ret     4 ; clean the passed parameter from stack.
    %endmacro
%endif

section .text ; yasm throws section redeclaration warnings if strucs are defined without a plain .text section

struc su_stack ; the structure of stack _as the units see it_
    .retaddr    RESPTR  1
%if BITS == 32              ; we dump everything with pushad, so this is unused in 32-bit
                RESPTR  1
%endif
    .val        RESPTR  1
    .wrk        RESPTR  1
%if BITS == 32              ; we dump everything with pushad, so this is unused in 32-bit
                RESPTR  1
%endif
    .com        RESPTR  1
    .synth      RESPTR  1
    .delaywrk   RESPTR  1
%if BITS == 32              ; we dump everything with pushad, so this is unused in 32-bit
                RESPTR  1
%endif
    .retaddrvm  RESPTR  1
    .voiceno    RESPTR  1
%ifdef INCLUDE_POLYPHONY
    .polyphony  RESPTR  1
%endif
    .output_sound:
    .rowtick    RESPTR  1    ; which tick within this row are we at
    .update_voices:
    .row        RESPTR  1    ; which total row of the song are we at
    .tick       RESPTR  1    ; which total tick of the song are we at
    .randseed   RESPTR  1
%ifdef RUNTIME_TABLES        ; RUNTIME_TABLES move these table ptrs to stack
    %if DELAY_ID > -1        ; allowing them to be in the heap &
    .delaytimes RESPTR 1     ; modifying them safely & having many synths
    %endif
    %ifdef INCLUDE_SAMPLES
    .sampleoffs RESPTR 1
    %endif
%endif
%ifdef INCLUDE_MULTIVOICE_TRACKS
    .voicetrack RESPTR  1
%endif
    .render_epilogue:
%if BITS == 32
                RESPTR  8   ; registers
    .retaddr_pl RESPTR  1
%elifidn __OUTPUT_FORMAT__,win64
                RESPTR  4   ; registers
%else
                RESPTR  2   ; registers
%endif
    .bufferptr  RESPTR  1
    .size:
endstruc

;===============================================================================
;   Uninitialized data: The one and only synth object
;===============================================================================
SECT_BSS(susynth)

su_synth_obj            resb    su_synthworkspace.size

%if DELAY_ID > -1       ; if we use delay, then the synth obj should be immediately followed by the delay workspaces
                        resb   NUM_DELAY_LINES*su_delayline_wrk.size
%endif

;===============================================================================
; The opcode table jump table. This is constructed to only include the opcodes
; that are used so that the jump table is as small as possible.
;===============================================================================
SECT_DATA(suoptabl)

su_synth_commands       DPTR    OPCODES

;===============================================================================
; The number of transformed parameters each opcode takes
;===============================================================================
SECT_DATA(suparcnt)

su_opcode_numparams     db      NUMPARAMS

;-------------------------------------------------------------------------------
;   su_run_vm function: runs the entire virtual machine once, creating 1 sample
;-------------------------------------------------------------------------------
;   Input:      su_synth_obj.left   :   Set to 0 before calling
;               su_synth_obj.right  :   Set to 0 before calling
;               _CX                 :   Pointer to delay workspace (if needed)
;               _DX                 :   Pointer to synth object
;               COM                 :   Pointer to command stream
;               VAL                 :   Pointer to value stream
;               WRK                 :   Pointer to the last workspace processed
;               _DI                 :   Number of voices to process
;   Output:     su_synth_obj.left   :   left sample
;               su_synth_obj.right  :   right sample
;   Dirty:      everything
;-------------------------------------------------------------------------------
SECT_TEXT(surunvm)

EXPORT MANGLE_FUNC(su_run_vm,0)
    push_registers _CX, _DX, COM, WRK, VAL          ; save everything to stack
su_run_vm_loop:                                     ; loop until all voices done
    movzx   edi, byte [COM]                         ; edi = command byte
    inc     COM                                     ; move to next instruction
    add     WRK, su_unit.size                       ; move WRK to next unit
    shr     edi, 1                                  ; shift out the LSB bit = stereo bit
    mov     INP, [_SP+su_stack.wrk-PTRSIZE]         ; reset INP to point to the inputs part of voice
    add     INP, su_voice.inputs
    xor     ecx, ecx                                ; counter = 0
    xor     eax, eax                                ; clear out high bits of eax, as lodsb only sets al
su_transform_values_loop:
 do{cmp     cl, byte [},su_opcode_numparams,_DI,]   ; compare the counter to the value in the param count table
    je      su_transform_values_out
    lodsb                                           ; load the byte value from VAL stream
    push    _AX                                     ; push it to memory so FPU can read it
    fild    dword [_SP]                             ; load the value to FPU stack
 do fmul    dword [,c_i128,]                        ; divide it by 128 (0 => 0, 128 => 1.0)
    fadd    dword [WRK+su_unit.ports+_CX*4]         ; add the modulations in the current workspace
    fstp    dword [INP+_CX*4]                       ; store the modulated value in the inputs section of voice
    xor     eax, eax
    mov     dword [WRK+su_unit.ports+_CX*4], eax    ; clear out the modulation ports
    pop     _AX
    inc     ecx
    jmp     su_transform_values_loop
su_transform_values_out:
    bt      dword [COM-1],0                         ; LSB of COM = stereo bit => carry
 do call    [,su_synth_commands,_DI*PTRSIZE,]       ; call the function corresponding to the instruction
    cmp     dword [_SP+su_stack.voiceno-PTRSIZE],0  ; do we have more voices to process?
    jne     su_run_vm_loop                          ;   if there's more voices to process, goto vm_loop
    pop_registers _CX, _DX, COM, WRK, VAL           ; pop everything from stack
    ret

;-------------------------------------------------------------------------------
;   su_nonlinear_map function: returns 2^(-24*x) of parameter number _AX
;-------------------------------------------------------------------------------
;   Input:      _AX     :   parameter number (e.g. for envelope: 0 = attac, 1 = decay...)
;               INP     :   pointer to transformed values
;   Output:     st0     :   2^(-24*x), where x is the parameter in the range 0-1
;-------------------------------------------------------------------------------
SECT_TEXT(supower)

%if ENVELOPE_ID > -1 || COMPRES_ID > -1
su_nonlinear_map:
    fld     dword [INP+_AX*4]   ; x, where x is the parameter in the range 0-1
 do fimul   dword [,c_24,]      ; 24*x
    fchs                        ; -24*x
    ; flow into Power function, which outputs 2^(-24*x)
%endif

;-------------------------------------------------------------------------------
;   su_power function: computes 2^x
;-------------------------------------------------------------------------------
;   Input:      st0     :   x
;   Output:     st0     :   2^x
;-------------------------------------------------------------------------------
EXPORT MANGLE_FUNC(su_power,0)
    fld1          ; 1 x
    fld st1       ; x 1 x
    fprem         ; mod(x,1) 1 x
    f2xm1         ; 2^mod(x,1)-1 1 x
    faddp st1,st0 ; 2^mod(x,1) x
    fscale        ; 2^mod(x,1)*2^trunc(x) x
                  ; Equal to:
                  ; 2^x x
    fstp st1      ; 2^x
    ret

%ifndef SU_DISABLE_PLAYER

;-------------------------------------------------------------------------------
;   output_sound macro: used by the render function to write sound to buffer
;-------------------------------------------------------------------------------
;   The macro contains the ifdef hell to handle 16bit output and clipping cases
;   to keep the main function more readable
;   Stack   :   sample row pushad output_ptr
;-------------------------------------------------------------------------------
%macro output_sound 0
    %ifndef SU_OUTPUT_16BIT
        %ifndef SU_CLIP_OUTPUT ; The modern way. No need to clip; OS can do it.
            mov     _DI, [_SP+su_stack.bufferptr - su_stack.output_sound] ; edi containts ptr
            mov     _SI, PTRWORD su_synth_obj + su_synthworkspace.left
            movsd   ; copy left channel to output buffer
            movsd   ; copy right channel to output buffer
            mov     [_SP+su_stack.bufferptr - su_stack.output_sound], _DI ; save back the updated ptr
            lea     _DI, [_SI-8]
            xor     eax, eax
            stosd   ; clear left channel so the VM is ready to write them again
            stosd   ; clear right channel so the VM is ready to write them again
        %else
            mov     _SI, qword [_SP+su_stack.bufferptr - su_stack.output_sound] ; esi points to the output buffer
            xor     _CX,_CX
            xor     eax,eax
            %%loop: ; loop over two channels, left & right
             do fld     dword [,su_synth_obj+su_synthworkspace.left,_CX*4,]
                call    su_clip
                fstp    dword [_SI]
             do mov     dword [,su_synth_obj+su_synthworkspace.left,_CX*4,{],eax} ; clear the sample so the VM is ready to write it
                add     _SI,4
                cmp     ecx,2
                jl      %%loop
            mov     dword [_SP+su_stack.bufferptr - su_stack.output_sound], _SI ; save esi back to stack
            %define SU_INCLUDE_CLIP
        %endif
    %else ; 16-bit output, always clipped. This is a bit legacy method.
        mov     _SI, [_SP+su_stack.bufferptr - su_stack.output_sound] ; esi points to the output buffer
        mov     _DI, PTRWORD su_synth_obj+su_synthworkspace.left
        mov     ecx, 2
        %%loop: ; loop over two channels, left & right
            fld     dword [_DI]
            call    su_clip
         do fmul    dword [,c_32767,]
            push    _AX
            fistp   dword [_SP]
            pop     _AX
            mov     word [_SI],ax   ; // store integer converted right sample
            xor     eax,eax
            stosd
            add     _SI,2
            loop    %%loop
        mov     [_SP+su_stack.bufferptr - su_stack.output_sound], _SI ; save esi back to stack
        %define USE_C_32767
        %define SU_INCLUDE_CLIP
    %endif
%endmacro

;-------------------------------------------------------------------------------
;   su_render_song function: the entry point for the synth
;-------------------------------------------------------------------------------
;   Has the signature su_render_song(void *ptr), where ptr is a pointer to
;   the output buffer. Renders the compile time hard-coded song to the buffer.
;   Stack:  output_ptr
;-------------------------------------------------------------------------------
SECT_TEXT(surensng)

EXPORT MANGLE_FUNC(su_render_song,PTRSIZE)   ; Stack: ptr
    render_prologue
    xor     eax, eax
%ifdef INCLUDE_MULTIVOICE_TRACKS
    push    VOICETRACK_BITMASK
%endif
%ifdef RUNTIME_TABLES ; runtime tables is actually only useful in the api use case, but it's nice it works also in the render case
    %ifdef INCLUDE_SAMPLES
 do push    ,MANGLE_DATA(su_sample_offsets) ; &su_sample_offsets is in the stack, instead of hard coded
    %endif
    %if DELAY_ID > -1
 do push    ,MANGLE_DATA(su_delay_times) ; &su_delay_times is in the stack, instead of hard coded
    %endif
%endif
    push    1                           ; randseed
    push    _AX                         ; global tick time
su_render_rowloop:                      ; loop through every row in the song
        push    _AX                     ; Stack: row pushad ptr
        call    su_update_voices        ; update instruments for the new row
        xor     eax, eax                ; ecx is the current sample within row
su_render_sampleloop:                   ; loop through every sample in the row
            push    _AX                 ; Stack: sample row pushad ptr
            %ifdef INCLUDE_POLYPHONY
                push    POLYPHONY_BITMASK ; does the next voice reuse the current opcodes?
            %endif
            push    MAX_VOICES
            mov     _DX, PTRWORD su_synth_obj                       ; _DX points to the synth object
            mov     COM, PTRWORD MANGLE_DATA(su_commands)           ; COM points to vm code
            mov     VAL, PTRWORD MANGLE_DATA(su_params)             ; VAL points to unit params
            %if DELAY_ID > -1
                lea     _CX, [_DX + su_synthworkspace.size - su_delayline_wrk.filtstate]
            %endif
            lea     WRK, [_DX + su_synthworkspace.voices]            ; WRK points to the first voice
            call    MANGLE_FUNC(su_run_vm,0) ; run through the VM code
            pop     _AX
            %ifdef INCLUDE_POLYPHONY
                pop     _AX
            %endif
            output_sound                ; *ptr++ = left, *ptr++ = right
            pop     _AX
            inc     dword [_SP + PTRSIZE] ; increment global time, used by delays
            inc     eax
            cmp     eax, SAMPLES_PER_ROW
            jl      su_render_sampleloop
        pop     _AX                     ; Stack: pushad ptr
        inc     eax
        cmp     eax, TOTAL_ROWS
        jl      su_render_rowloop
    ; rewind the stack. Use add when there's a lot to clean, repeated pops if only a little
%if su_stack.render_epilogue - su_stack.tick > 3*PTRSIZE
    add     _SP, su_stack.render_epilogue - su_stack.tick ; rewind the remaining tack
%else
    %rep    (su_stack.render_epilogue - su_stack.tick)/PTRSIZE 
        pop _AX ; the entropy of 3 x pop _AX is probably lower than 3 byte add
    %endrep     ; but this needs to be confirmed
%endif
    render_epilogue

;-------------------------------------------------------------------------------
;   su_update_voices function: polyphonic & chord implementation
;-------------------------------------------------------------------------------
;   Input:      eax     :   current row within song
;   Dirty:      pretty much everything
;-------------------------------------------------------------------------------
SECT_TEXT(suupdvce)

%ifdef INCLUDE_MULTIVOICE_TRACKS

su_update_voices: ; Stack: retaddr row
    xor     edx, edx
    mov     ebx, PATTERN_SIZE                   ; we could do xor ebx,ebx; mov bl,PATTERN_SIZE, but that would limit patternsize to 256...
    div     ebx                                 ; eax = current pattern, edx = current row in pattern
 do{lea     _SI, [},MANGLE_DATA(su_tracks),_AX,]  ; esi points to the pattern data for current track
    xor     eax, eax                            ; eax is the first voice of next track
    xor     ebx, ebx                            ; ebx is the first voice of current track
    mov     _BP, PTRWORD su_synth_obj           ; ebp points to the current_voiceno array
su_update_voices_trackloop:
        movzx   eax, byte [_SI]                     ; eax = current pattern
        imul    eax, PATTERN_SIZE                   ; eax = offset to current pattern data
     do{movzx   eax,byte [},MANGLE_DATA(su_patterns),_AX,_DX,]  ; eax = note
        push    _DX                                 ; Stack: ptrnrow
        xor     edx, edx                            ; edx=0
        mov     ecx, ebx                            ; ecx=first voice of the track to be done
su_calculate_voices_loop:                           ; do {
        bt      dword [_SP + su_stack.voicetrack - su_stack.update_voices + 2*PTRSIZE],ecx ; test voicetrack_bitmask// notice that the incs don't set carry
        inc     edx                                 ;   edx++   // edx=numvoices
        inc     ecx                                 ;   ecx++   // ecx=the first voice of next track
        jc      su_calculate_voices_loop            ; } while bit ecx-1 of bitmask is on
        push    _CX                                 ; Stack: next_instr ptrnrow
        cmp     al, HLD                             ; anything but hold causes action
        je      short su_update_voices_nexttrack
        mov     cl, byte [_BP]
        mov     edi, ecx
        add     edi, ebx
        shl     edi, MAX_UNITS_SHIFT + 6            ; each unit = 64 bytes and there are 1<<MAX_UNITS_SHIFT units + small header
     do inc     dword [,su_synth_obj+su_synthworkspace.voices+su_voice.release,_DI,] ; set the voice currently active to release; notice that it could increment any number of times
        cmp     al, HLD                             ; if cl < HLD (no new note triggered)
        jl      su_update_voices_nexttrack          ;   goto nexttrack
        inc     ecx                                 ; curvoice++
        cmp     ecx, edx                            ; if (curvoice >= num_voices)
        jl      su_update_voices_skipreset
        xor     ecx,ecx                             ;   curvoice = 0
su_update_voices_skipreset:
        mov     byte [_BP],cl
        add     ecx, ebx
        shl     ecx, MAX_UNITS_SHIFT + 6            ; each unit = 64 bytes and there are 1<<MAX_UNITS_SHIFT units + small header
     do{lea    _DI,[},su_synth_obj+su_synthworkspace.voices,_CX,]
        stosd                                       ; save note
        mov     ecx, (su_voice.size - su_voice.release)/4
        xor     eax, eax
        rep stosd                                   ; clear the workspace of the new voice, retriggering oscillators
su_update_voices_nexttrack:
        pop     _BX                                 ; ebx=first voice of next instrument, Stack: ptrnrow
        pop     _DX                                 ; edx=patrnrow
        add     _SI, MAX_PATTERNS
        inc     _BP
     do{cmp     _BP,},su_synth_obj+MAX_TRACKS
        jl      su_update_voices_trackloop
    ret

%else ; INCLUDE_MULTIVOICE_TRACKS not defined -> one voice per track ve_SIon

su_update_voices: ; Stack: retaddr row
    xor     edx, edx
    xor     ebx, ebx
    mov     bl, PATTERN_SIZE
    div     ebx                                 ; eax = current pattern, edx = current row in pattern
 do{lea     _SI, [},MANGLE_DATA(su_tracks),_AX,]; esi points to the pattern data for current track
    mov     _DI, PTRWORD su_synth_obj+su_synthworkspace.voices
    mov     bl, MAX_TRACKS                      ; MAX_TRACKS is always <= 32 so this is ok
su_update_voices_trackloop:
        movzx   eax, byte [_SI]                     ; eax = current pattern
        imul    eax, PATTERN_SIZE                   ; eax = offset to current pattern data
     do{movzx   eax, byte [}, MANGLE_DATA(su_patterns),_AX,_DX,]  ; ecx = note
        cmp     al, HLD                             ; anything but hold causes action
        je      short su_update_voices_nexttrack
        inc     dword [_DI+su_voice.release]        ; set the voice currently active to release; notice that it could increment any number of times
        jb      su_update_voices_nexttrack          ; if cl < HLD (no new note triggered)  goto nexttrack
su_update_voices_retrigger:
        stosd                                       ; save note
        mov     ecx, (su_voice.size - su_voice.release)/4  ; could be xor ecx, ecx; mov ch,...>>8, but will it actually be smaller after compression?
        xor     eax, eax
        rep stosd                                   ; clear the workspace of the new voice, retriggering oscillators
        jmp     short su_update_voices_skipadd
su_update_voices_nexttrack:
        add     _DI, su_voice.size
su_update_voices_skipadd:
        add     _SI, MAX_PATTERNS
        dec     ebx
        jnz     short su_update_voices_trackloop
    ret

%endif ;INCLUDE_MULTIVOICE_TRACKS

%endif ; SU_DISABLE_PLAYER

;-------------------------------------------------------------------------------
;   Include the rest of the code
;-------------------------------------------------------------------------------
%include "sointu/arithmetic_footer.inc"
%include "sointu/flowcontrol_footer.inc"
%include "sointu/sources_footer.inc"
%include "sointu/sinks_footer.inc"
; warning: at the moment effects has to be assembled after
; sources, as sources.asm defines SU_USE_WAVESHAPER
; if needed.
%include "sointu/effects_footer.inc"

%ifidn __OUTPUT_FORMAT__,win64
    %include "sointu/win64/gmdls_win64_footer.inc"
%endif

%ifidn __OUTPUT_FORMAT__,win32
    %include "sointu/win32/gmdls_win32_footer.inc"
%endif

;-------------------------------------------------------------------------------
;    Constants
;-------------------------------------------------------------------------------
SECT_DATA(suconst)

c_24                    dd      24
c_i128                  dd      0.0078125
c_RandDiv               dd      65536*32768
c_0_5                   dd      0.5
c_i12                   dd      0x3DAAAAAA
c_lfo_normalize         dd      0.000038
c_freq_normalize        dd      0.000092696138  ; // 220.0/(2^(69/12)) / 44100.0

%ifdef USE_C_DC_CONST
    c_dc_const          dd      0.99609375      ; R = 1 - (pi*2 * frequency /samplerate)
%endif

%ifdef USE_C_32767
    c_32767             dd      32767.0
%endif

%ifdef USE_C_BPMSCALE
    c_bpmscale          dd      2.206896551724138 ; 64/29, 29 values will be double speed, so you can go from ~ 1/2.2 speed to 2.2x speed
%endif

%ifdef USE_C_16
    c_16                dd      16.0
%endif

%ifdef USE_C_SAMPLEFREQ_SCALING
    c_samplefreq_scaling    dd      84.28074964676522       ; o = 0.000092696138, n = 72, f = 44100*o*2**(n/12), scaling = 22050/f <- so note 72 plays at the "normal rate"
%endif
