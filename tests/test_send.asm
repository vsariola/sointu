%define BPM 100
%define SINGLE_FILE
%define USE_SECTIONS

%include "../src/sointu.inc"

SU_BEGIN_PATTERNS
    PATTERN 64, HLD, HLD, HLD, HLD, HLD, HLD, HLD,  0, 0, 0, 0, 0, 0, 0, 0      
SU_END_PATTERNS

SU_BEGIN_TRACKS
    TRACK VOICES(1),0
SU_END_TRACKS

SU_BEGIN_PATCH
    SU_BEGIN_INSTRUMENT VOICES(1) ; Instrument0
        SU_LOADVAL MONO,VALUE(32)  ; should receive -0.5
        SU_SEND     MONO,AMOUNT(96),PORT(5,load_val,value) ; should send -0.25  
        SU_SEND     MONO,AMOUNT(96),PORT(6,load_val,value) + SEND_POP ; should send -0.25   
        SU_LOADVAL MONO,VALUE(128) ; should receive 1
        SU_SEND     MONO,AMOUNT(96),PORT(6,load_val,value) + SEND_POP ; should send 0.5
        SU_LOADVAL MONO,VALUE(64) ; should receive -0.5
        SU_LOADVAL MONO,VALUE(64) ; should receive 0.5
        SU_OUT      STEREO,GAIN(128)
    SU_END_INSTRUMENT
SU_END_PATCH

%include "../src/sointu.asm"
