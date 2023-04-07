import { readFile, writeFile } from 'fs/promises';

const wasm = (await readFile('./temp_song_file.wasm'));

const mod = await WebAssembly.instantiate(wasm, {});
mod.instance.exports.render();
const mem = mod.instance.exports.m;
await writeFile('temp_song_file.raw', new Uint8Array(mem.buffer,mod.instance.exports.s,mod.instance.exports.l));
