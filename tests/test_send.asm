%define BPM 100

%include "sointu/header.inc"

BEGIN_PATTERNS
    PATTERN 64, HLD, HLD, HLD, HLD, HLD, HLD, HLD,  0, 0, 0, 0, 0, 0, 0, 0      
END_PATTERNS

BEGIN_TRACKS
    TRACK VOICES(1),0
END_TRACKS

BEGIN_PATCH
    BEGIN_INSTRUMENT VOICES(1) ; Instrument0
        SU_LOADVAL MONO,VALUE(32)  ; should receive -0.5
        SU_SEND     MONO,AMOUNT(96),UNIT(5),PORT(0),FLAGS(NONE) ; should send -0.25  
        SU_SEND     MONO,AMOUNT(96),UNIT(6),PORT(0),FLAGS(SEND_POP) ; should send -0.25   
        SU_LOADVAL MONO,VALUE(128) ; should receive 1
        SU_SEND     MONO,AMOUNT(96),UNIT(6),PORT(0),FLAGS(SEND_POP) ; should send 0.5
        SU_LOADVAL MONO,VALUE(64) ; should receive -0.5
        SU_LOADVAL MONO,VALUE(64) ; should receive 0.5
        SU_OUT      STEREO,GAIN(128)
    END_INSTRUMENT
END_PATCH

%include "sointu/footer.inc"
