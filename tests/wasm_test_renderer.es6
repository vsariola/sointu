'use strict';
const fs = require('fs');
const path = require('path');
const { exit } = require('process');

if (process.argv.length <= 3) {
  console.log("Usage: wasm_test_renderer.es6 path/to/compiled_wasm_song.wasm path/to/expected_output.raw")
  console.log("The test renderer needs to know the location and length of the output buffer in wasm memory; remember to sointu-compile the .wat with TBW")
  exit(2)
}

(async () => {
  var file,wasm,instance
  try {
    file = fs.readFileSync(process.argv[2])
  } catch (err) {
    console.error("could not read wasmfile "+process.argv[2]+": "+err);
    return 1
  }

  try {
    wasm = await WebAssembly.compile(file);
  } catch (err) {
    console.error("could not compile wasmfile "+process.argv[2]+": "+err);
    return 1
  }

  try {
    instance = await WebAssembly.instantiate(wasm,{m:Math});
  } catch (err) {
    console.error("could not instantiate wasmfile "+process.argv[2]+": "+err);
    return 1
  }

  let gotBuffer = instance.exports.t.value ?
    new Int16Array(instance.exports.m.buffer,instance.exports.s.value,instance.exports.l.value/2) :
    new Float32Array(instance.exports.m.buffer,instance.exports.s.value,instance.exports.l.value/4);


  const gotFileName = path.join(path.parse(process.argv[2]).dir,"wasm_got_" + path.parse(process.argv[3]).name+".raw");
  try {
    const gotByteBuffer = Buffer.from(instance.exports.m.buffer,instance.exports.s.value,instance.exports.l.value);
    fs.writeFileSync(gotFileName, gotByteBuffer);
  } catch (err) {
    console.error("could not save the buffer we got to disk "+gotFileName+": "+err);
    return 1
  }

  const expectedFile = fs.readFileSync(process.argv[3]);
  let expectedBuffer = instance.exports.t.value ?
    new Int16Array(expectedFile.buffer, expectedFile.offset, expectedFile.byteLength/2) :
    new Float32Array(expectedFile.buffer, expectedFile.offset, expectedFile.byteLength/4);

  if (gotBuffer.length < expectedBuffer.length)
  {
    console.error("got shorter buffer than expected");
    return 1
  }

  if (gotBuffer.length > expectedBuffer.length)
  {
    console.error("got longer buffer than expected");
    return 1
  }

  let margin = 1e-2 * (instance.exports.t.value ? 32767 : 1);

  var firstError = true, firstErrorPos, errorCount = 0
  // we still have occasional sample wrong here or there. We only consider this a true error
  // if the total number of errors is too high
  for (var i = 2; i < gotBuffer.length-2; i++) {
    // Pulse oscillators with their sharp changes can sometimes be one sample late
    // due to rounding errors, causing the test fail. So, we test three samples
    // and if none match, then this sample is really wrong. Note that this is stereo
    // buffer so -2 index is the previous sample.
    // Also, we're pretty liberal on the accuracy, as small rounding errors
    // in frequency cause tests fails as the waves developed a phase shift over time
    // (or rounding errors in delay buffers etc.)
    if (Math.abs(gotBuffer[i] - expectedBuffer[i-2]) > margin &&
        Math.abs(gotBuffer[i] - expectedBuffer[i]) > margin &&
        Math.abs(gotBuffer[i] - expectedBuffer[i+2]) > margin) {
        if (firstError) {
            firstErrorPos = i
            firstError = false
        }
        errorCount++
    }
    if (errorCount > 200) {
        console.error("got different buffer than expected. First error at: "+(firstErrorPos/2|0)+(firstErrorPos%1," right"," left"));
        return 1;
    }
  }

  return 0;
})().then(retval => exit(retval));
