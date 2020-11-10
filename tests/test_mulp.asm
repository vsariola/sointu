%define BPM 100

%include "sointu/header.inc"

BEGIN_PATTERNS
    PATTERN 64,HLD,HLD,HLD,HLD,HLD,HLD,HLD,0,0,0,0,0,0,0,0
END_PATTERNS

BEGIN_TRACKS
    TRACK VOICES(1),0
END_TRACKS

BEGIN_PATCH
    BEGIN_INSTRUMENT VOICES(1)
        SU_LOADVAL STEREO(0),VALUE(96)
        SU_LOADVAL STEREO(0),VALUE(0)
        SU_MULP    STEREO(0)
        SU_LOADVAL STEREO(0),VALUE(96)
        SU_LOADVAL STEREO(0),VALUE(128)
        SU_MULP    STEREO(0)
        SU_OUT     STEREO(1),GAIN(128)
    END_INSTRUMENT
END_PATCH

%include "sointu/footer.inc"
