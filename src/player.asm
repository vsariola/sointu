;-------------------------------------------------------------------------------
;    Uninitialized data
;-------------------------------------------------------------------------------
%ifdef INCLUDE_MULTIVOICE_TRACKS

SECT_BSS(subss)

su_current_voiceno      resd    MAX_TRACKS ; number of the last voice used for each track

SECT_DATA(suconst)

su_voicetrack_bitmask   dd      VOICETRACK_BITMASK; does the following voice belong to the same track

%endif

;-------------------------------------------------------------------------------
;    Constants
;-------------------------------------------------------------------------------
SECT_DATA(suconst)

%ifdef SU_USE_16BIT_OUTPUT
c_32767     dd      32767.0
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
            mov     edi, dword [esp+44] ; edi containts ptr            
            mov     esi, su_synth_obj+su_synth.left
            movsd   ; copy left channel to output buffer
            movsd   ; copy right channel to output buffer
            mov     dword [esp+44], edi ; save back the updated ptr
            lea     edi, [esi-8]            
            xor     eax,eax
            stosd   ; clear left channel so the VM is ready to write them again
            stosd   ; clear right channel so the VM is ready to write them again            
        %else
            mov     esi, dword [esp+44] ; esi points to the output buffer
            xor     ecx,ecx  
            xor     eax,eax
            %%loop: ; loop over two channels, left & right
                fld     dword [su_synth_obj+su_synth.left+ecx*4]
                call    su_clip
                fstp    dword [esi]
                mov     dword [su_synth_obj+su_synth.left+ecx*4],eax ; clear the sample so the VM is ready to write it
                add     esi,4
                cmp     ecx,2
                jl      %%loop
            mov     dword [esp+44], esi ; save esi back to stack
        %endif
    %else ; 16-bit output, always clipped. This is a bit legacy method.
        mov     esi, dword [esp+44] ; esi points to the output buffer
        mov     edi, su_synth_obj+su_synth.left
        mov     ecx, 2
        %%loop: ; loop over two channels, left & right
            fld     dword [edi]
            call    MANGLE_FUNC(su_clip_op,0)
            fmul    dword [c_32767]
            push    eax        
            fistp   dword [esp]
            pop     eax
            mov     word [esi],ax   ; // store integer converted right sample
            xor     eax,eax
            stosd
            add     esi,2
            loop    %%loop
        mov     dword [esp+44], esi ; save esi back to stack
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

EXPORT MANGLE_FUNC(su_render,4)         ; Stack: ptr
    pushad                              ; Stack: pushad ptr
    xor     eax, eax                    ; ecx is the current row
su_render_rowloop:                      ; loop through every row in the song        
        push    eax                     ; Stack: row pushad ptr
        call    su_update_voices        ; update instruments for the new row
        xor     eax, eax                ; ecx is the current sample within row
su_render_sampleloop:                   ; loop through every sample in the row
            push    eax                 ; Stack: sample row pushad ptr
            call    MANGLE_FUNC(su_run_vm,0) ; run through the VM code
            output_sound                ; *ptr++ = left, *ptr++ = right
            pop     eax                 ; Stack: row pushad ptr
            inc     eax
            cmp     eax, SAMPLES_PER_ROW        
            jl      su_render_sampleloop
        pop     eax                     ; Stack: pushad ptr
        inc     eax
        cmp     eax, TOTAL_ROWS        
        jl      su_render_rowloop
    popad                               ; Stack: ptr
    ret     4                           ; Stack emptied by ret

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
    mov     ebx, PATTERN_SIZE             
    div     ebx                                 ; eax = current pattern, edx = current row in pattern
    lea     esi, [MANGLE_DATA(su_tracks)+eax]   ; esi points to the pattern data for current track                
    xor     eax, eax                            ; eax is the first voice of next track
    xor     ebx, ebx                            ; ebx is the first voice of current track
    xor     ebp, ebp                            ; ebp is the current track being processed    
su_update_voices_trackloop:              
        movzx   eax, byte [esi]                     ; eax = current pattern        
        imul    eax, PATTERN_SIZE                   ; eax = offset to current pattern data
        movzx   eax, byte [MANGLE_DATA(su_patterns)+eax+edx]  ; eax = note     
        push    edx                                 ; Stack: ptrnrow 
        xor     edx, edx                            ; edx=0
        mov     ecx, ebx                            ; ecx=first voice of the track to be done
su_calculate_voices_loop:                           ; do {
        bt      dword [su_voicetrack_bitmask],ecx   ;   // notice that the incs don't set carry
        inc     edx                                 ;   edx++   // edx=numvoices
        inc     ecx                                 ;   ecx++   // ecx=the first voice of next track
        jc      su_calculate_voices_loop            ; } while bit ecx-1 of bitmask is on
        push    ecx                                 ; Stack: next_instr ptrnrow                   
        cmp     al, HLD                             ; anything but hold causes action
        je      short su_update_voices_nexttrack             
        mov     ecx, dword [su_current_voiceno+ebp*4]
        mov     edi, ecx
        add     edi, ebx
        imul    edi, edi, su_voice.size        
        mov     dword [su_synth_obj+su_synth.voices+edi+su_voice.release],1      ; set the voice currently active to release        
        cmp     al, HLD                             ; if cl < HLD (no new note triggered)
        jl      su_update_voices_nexttrack          ;   goto nexttrack        
        inc     ecx                                 ; curvoice++
        cmp     ecx, edx                            ; if (curvoice >= num_voices)
        jl      su_update_voices_skipreset
        xor     ecx,ecx                             ;   curvoice = 0
su_update_voices_skipreset:
        mov     dword [su_current_voiceno+ebp*4],ecx 
        add     ecx, ebx
        imul    ecx, ecx, su_voice.size        
        lea     edi, [su_synth_obj+su_synth.voices+ecx]
        stosd                                       ; save note
        mov     ecx, (su_voice.size - su_voice.release)/4
        xor     eax, eax
        rep stosd                                   ; clear the workspace of the new voice, retriggering oscillators        
su_update_voices_nexttrack:                               
        pop     ebx                                 ; ebx=first voice of next instrument, Stack: ptrnrow         
        pop     edx                                 ; edx=patrnrow 
        add     esi, MAX_PATTERNS 
        inc     ebp                
        cmp     ebp, MAX_TRACKS
        jl      short su_update_voices_trackloop
    ret

%else ; INCLUDE_MULTIVOICE_TRACKS not defined -> one voice per track version

su_update_voices: ; Stack: retaddr row      
    xor     edx, edx    
    mov     ebx, PATTERN_SIZE             
    div     ebx                                 ; eax = current pattern, edx = current row in pattern
    lea     esi, [MANGLE_DATA(su_tracks)+eax]   ; esi points to the pattern data for current track            
    lea     edi, [su_synth_obj+su_synth.voices]
    mov     ebp, MAX_TRACKS
su_update_voices_trackloop:                    
        movzx   eax, byte [esi]                     ; eax = current pattern        
        imul    eax, PATTERN_SIZE                   ; eax = offset to current pattern data
        movzx   eax, byte [MANGLE_DATA(su_patterns)+eax+edx]  ; ecx = note        
        cmp     al, HLD                             ; anything but hold causes action
        je      short su_update_voices_nexttrack                     
        mov     dword [edi+su_voice.release],1      ; set the voice currently active to release         
        jl      su_update_voices_nexttrack          ; if cl < HLD (no new note triggered)  goto nexttrack
su_update_voices_retrigger:       
        stosd                                       ; save note
        mov     ecx, (su_voice.size - su_voice.release)/4
        xor     eax, eax
        rep stosd                                   ; clear the workspace of the new voice, retriggering oscillators        
        jmp     short su_update_voices_skipadd
su_update_voices_nexttrack:        
        add     edi, su_voice.size
su_update_voices_skipadd:   
        add     esi, MAX_PATTERNS
        dec     ebp                
        jnz     short su_update_voices_trackloop
    ret

%endif ;INCLUDE_MULTIVOICE_TRACKS
