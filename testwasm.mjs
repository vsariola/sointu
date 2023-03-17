import { readFile, writeFile } from 'fs/promises';

const wasm = (await readFile('./groove.wasm'));

const mod = await WebAssembly.instantiate(wasm, {
    m: {
        pow: Math.pow,
        log2: Math.log2,
        sin: Math.sin
    }
});

const mem = mod.instance.exports.m;

await writeFile('test.raw', new Uint8Array(mem.buffer));