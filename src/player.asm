%if BITS == 32
    %define render_prologue pushad ; stdcall & everything nonvolatile except eax, ecx, edx
    %macro render_epilogue 0
        popad
        ret     4 ; clean the passed parameter from stack.
    %endmacro
%elifidn __OUTPUT_FORMAT__,win64
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

struc su_playerstack ; the structure of stack _as the output sound sees it_
    .rowtick    RESPTR  1    ; which tick within this row are we at
    .row        RESPTR  1    ; which total row of the song are we at
    .tick       RESPTR  1    ; which total tick of the song are we at
    .randseed   RESPTR  1
%ifdef INCLUDE_MULTIVOICE_TRACKS
    .trackbits  RESPTR  1
%endif
    .cleanup
%if BITS == 32
    .regs       RESPTR  8
    .retaddr    RESPTR  1
%elifidn __OUTPUT_FORMAT__,win64
    .regs       RESPTR  4
%else
    .regs       RESPTR  2
%endif
    .bufferptr  RESPTR  1
    .size
endstruc

;===============================================================================
;   Uninitialized data: The one and only synth object
;===============================================================================
SECT_BSS(susynth)

su_synth_obj            resb    su_synth.size

%if DELAY_ID > -1       ; if we use delay, then the synth obj should be immediately followed by the delay workspaces
                        resb   NUM_DELAY_LINES*su_delayline_wrk.size
%endif
%ifdef INCLUDE_MULTIVOICE_TRACKS
su_curvoices            resd    MAX_TRACKS 
%endif

;-------------------------------------------------------------------------------
;    Constants
;-------------------------------------------------------------------------------
SECT_DATA(suconst)

%ifdef SU_USE_16BIT_OUTPUT
    %ifndef C_32767
        c_32767     dd      32767.0
        %define C_32767
    %endif
%endif

;-------------------------------------------------------------------------------
;   output_sound macro: used by the render function to write sound to buffer
;-------------------------------------------------------------------------------
;   The macro contains the ifdef hell to handle 16bit output and clipping cases
;   to keep the main function more readable
;   Stack   :   sample row pushad output_ptr
;-------------------------------------------------------------------------------
%macro output_sound 0
    %ifndef SU_USE_16BIT_OUTPUT
        %ifndef SU_CLIP_OUTPUT ; The modern way. No need to clip; OS can do it.
            mov     _DI, [_SP+su_playerstack.bufferptr] ; edi containts ptr
            mov     _SI, PTRWORD su_synth_obj + su_synth.left
            movsd   ; copy left channel to output buffer
            movsd   ; copy right channel to output buffer
            mov     [_SP+su_playerstack.bufferptr], _DI ; save back the updated ptr
            lea     _DI, [_SI-8]
            xor     eax, eax
            stosd   ; clear left channel so the VM is ready to write them again
            stosd   ; clear right channel so the VM is ready to write them again
        %else
            mov     _SI, qword [_SP+su_playerstack.bufferptr] ; esi points to the output buffer
            xor     _CX,_CX
            xor     eax,eax
            %%loop: ; loop over two channels, left & right
             do fld     dword [,su_synth_obj+su_synth.left,_CX*4,]
                call    su_clip
                fstp    dword [_SI]
             do mov     dword [,su_synth_obj+su_synth.left,_CX*4,{],eax} ; clear the sample so the VM is ready to write it
                add     _SI,4
                cmp     ecx,2
                jl      %%loop
            mov     dword [_SP+su_playerstack.bufferptr], _SI ; save esi back to stack
        %endif
    %else ; 16-bit output, always clipped. This is a bit legacy method.
        mov     _SI, [_SP+su_playerstack.bufferptr] ; esi points to the output buffer
        mov     _DI, PTRWORD su_synth_obj+su_synth.left
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
        mov     [_SP+su_playerstack.bufferptr], _SI ; save esi back to stack
    %endif
%endmacro

;-------------------------------------------------------------------------------
;   su_render function: the entry point for the synth
;-------------------------------------------------------------------------------
;   Has the signature su_render(void *ptr), where ptr is a pointer to
;   the output buffer
;   Stack:  output_ptr
;-------------------------------------------------------------------------------
SECT_TEXT(surender)

EXPORT MANGLE_FUNC(su_render,PTRSIZE)   ; Stack: ptr
    render_prologue
%ifdef INCLUDE_GMDLS
    call    su_gmdls_load
%endif
    xor     eax, eax
%ifdef INCLUDE_MULTIVOICE_TRACKS
    push    VOICETRACK_BITMASK
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
            mov     WRK, PTRWORD su_synth_obj                       ; WRK points to the synth object
            mov     COM, PTRWORD MANGLE_DATA(su_commands)           ; COM points to vm code
            mov     VAL, PTRWORD MANGLE_DATA(su_params)             ; VAL points to unit params
            call    MANGLE_FUNC(su_run_vm,0) ; run through the VM code
            %ifdef INCLUDE_POLYPHONY
                pop     _AX
            %endif  
            output_sound                ; *ptr++ = left, *ptr++ = right
            pop     _AX                 ; Stack: row pushad ptr
            inc     dword [_SP + PTRSIZE] ; increment global time, used by delays
            inc     eax
            cmp     eax, SAMPLES_PER_ROW
            jl      su_render_sampleloop
        pop     _AX                     ; Stack: pushad ptr
        inc     eax
        cmp     eax, TOTAL_ROWS
        jl      su_render_rowloop
%ifdef INCLUDE_MULTIVOICE_TRACKS
    add     _SP, su_playerstack.cleanup - su_playerstack.tick ; rewind the remaining tack
%else
    pop     _AX
    pop     _AX
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
        bt      dword [_SP + su_playerstack.trackbits + PTRSIZE],ecx ; test voicetrack_bitmask// notice that the incs don't set carry
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
     do inc     dword [,su_synth_obj+su_synth.voices+su_voice.release,_DI,] ; set the voice currently active to release; notice that it could increment any number of times
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
     do{lea    _DI,[},su_synth_obj+su_synth.voices,_CX,]
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
    mov     _DI, PTRWORD su_synth_obj+su_synth.voices
    mov     bl, MAX_TRACKS                      ; MAX_TRACKS is always <= 32 so this is ok
su_update_voices_trackloop:
        movzx   eax, byte [_SI]                     ; eax = current pattern
        imul    eax, PATTERN_SIZE                   ; eax = offset to current pattern data
     do{movzx   eax, byte [}, MANGLE_DATA(su_patterns),_AX,_DX,]  ; ecx = note
        cmp     al, HLD                             ; anything but hold causes action
        je      short su_update_voices_nexttrack
        inc     dword [_DI+su_voice.release]        ; set the voice currently active to release; notice that it could increment any number of times
        cmp     al, HLD
        jl      su_update_voices_nexttrack          ; if cl < HLD (no new note triggered)  goto nexttrack
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
