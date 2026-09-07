import { readdir, readFile } from "node:fs/promises";
import { URL } from "node:url";
import { ThemeContract } from "./design/theme-contract.ts";

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
  ".text-accent-readable",
  "scroll-fade",
];

for (const marker of requiredMarkers) {
  if (!compiledCss.includes(marker)) {
    throw new Error(`The compiled design theme is missing ${marker}`);
  }
}

function tokenColor(name) {
  const match = compiledCss.match(
    new RegExp(`${name}:(#[0-9a-f]{6}|#[0-9a-f]{3})(?=[;}])`),
  );
  if (!match) {
    throw new Error(`The compiled design theme is missing color token ${name}`);
  }
  const color = match[1];
  return color.length === 4
    ? `#${[...color.slice(1)].map((channel) => channel.repeat(2)).join("")}`
    : color;
}

const sourceCss = await readFile(
  new URL("../src/design-system/theme.css", import.meta.url),
  "utf8",
);
const theme = new ThemeContract(sourceCss);
for (const pair of theme.matrix()) {
  if (!pair.passes) throw new Error(`Source contrast failed: ${pair.id}`);
}
for (const [name, color] of theme.tokens) {
  if (/^#[a-f0-9]{6}$/i.test(color) && tokenColor(name) !== color) {
    throw new Error(`Compiled color changed: ${name}`);
  }
}
const manifest = JSON.parse(
  await readFile(
    new URL("../public/fonts/manifest.json", import.meta.url),
    "utf8",
  ),
);
for (const item of manifest) {
  const original = await readFile(
    new URL(`../public/fonts/${item.file}`, import.meta.url),
  );
  const built = await readFile(
    new URL(`../dist/fonts/${item.file}`, import.meta.url),
  );
  if (!original.equals(built))
    throw new Error(`Built font changed: ${item.file}`);
  if (!compiledCss.includes(`/fonts/${item.file.toLowerCase()}`))
    throw new Error(`Font face missing: ${item.file}`);
}
const scripts = await Promise.all(
  assetNames
    .filter((name) => name.endsWith(".js"))
    .map((name) => readFile(new URL(name, assetDirectory), "utf8")),
);
if (
  /token-preview|Design foundation|tokens-preview|\/__design\/(tokens|components)|Предельная длина|UISTATE-17|Finance\. Clearer\. Closer\./.test(
    scripts.join("\n"),
  ) ||
  compiledCss.includes(".token-preview") ||
  compiledCss.includes(".component-catalog") ||
  compiledCss.includes(".catalog-composition")
) {
  throw new Error(
    "Development token specimens leaked into the production build",
  );
}
