{{template "structs.asm" .}}
;-------------------------------------------------------------------------------
;   Uninitialized data: The synth object
;-------------------------------------------------------------------------------
{{.SectBss "synth_object"}}
su_synth_obj:
    resb    su_synthworkspace.size
    resb    {{.Song.Patch.NumDelayLines}}*su_delayline_wrk.size

{{- if or .RowSync (.HasOp "sync")}}
{{- if or (and (eq .OS "windows") (not .Amd64)) (eq .OS "darwin")}}
extern _syncBuf
{{- else}}
extern syncBuf
{{- end}}
{{- end}}


;-------------------------------------------------------------------------------
;   su_render_song function: the entry point for the synth
;-------------------------------------------------------------------------------
;   Has the signature su_render_song(void *ptr), where ptr is a pointer to
;   the output buffer. Renders the compile time hard-coded song to the buffer.
;   Stack:  output_ptr
;-------------------------------------------------------------------------------
{{.ExportFunc "su_render_song" "OutputBufPtr"}}
    {{-  if .Amd64}}
    {{- if eq .OS "windows"}}
    {{- .PushRegs "rcx" "OutputBufPtr" "rdi" "NonVolatileRsi" "rsi" "NonVolatile" "rbx" "NonVolatileRbx" "rbp" "NonVolatileRbp" | indent 4}} ; rcx = ptr to buf. rdi,rsi,rbx,rbp  nonvolatile
    {{- else}} ; SystemV amd64 ABI, linux mac or hopefully something similar
    {{- .PushRegs "rdi" "OutputBufPtr" "rbx" "NonVolatileRbx" "rbp" "NonVolatileRbp" | indent 4}}
    {{- end}}
    {{- else}}
    {{- .PushRegs | indent 4}}
    {{- end}}
    {{- $prologsize := len .Stacklocs}}
    {{- if or .RowSync (.HasOp "sync")}}
    {{- if or (and (eq .OS "windows") (not .Amd64)) (eq .OS "darwin")}}
    {{- .Prepare "_syncBuf"}}
    {{.Push (.Use "_syncBuf") "SyncBufPtr"}}
    {{- else}}
    {{- .Prepare "syncBuf"}}
    {{.Push (.Use "syncBuf") "SyncBufPtr"}}
    {{- end}}
    {{- end}}
    xor     eax, eax
    {{- if ne .VoiceTrackBitmask 0}}
    {{.Push (.VoiceTrackBitmask | printf "%v") "VoiceTrackBitmask"}}
    {{- end}}
    {{.Push "1" "RandSeed"}}
    {{.Push .AX "GlobalTick"}}
su_render_rowloop:                      ; loop through every row in the song
        {{.Push .AX "Row"}}
        {{.Call "su_update_voices"}}   ; update instruments for the new row
        xor     eax, eax                ; ecx is the current sample within row
su_render_sampleloop:                   ; loop through every sample in the row
            {{.Push .AX "Sample"}}
            {{- if .SupportsPolyphony}}
            {{.Push (.PolyphonyBitmask | printf "%v") "PolyphonyBitmask"}} ; does the next voice reuse the current opcodes?
            {{- end}}
            {{.Push (.Song.Patch.NumVoices | printf "%v") "VoicesRemain"}}
            mov     {{.DX}}, {{.PTRWORD}} su_synth_obj                       ; {{.DX}} points to the synth object
            mov     {{.COM}}, {{.PTRWORD}} su_patch_opcodes           ; COM points to vm code
            mov     {{.VAL}}, {{.PTRWORD}} su_patch_operands             ; VAL points to unit params
            {{- if .HasOp "delay"}}
            mov     {{.CX}}, {{.PTRWORD}} su_synth_obj + su_synthworkspace.size - su_delayline_wrk.filtstate
            {{- end}}
            lea     {{.WRK}}, [{{.DX}} + su_synthworkspace.voices]            ; WRK points to the first voice
            {{.Call "su_run_vm"}} ; run through the VM code
            {{.Pop .AX}}
            {{- if .SupportsPolyphony}}
            {{.Pop .AX}}
            {{- end}}
            {{- template "output_sound.asm" .}}                ; *ptr++ = left, *ptr++ = right
            {{.Pop .AX}}
            inc     dword [{{.Stack "GlobalTick"}}] ; increment global time, used by delays
            inc     eax
            cmp     eax, {{.Song.SamplesPerRow}}
            jl      su_render_sampleloop
        {{.Pop .AX}}                  ; Stack: pushad ptr
        inc     eax
        cmp     eax, {{mul .PatternLength .SequenceLength}}
        jl      su_render_rowloop
    ; rewind the stack the entropy of multiple pop {{.AX}} is probably lower than add
    {{- range slice .Stacklocs $prologsize}}
    {{$.Pop $.AX}}
    {{- end}}
    {{-  if .Amd64}}
    {{- if eq .OS "windows"}}
    ; Windows64 ABI, rdi rsi rbx rbp non-volatile
    {{- .PopRegs "rcx" "rdi" "rsi" "rbx" "rbp" | indent 4}}
    {{- else}}
    ; SystemV64 ABI (linux mac or hopefully something similar), rbx rbp non-volatile
    {{- .PopRegs "rdi" "rbx" "rbp" | indent 4}}
    {{- end}}
    ret
    {{- else}}
    {{- .PopRegs | indent 4}}
    ret     4
    {{- end}}

;-------------------------------------------------------------------------------
;   su_update_voices function: polyphonic & chord implementation
;-------------------------------------------------------------------------------
;   Input:      eax     :   current row within song
;   Dirty:      pretty much everything
;-------------------------------------------------------------------------------
{{.Func "su_update_voices"}}
{{- if ne .VoiceTrackBitmask 0}}
; The more complicated implementation: one track can trigger multiple voices
    xor     edx, edx
    mov     ebx, {{.PatternLength}}                   ; we could do xor ebx,ebx; mov bl,PATTERN_SIZE, but that would limit patternsize to 256...
    div     ebx                                 ; eax = current pattern, edx = current row in pattern
    {{.Prepare "su_tracks"}}
    lea     {{.SI}}, [{{.Use "su_tracks"}}+{{.AX}}]  ; esi points to the pattern data for current track
    xor     eax, eax                            ; eax is the first voice of next track
    xor     ebx, ebx                            ; ebx is the first voice of current track
    mov     {{.BP}}, {{.PTRWORD}} su_synth_obj           ; ebp points to the current_voiceno array
su_update_voices_trackloop:
        movzx   eax, byte [{{.SI}}]                     ; eax = current pattern
        imul    eax, {{.PatternLength}}                   ; eax = offset to current pattern data
{{- .Prepare "su_patterns" .AX | indent 4}}
        movzx   eax,byte [{{.Use "su_patterns" .AX}} + {{.DX}}]  ; eax = note
        push    {{.DX}}                                 ; Stack: ptrnrow
        xor     edx, edx                            ; edx=0
        mov     ecx, ebx                            ; ecx=first voice of the track to be done
su_calculate_voices_loop:                           ; do {
        bt      dword [{{.Stack "VoiceTrackBitmask"}} + {{.PTRSIZE}}],ecx ; test voicetrack_bitmask// notice that the incs don't set carry
        inc     edx                                 ;   edx++   // edx=numvoices
        inc     ecx                                 ;   ecx++   // ecx=the first voice of next track
        jc      su_calculate_voices_loop            ; } while bit ecx-1 of bitmask is on
        push    {{.CX}}                                 ; Stack: next_instr ptrnrow
        cmp     al, {{.Hold}}                    ; anything but hold causes action
        je      short su_update_voices_nexttrack
        mov     cl, byte [{{.BP}}]
        mov     edi, ecx
        add     edi, ebx
        shl     edi, 12           ; each unit = 64 bytes and there are 1<<MAX_UNITS_SHIFT units + small header
{{- .Prepare "su_synth_obj" | indent 4}}
        and     dword [{{.Use "su_synth_obj"}} + su_synthworkspace.voices + su_voice.sustain + {{.DI}}], 0 ; set the voice currently active to release; notice that it could increment any number of times
        cmp     al, {{.Hold}}                    ; if cl < HLD (no new note triggered)
        jl      su_update_voices_nexttrack          ;   goto nexttrack
        inc     ecx                                 ; curvoice++
        cmp     ecx, edx                            ; if (curvoice >= num_voices)
        jl      su_update_voices_skipreset
        xor     ecx,ecx                             ;   curvoice = 0
su_update_voices_skipreset:
        mov     byte [{{.BP}}],cl
        add     ecx, ebx
        shl     ecx, 12                           ; each unit = 64 bytes and there are 1<<6 units + small header
        lea     {{.DI}},[{{.Use "su_synth_obj"}} + su_synthworkspace.voices + {{.CX}}]
        stosd                                       ; save note
        stosd                                       ; save release
        mov     ecx, (su_voice.size - su_voice.inputs)/4
        xor     eax, eax
        rep stosd                                   ; clear the workspace of the new voice, retriggering oscillators
su_update_voices_nexttrack:
        pop     {{.BX}}                                 ; ebx=first voice of next instrument, Stack: ptrnrow
        pop     {{.DX}}                                 ; edx=patrnrow
        add     {{.SI}}, {{.SequenceLength}}
        inc     {{.BP}}
{{- $addrname := len .Song.Score.Tracks | printf "su_synth_obj + %v"}}
{{- .Prepare $addrname | indent 8}}
        cmp     {{.BP}},{{.Use $addrname}}
        jl      su_update_voices_trackloop
    ret
{{- else}}
; The simple implementation: each track triggers always the same voice
    xor     edx, edx
    xor     ebx, ebx
    mov     bl, {{.PatternLength}}           ; rows per pattern
    div     ebx                                 ; eax = current pattern, edx = current row in pattern
{{- .Prepare "su_tracks" | indent 4}}
    lea     {{.SI}}, [{{.Use "su_tracks"}}+{{.AX}}]; esi points to the pattern data for current track
    mov     {{.DI}}, {{.PTRWORD}} su_synth_obj+su_synthworkspace.voices
    mov     bl, {{len .Song.Score.Tracks}}                      ; MAX_TRACKS is always <= 32 so this is ok
su_update_voices_trackloop:
        movzx   eax, byte [{{.SI}}]                     ; eax = current pattern
        imul    eax, {{.PatternLength}}           ; multiply by rows per pattern, eax = offset to current pattern data
{{- .Prepare "su_patterns" .AX | indent 8}}
        movzx   eax, byte [{{.Use "su_patterns" .AX}} + {{.DX}}]  ; ecx = note
        cmp     al, {{.Hold}}                   ; anything but hold causes action
        je      short su_update_voices_nexttrack
        mov     dword [{{.DI}}+su_voice.sustain], eax     ; set the voice currently active to release
        jb      su_update_voices_nexttrack          ; if cl < HLD (no new note triggered)  goto nexttrack
su_update_voices_retrigger:
        stosd                                       ; save note
        stosd                                       ; save sustain
        mov     ecx, (su_voice.size - su_voice.inputs)/4  ; could be xor ecx, ecx; mov ch,...>>8, but will it actually be smaller after compression?
        xor     eax, eax
        rep stosd                                   ; clear the workspace of the new voice, retriggering oscillators
        jmp     short su_update_voices_skipadd
su_update_voices_nexttrack:
        add     {{.DI}}, su_voice.size
su_update_voices_skipadd:
        add     {{.SI}}, {{.SequenceLength}}
        dec     ebx
        jnz     short su_update_voices_trackloop
    ret
{{- end}}

{{template "patch.asm" .}}

;-------------------------------------------------------------------------------
;    Patterns
;-------------------------------------------------------------------------------
{{.Data "su_patterns"}}
{{- range .Patterns}}
    db {{. | toStrings | join ","}}
{{- end}}

;-------------------------------------------------------------------------------
;    Tracks
;-------------------------------------------------------------------------------
{{.Data "su_tracks"}}
{{- range .Sequences}}
    db {{. | toStrings | join ","}}
{{- end}}

{{- if gt (.SampleOffsets | len) 0}}
;-------------------------------------------------------------------------------
;    Sample offsets
;-------------------------------------------------------------------------------
{{.Data "su_sample_offsets"}}
{{- range .SampleOffsets}}
    dd {{.Start}}
    dw {{.LoopStart}}
    dw {{.LoopLength}}
{{- end}}
{{end}}

{{- if gt (.DelayTimes | len ) 0}}
;-------------------------------------------------------------------------------
;    Delay times
;-------------------------------------------------------------------------------
{{.Data "su_delay_times"}}
    dw {{.DelayTimes | toStrings | join ","}}
{{end}}

;-------------------------------------------------------------------------------
;    The code for this patch, basically indices to vm jump table
;-------------------------------------------------------------------------------
{{.Data "su_patch_opcodes"}}
    db {{.Opcodes | toStrings | join ","}}

;-------------------------------------------------------------------------------
;    The parameters / inputs to each opcode
;-------------------------------------------------------------------------------
{{.Data "su_patch_operands"}}
    db {{.Operands | toStrings | join ","}}

;-------------------------------------------------------------------------------
;    Constants
;-------------------------------------------------------------------------------
{{.SectData "constants"}}
{{.Constants}}
