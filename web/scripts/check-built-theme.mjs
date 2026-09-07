import { readdir, readFile } from "node:fs/promises";
import { URL } from "node:url";

const assetDirectory = new URL("../dist/assets/", import.meta.url);
const assetNames = await readdir(assetDirectory);
const stylesheetNames = assetNames.filter((name) => name.endsWith(".css"));

if (stylesheetNames.length === 0) {
  throw new Error("The web build did not produce a stylesheet");
}

const stylesheets = await Promise.all(
  stylesheetNames.map((name) =>
    readFile(new URL(name, assetDirectory), "utf8"),
  ),
);
const compiledCss = stylesheets.join("\n").toLowerCase();
const requiredMarkers = [
  "--primary:#5f4ef5",
  "--color-primary:var(--primary)",
  ".bg-background",
  ".text-primary",
  "scroll-fade",
];

for (const marker of requiredMarkers) {
  if (!compiledCss.includes(marker)) {
    throw new Error(`The compiled design theme is missing ${marker}`);
  }
}
