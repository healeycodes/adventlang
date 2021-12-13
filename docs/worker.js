importScripts("wasm_exec.js");
const runtime = (async () => {
  if (!WebAssembly.instantiateStreaming) {
    // polyfill
    WebAssembly.instantiateStreaming = async (resp, importObject) => {
      const source = await (await resp).arrayBuffer();
      return await WebAssembly.instantiate(source, importObject);
    };
  }
  const go = new self.Go();
  const wasm = await fetch("adventlang.wasm");
  const result = await WebAssembly.instantiateStreaming(wasm, go.importObject);
  go.run(result.instance);
})();

onmessage = async (e) => {
  await runtime;

  const start = Date.now();

  // Capture log()
  let logs = "";
  self.console.log = (s) => (logs += `${s}\n`);

  const result = self.adventlang(e.data);
  postMessage([`${logs}` + result, Date.now() - start]);
};
