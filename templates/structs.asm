;-------------------------------------------------------------------------------
;   unit struct
;-------------------------------------------------------------------------------
struc su_unit
    .state      resd    8
    .ports      resd    8
    .size:
endstruc

;-------------------------------------------------------------------------------
;   voice struct
;-------------------------------------------------------------------------------
struc su_voice
    .note       resd    1
    .release    resd    1
    .inputs     resd    8
    .reserved   resd    6 ; this is done to so the whole voice is 2^n long, see polyphonic player
    .workspace  resb    63 * su_unit.size
    .size:
endstruc

;-------------------------------------------------------------------------------
;   synthworkspace struct
;-------------------------------------------------------------------------------
struc su_synthworkspace
    .curvoices  resb    32      ; these are used by the multitrack player to store which voice is playing on which track
    .left       resd    1
    .right      resd    1
    .aux        resd    6       ; 3 auxiliary signals
    .voices     resb    32 * su_voice.size
    .size:
endstruc

;-------------------------------------------------------------------------------
;   su_delayline_wrk struct
;-------------------------------------------------------------------------------
struc   su_delayline_wrk
    .dcin       resd    1
    .dcout      resd    1
    .filtstate  resd    1
    .buffer     resd    65536
    .size:
endstruc
