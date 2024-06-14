// @ts-check

async function readStream(stream, encoding = "utf8") {
  stream.setEncoding(encoding);

  return new Promise((resolve, reject) => {
    let data = "";

    stream.on("data", (chunk) => (data += chunk));
    stream.on("end", () => resolve(data));
    stream.on("error", (error) => reject(error));
  });
}

function getSolcJs(solcJsPath) {
  const solcWrapper = require("solc/wrapper");
  return solcWrapper(require(solcJsPath));
}

async function main() {
  const input = await readStream(process.stdin);

  const solcjsPath = process.argv[2];
  const solc = getSolcJs(solcjsPath);

  const output = solc.compile(input);

  console.log(output);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
