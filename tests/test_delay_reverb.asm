%define BPM 100

%include "sointu/header.inc"

BEGIN_PATTERNS
    PATTERN 64, 0, 68, 0, 32, 0, 0, 0,  75, 0, 78, 0,   0, 0, 0, 0
END_PATTERNS

BEGIN_TRACKS
    TRACK VOICES(1),0
END_TRACKS

BEGIN_PATCH
    BEGIN_INSTRUMENT VOICES(1) ; Instrument0
        SU_ENVELOPE MONO,ATTACK(80),DECAY(80),SUSTAIN(64),RELEASE(80),GAIN(128)
        SU_OSCILLAT MONO,TRANSPOSE(64),DETUNE(64),PHASE(0),COLOR(128),SHAPE(64),GAIN(128),TYPE(SINE),LFO(0),UNISON(0)
        SU_MULP     MONO
        SU_DELAY    MONO,PREGAIN(40),DRY(128),FEEDBACK(125),DAMP(64),DELAY(0),COUNT(8),NOTETRACKING(0)
        SU_PAN      MONO,PANNING(64)
        SU_OUT      STEREO, GAIN(128)
    END_INSTRUMENT
END_PATCH

BEGIN_DELTIMES
    DELTIME 1116,1188,1276,1356,1422,1492,1556,1618
END_DELTIMES

%include "sointu/footer.inc"
