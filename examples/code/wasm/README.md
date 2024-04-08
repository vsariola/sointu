Requirements: sointu binaries, `wabt`

To generate the .wasm file:

```
sointu-compile -o . -arch=wasm tests/test_chords.yml
wat2wasm --enable-bulk-memory test_chords.wat
```

To run the example:

```
npx serve examples/code/wasm
```
