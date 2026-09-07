import { createHash } from "node:crypto";
import { readFile } from "node:fs/promises";

import { describe, expect, it } from "vitest";

import { ThemeContract } from "../scripts/design/theme-contract";
import { contrastPairs } from "@/design-system/contrast-pairs";

const web = new URL("../", import.meta.url);
const css = await readFile(new URL("src/design-system/theme.css", web), "utf8");
const theme = new ThemeContract(css);

describe("design-tokens", () => {
  it("preserves the agreed colors and radii", () => {
    for (const [token, expected] of Object.entries({
      "--background": "#1a1a1a",
      "--card": "#202020",
      "--popover": "#262626",
      "--primary": "#5f4ef5",
      "--wk-sand": "#c7af8f",
      "--wk-surface-warm": "#242220",
      "--wk-focus-fill": "#f5f5f5",
      "--wk-focus-foreground": "#1a1a1a",
      "--input": "#777777",
      "--border": "#3a3a3a",
      "--wk-radius-badge": "2px",
      "--radius": "4px",
      "--wk-radius-panel": "8px",
    }))
      expect(theme.value(token)).toBe(expected);
    expect(css).toContain("--color-primary: var(--primary)");
    expect(css).toContain("--font-sans: var(--wk-font-interface)");
    expect(css).toContain("--font-brand: var(--wk-font-brand)");
  });

  it("keeps regular and event timing within the motion contract", () => {
    for (const [name, minimum, maximum] of [
      ["fast", 150, 200],
      ["normal", 150, 200],
      ["account", 600, 1000],
      ["saving", 600, 1000],
      ["goal", 1000, 1600],
      ["shared-goal", 1, 1800],
      ["limit", 150, 200],
    ] as const) {
      const duration = theme.value(`--wk-motion-${name}`);
      expect(duration).toMatch(/^\d+ms$/);
      expect(parseInt(duration)).toBeGreaterThanOrEqual(minimum);
      expect(parseInt(duration)).toBeLessThanOrEqual(maximum);
    }
  });

  it("checks each permitted contrast pair and keeps the report reproducible", async () => {
    expect(new Set(contrastPairs.map((pair) => pair.id)).size).toBe(
      contrastPairs.length,
    );
    expect(theme.matrix().filter((row) => !row.passes)).toEqual([]);
    expect(
      await readFile(
        new URL("../spec/001-want-keep-mvp/design-contrast.md", web),
        "utf8",
      ),
    ).toBe(theme.report());
    expect(ThemeContract.contrast("#000000", "#ffffff")).toBe(21);
    expect(ThemeContract.contrast("#777777", "#ffffff")).toBeLessThan(4.5);
    expect(() => ThemeContract.contrast("transparent", "#ffffff")).toThrow();
    expect(() => theme.value("--unknown")).toThrow();
    expect(() =>
      new ThemeContract(
        css.replace("--foreground: #f5f5f5", "--foreground: #202020"),
      ).report(),
    ).toThrow("Contrast failed");
  });

  it("keeps fonts complete, licensed, local and pinned", async () => {
    const manifest: {
      file: string;
      license: string;
      sha256: string;
      licenseSha256: string;
      source: string;
      revision: string;
    }[] = JSON.parse(
      await readFile(new URL("public/fonts/manifest.json", web), "utf8"),
    );
    expect(manifest).toHaveLength(2);
    const typography = await readFile(
      new URL("src/design-system/typography.css", web),
      "utf8",
    );
    for (const item of manifest) {
      for (const [file, digest] of [
        [item.file, item.sha256],
        [item.license, item.licenseSha256],
      ]) {
        const bytes = await readFile(new URL(`public/fonts/${file}`, web));
        expect(createHash("sha256").update(bytes).digest("hex")).toBe(digest);
      }
      expect(item.source).toContain(
        `https://raw.githubusercontent.com/google/fonts/${item.revision}/`,
      );
      expect(typography).toContain(`/fonts/${item.file}`);
      expect(
        await readFile(new URL(`public/fonts/${item.license}`, web), "utf8"),
      ).toContain("SIL OPEN FONT LICENSE Version 1.1");
    }
    expect(typography.match(/font-display: swap/g)).toHaveLength(2);
    expect(typography).toContain("lining-nums tabular-nums");
    expect(theme.value("--wk-font-brand")).toBe(
      '"Pixelify Sans", "Manrope", system-ui, sans-serif',
    );
    expect(typography).not.toMatch(/https?:\/\//);
  });

  it("keeps the supplied logo byte-identical and the first paint dark", async () => {
    const logo = await readFile(new URL("public/brand/logo_512px.svg", web));
    expect(
      logo.equals(
        await readFile(
          new URL("../spec/001-want-keep-mvp/assets/logo_512px.svg", web),
        ),
      ),
    ).toBe(true);
    expect(logo.toString()).toContain('viewBox="0 0 455 512"');
    const html = await readFile(new URL("index.html", web), "utf8");
    expect(html).toContain(`background: ${theme.value("--background")}`);
    expect(html).toContain(
      `name="theme-color" content="${theme.value("--background")}"`,
    );
    expect(html).toContain('name="color-scheme" content="only dark"');
  });
});
