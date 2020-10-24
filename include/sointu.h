#ifndef _SOINTU_H
#define _SOINTU_H

#pragma pack(push,4) // this should be fine for both Go and assembly
typedef struct Unit {
    float State[8];
    float Ports[8];
} Unit;

typedef struct Voice {
    int Note;
    int Release;
    float Inputs[8];
    float Reserved[6];
    struct Unit Units[63];
} Voice;

typedef struct Synth {
    unsigned char Curvoices[32];
    float Left;
    float Right;
    float Aux[6];
    struct Voice Voices[32];
} Synth;

typedef struct DelayWorkspace {
    float Buffer[65536];
    float Dcin;
    float Dcout;
    float Filtstate;
} DelayWorkspace;

typedef struct SynthState {
    struct Synth Synth;
    struct DelayWorkspace Delaywrks[64]; // let's keep this as 64 for now, so the delays take 16 meg. If that's too little or too much, we can change this in future.
    unsigned char Commands[32 * 64];
    unsigned char Values[32 * 64 * 8];
    unsigned int Polyphony;
    unsigned int NumVoices;
    unsigned int RandSeed;
    unsigned int GlobalTick;
    unsigned int RowTick;
    unsigned int SamplesPerRow; // nominal value, actual rows could be more or less due to speed modulation
} SynthState;
#pragma pack(pop)

#if UINTPTR_MAX == 0xffffffff // are we 32-bit?
#if defined(__clang__) || defined(__GNUC__)
#define CALLCONV __attribute__ ((stdcall))
#elif defined(_WIN32)
#define CALLCONV __stdcall // on 32-bit platforms, we just use stdcall, as all know it
#endif
#else // 64-bit
#define CALLCONV  // the asm will use honor honor correct x64 ABI on all 64-bit platforms
#endif

#ifdef INCLUDE_GMDLS
extern void CALLCONV su_load_gmdls(void);
#endif

// su_render_samples(SynthState* synthState, int maxSamples, float* buffer):
//      Renders at most maxsamples to the buffer, using and modifying the
//      synthesizer state in synthState.
//
// Parameters:
//      synthState  pointer to current synthState. RandSeed should be > 0 e.g. 1
//                  Also synthState->SamplesPerRow cannot be 0 or nothing will be
//                  rendered; either set it to INT32_MAX to always render full
//                  buffer, or something like SAMPLE_RATE * 60 / (BPM * 4) for
//                  having 4 rows per beat.
//      maxSamples  maximum number of samples to be rendered.  buffer should
//                  have a length of 2 * maxsamples as the audio is stereo.
//      buffer      audio sample buffer, L R L R ...
//
// Returns:
//      -1  end of row was not reached & buffer full
//      0   end of row was reached & buffer full (there is space for zero
//          samples in the buffer)
//      n>0 end of row was reached & there is space for n samples in the buffer
//
// Beware of infinite loops: with a rowlen of 0; or without resetting rowtick
// between rows; or with a problematic synth patch e.g. if the speed is
// modulated to be become infinite, this function might return maxsamples i.e.
// not render any samples. If you try to call this with your buffer until the
// whole buffer is filled, you will be stuck in an infinite loop.
//
// So a reasonable track player would be something like:
//
// function render_buffer(maxsamples,buffer) {
//   remaining = maxsamples
//   for i = 0..MAX_TRIES       // limit retries to prevent infinite loop
//        remaining = su_render_samples(synthState,
//                                      remaining,
//                                      &buffer[(maxsamples-remaining)*2])
//        if remaining >= 0     // end of row reached
//            song_row++        // advance row
//            retrigger/release voices based on the new row
//        if remaining <= 0     // buffer full
//            return
//    return // could not fill buffer despite MAX_TRIES, something is wrong
//           // audio will come to sudden end
//  }
extern int CALLCONV su_render_samples(SynthState* synthState, int maxSamples, float* buffer);

// Arithmetic opcode ids
extern const int su_add_id;
extern const int su_addp_id;
extern const int su_pop_id;
extern const int su_loadnote_id;
extern const int su_mul_id;
extern const int su_mulp_id;
extern const int su_push_id;
extern const int su_xch_id;

// Effect opcode ids
extern const int su_distort_id;
extern const int su_hold_id;
extern const int su_crush_id;
extern const int su_gain_id;
extern const int su_invgain_id;
extern const int su_filter_id;
extern const int su_clip_id;
extern const int su_pan_id;
extern const int su_delay_id;
extern const int su_compres_id;

// Flowcontrol opcode ids
extern const int su_advance_id;
extern const int su_speed_id;

// Sink opcode ids
extern const int su_out_id;
extern const int su_outaux_id;
extern const int su_aux_id;
extern const int su_send_id;

// Source opcode ids
extern const int su_envelope_id;
extern const int su_noise_id;
extern const int su_oscillat_id;
extern const int su_loadval_id;
extern const int su_receive_id;
extern const int su_in_id;

#endif // _SOINTU_H
