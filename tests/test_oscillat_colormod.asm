%define BPM 100

%include "sointu/header.inc"

BEGIN_PATTERNS
    PATTERN 80, HLD, HLD, HLD, HLD, HLD, HLD, HLD, HLD, HLD, HLD, 0, 0, 0, 0, 0
END_PATTERNS

BEGIN_TRACKS
    TRACK   VOICES(1),0
END_TRACKS

BEGIN_PATCH
    BEGIN_INSTRUMENT VOICES(1) ; Instrument0
        SU_ENVELOPE MONO,ATTACK(80),DECAY(80),SUSTAIN(64),RELEASE(80),GAIN(128)
        SU_OSCILLAT MONO,TRANSPOSE(64),DETUNE(64),PHASE(0),COLOR(128),SHAPE(64),GAIN(128),TYPE(SINE),LFO(0),UNISON(0)
        SU_MULP     MONO
        SU_PUSH     MONO
        SU_OSCILLAT MONO,TRANSPOSE(70),DETUNE(64),PHASE(64),COLOR(128),SHAPE(64),GAIN(128),TYPE(SINE),LFO(1),UNISON(0)
        SU_SEND     MONO,AMOUNT(68),UNIT(1),PORT(3),SENDPOP(1)
        SU_OUT      STEREO,GAIN(128)
    END_INSTRUMENT
END_PATCH

%include "sointu/footer.inc"
