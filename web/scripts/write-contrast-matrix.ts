import { readFile, writeFile } from "node:fs/promises";

import { ThemeContract } from "./design/theme-contract.ts";

const css = await readFile(
  new URL("../src/design-system/theme.css", import.meta.url),
  "utf8",
);
const report = new ThemeContract(css).report();
const output = new URL(
  "../../spec/001-want-keep-mvp/design-contrast.md",
  import.meta.url,
);
if (process.argv.includes("--check")) {
  if ((await readFile(output, "utf8")) !== report)
    throw new Error("Contrast matrix is stale");
} else {
  await writeFile(output, report);
}
