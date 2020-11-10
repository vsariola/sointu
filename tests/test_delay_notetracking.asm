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
        SU_ENVELOPE   STEREO(0),ATTACK(0),DECAY(0),SUSTAIN(96),RELEASE(96),GAIN(128)
        SU_ENVELOPE   STEREO(0),ATTACK(0),DECAY(48),SUSTAIN(0),RELEASE(0),GAIN(128)
        SU_OSCILLATOR STEREO(0),TRANSPOSE(64),DETUNE(64),PHASE(0),COLOR(64),SHAPE(127),GAIN(64),TYPE(SINE),LFO(0),UNISON(0)
        SU_MULP       STEREO(0)
        SU_FILTER     STEREO(0),FREQUENCY(32),RESONANCE(128),LOWPASS(1),BANDPASS(1),HIGHPASS(1),NEGBANDPASS(0),NEGHIGHPASS(0)
        SU_DELAY      STEREO(0),PREGAIN(128),DRY(128),FEEDBACK(128),DAMP(16),DELAY(0),COUNT(1),NOTETRACKING(1)
        SU_FILTER     STEREO(0),FREQUENCY(24),RESONANCE(128),LOWPASS(1),BANDPASS(1),HIGHPASS(1),NEGBANDPASS(0),NEGHIGHPASS(0)
        SU_MULP       STEREO(0)
        SU_PAN        STEREO(0),PANNING(64)
        SU_OUT        STEREO(1),GAIN(128)
    END_INSTRUMENT
END_PATCH

BEGIN_DELTIMES
    DELTIME 10787
END_DELTIMES

%include "sointu/footer.inc"
