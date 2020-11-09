%define BPM 100

%include "sointu/header.inc"

BEGIN_PATTERNS
    PATTERN 64, 0, 68, 0, 32, 0, 0, 0,  75, 0, 78, 0,   0, 0, 0, 0
END_PATTERNS

BEGIN_TRACKS
    TRACK   VOICES(1),0
END_TRACKS

BEGIN_PATCH
    BEGIN_INSTRUMENT VOICES(1) ; Instrument0
        SU_ENVELOPE MONO, ATTACK(32),DECAY(32),SUSTAIN(128),RELEASE(64),GAIN(128)
        SU_ENVELOPE MONO, ATTACK(32),DECAY(32),SUSTAIN(128),RELEASE(64),GAIN(128)
        SU_OSCILLAT MONO, TRANSPOSE(64),DETUNE(64),PHASE(0),COLOR(96),SHAPE(64),GAIN(128),TYPE(SINE),LFO(0),UNISON(0)
        SU_OSCILLAT MONO, TRANSPOSE(72),DETUNE(64),PHASE(64),COLOR(64),SHAPE(96),GAIN(128),TYPE(SINE),LFO(0),UNISON(0)
        SU_MULP     STEREO
        SU_INVGAIN  STEREO,INVGAIN(64)
        SU_CLIP     MONO
        SU_GAIN     STEREO,GAIN(64)
        SU_OUT      STEREO,GAIN(128)
    END_INSTRUMENT
END_PATCH

%include "sointu/footer.inc"
