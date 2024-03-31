import { URL } from "url";
import { readFileSync, writeFileSync } from "fs";

const versionFilePath = new URL(
  "../../params/version.go",
  import.meta.url
).pathname;

const versionFileContent = readFileSync(versionFilePath, { encoding: "utf-8" });

const currentVersionPatch = versionFileContent.match(
  /VersionPatch = (?<patch>\d+)/
).groups.patch;

try {
  parseInt(currentVersionPatch);
} catch (err) {
  console.error(new Error("Failed to parse version in version.go file"));
  throw err;
}

// prettier-ignore
const newVersionPatch = `${parseInt(currentVersionPatch) + 1}`;

console.log(
  `Bump version from ${currentVersionPatch} to ${newVersionPatch}`
);

writeFileSync(
  versionFilePath,
  versionFileContent.replace(
    `VersionPatch = ${currentVersionPatch}`,
    `VersionPatch = ${newVersionPatch}`
  )
);
