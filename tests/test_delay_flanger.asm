%include "sointu/header.inc"

BEGIN_SONG BPM(100),OUTPUT_16BIT(0),CLIP_OUTPUT(0),DELAY_MODULATION(1)

BEGIN_PATTERNS
    PATTERN 80,HLD,HLD,HLD,HLD,HLD,HLD,HLD,HLD,HLD,HLD,0,0,0,0,0
END_PATTERNS

BEGIN_TRACKS
    TRACK VOICES(1),0
END_TRACKS

BEGIN_PATCH
    BEGIN_INSTRUMENT VOICES(1)
        SU_ENVELOPE   STEREO(0),ATTACK(80),DECAY(80),SUSTAIN(64),RELEASE(80),GAIN(128)
        SU_OSCILLATOR STEREO(0),TRANSPOSE(64),DETUNE(64),PHASE(0),COLOR(128),SHAPE(64),GAIN(128),TYPE(1),LFO(0),UNISON(0)
        SU_MULP       STEREO(0)
        SU_DELAY      STEREO(0),PREGAIN(40),DRY(128),FEEDBACK(0),DAMP(64),DELAY(0),COUNT(1),NOTETRACKING(0)
        SU_PAN        STEREO(0),PANNING(64)
        SU_OUT        STEREO(1),GAIN(128)
        SU_OSCILLATOR STEREO(0),TRANSPOSE(50),DETUNE(64),PHASE(64),COLOR(128),SHAPE(64),GAIN(128),TYPE(0),LFO(1),UNISON(0)
        SU_SEND       STEREO(0),AMOUNT(65),VOICE(0),UNIT(3),PORT(5),SENDPOP(1)
    END_INSTRUMENT
END_PATCH

BEGIN_DELTIMES
    DELTIME 1000
END_DELTIMES

END_SONG
