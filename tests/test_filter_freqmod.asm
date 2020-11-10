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
        SU_ENVELOPE   STEREO(0),ATTACK(64),DECAY(64),SUSTAIN(64),RELEASE(72),GAIN(128)
        SU_OSCILLATOR STEREO(0),TRANSPOSE(64),DETUNE(64),PHASE(0),COLOR(128),SHAPE(64),GAIN(128),TYPE(TRISAW),LFO(0),UNISON(0)
        SU_MULP       STEREO(0)
        SU_FILTER     STEREO(0),FREQUENCY(32),RESONANCE(64),LOWPASS(0),BANDPASS(1),HIGHPASS(0),NEGBANDPASS(0),NEGHIGHPASS(0)
        SU_PAN        STEREO(0),PANNING(64)
        SU_OUT        STEREO(1),GAIN(128)
        SU_OSCILLATOR STEREO(0),TRANSPOSE(70),DETUNE(64),PHASE(64),COLOR(128),SHAPE(64),GAIN(128),TYPE(SINE),LFO(1),UNISON(0)
        SU_SEND       STEREO(0),AMOUNT(32),VOICE(0),UNIT(3),PORT(0),SENDPOP(1)
    END_INSTRUMENT
END_PATCH

%include "sointu/footer.inc"
