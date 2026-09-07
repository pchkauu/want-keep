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

function relativeLuminance(color) {
  const channels = [1, 3, 5].map(
    (offset) => Number.parseInt(color.slice(offset, offset + 2), 16) / 255,
  );
  const linearChannels = channels.map((channel) =>
    channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4,
  );
  return (
    0.2126 * linearChannels[0] +
    0.7152 * linearChannels[1] +
    0.0722 * linearChannels[2]
  );
}

function contrastRatio(firstColor, secondColor) {
  const luminances = [
    relativeLuminance(firstColor),
    relativeLuminance(secondColor),
  ].sort((left, right) => right - left);
  return (luminances[0] + 0.05) / (luminances[1] + 0.05);
}

for (const [foregroundToken, backgroundToken, minimum] of [
  ["--accent-readable", "--card", 4.5],
  ["--input", "--card", 3],
]) {
  const ratio = contrastRatio(
    tokenColor(foregroundToken),
    tokenColor(backgroundToken),
  );
  if (ratio < minimum) {
    throw new Error(
      `${foregroundToken} on ${backgroundToken} has ${ratio.toFixed(2)}:1 contrast; expected at least ${minimum}:1`,
    );
  }
}
