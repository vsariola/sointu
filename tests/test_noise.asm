%define BPM 100

%include "sointu/header.inc"

BEGIN_PATTERNS
    PATTERN 64,0,68,0,32,0,0,0,75,0,78,0,0,0,0,0
END_PATTERNS

BEGIN_TRACKS
    TRACK VOICES(1),0
END_TRACKS

BEGIN_PATCH
    BEGIN_INSTRUMENT VOICES(1)
        SU_ENVELOPE STEREO(0),ATTACK(32),DECAY(32),SUSTAIN(64),RELEASE(64),GAIN(128)
        SU_NOISE    STEREO(0),SHAPE(64),GAIN(128)
        SU_MULP     STEREO(0)
        SU_ENVELOPE STEREO(0),ATTACK(32),DECAY(32),SUSTAIN(64),RELEASE(64),GAIN(128)
        SU_NOISE    STEREO(0),SHAPE(96),GAIN(128)
        SU_MULP     STEREO(0)
        SU_OUT      STEREO(1),GAIN(128)
    END_INSTRUMENT
END_PATCH

%include "sointu/footer.inc"
