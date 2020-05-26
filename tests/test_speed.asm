%define BPM 100

%include "../src/sointu.inc"

; warning: crashes ahead. Now that the bpm could be changed and even modulated by other
; signals, there is no easy way to figure out how many ticks your song is. Either
; allocate some extra memory of the output just in case or simulate exactly how many
; samples are outputted. Here the triplets are slightly faster than the original so 
; they fit the default MAX_TICKS that is calculated using the simple bpm assumption.
SU_BEGIN_PATTERNS
    PATTERN 64, 0, 64, 64, 64,  0, 64, 64, 64, 0, 64, 64,   65, 0, 65, 65,
    PATTERN 64, 0,  0,  0,  0,  0,  0,  0,  0, 0,  0,  0,    0, 0,  0,  0, ; 4-rows
    PATTERN 78, 0, 54,  0, 78,  0, 54,  0, 78, 0, 54,  0,   78, 0, 54,  0, ; triplets
SU_END_PATTERNS

SU_BEGIN_TRACKS
    TRACK   VOICES(1),0,0
    TRACK   VOICES(1),1,2
SU_END_TRACKS

SU_BEGIN_PATCH
    SU_BEGIN_INSTRUMENT VOICES(1) ; Instrument0
        SU_ENVELOPE MONO,ATTAC(64),DECAY(64),SUSTAIN(0),RELEASE(64),GAIN(128)
        SU_ENVELOPE MONO,ATTAC(64),DECAY(64),SUSTAIN(0),RELEASE(64),GAIN(128)
        SU_OSCILLAT MONO,TRANSPOSE(64),DETUNE(32),PHASE(0),COLOR(96),SHAPE(64),GAIN(128), FLAGS(TRISAW)
        SU_OSCILLAT MONO,TRANSPOSE(72),DETUNE(64),PHASE(64),COLOR(64),SHAPE(96),GAIN(128), FLAGS(TRISAW)
        SU_MULP     STEREO
        SU_OUT      STEREO,GAIN(128)
    SU_END_INSTRUMENT
    SU_BEGIN_INSTRUMENT VOICES(1) ; Speed changer
        SU_LOADNOTE MONO
        SU_SPEED
    SU_END_INSTRUMENT
SU_END_PATCH

%include "../src/sointu.asm"
