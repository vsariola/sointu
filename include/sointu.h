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
    unsigned int Globaltime;
    unsigned int RowTick;
    unsigned int RowLen;
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

// Returns the number of samples remaining in the buffer i.e. 0 if the buffer was
// filled completely.
//
// NOTE: The buffer should have a length of 2 * maxsamples, as the audio
// is stereo.
//
// You should always check if rowtick >= rowlen after calling this. If so, most
// likely you didn't get full buffer filled but the end of row was hit before
// filling the buffer. In that case, trigger/release new voices, set rowtick to 0.
//
// Beware of infinite loops: with a rowlen of 0; or without resetting rowtick
// between rows; or with a problematic synth patch e.g. if the speed is
// modulated to be become infinite, this function might return maxsamples i.e. not
// render any samples. If you try to call this with your buffer until the whole
// buffer is filled, you will be stuck in an infinite loop.
extern int CALLCONV su_render_samples(SynthState* synthState, int maxsamples, float* buffer);

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
extern const int su_aux_id;
extern const int su_oscillat_id;
extern const int su_loadval_id;
extern const int su_receive_id;
extern const int su_in_id;

#endif // _SOINTU_H
