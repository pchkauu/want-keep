import { contrastPairs } from "../../src/design-system/contrast-pairs.ts";

export class ThemeContract {
  readonly tokens: ReadonlyMap<string, string>;

  constructor(css: string) {
    const root = css.match(/:root\s*\{([^}]+)\}/)?.[1];
    if (!root) throw new Error("Theme has no root token block");
    const tokens = new Map<string, string>();
    for (const match of root.matchAll(/(--[\w-]+)\s*:\s*([^;]+);/g)) {
      if (tokens.has(match[1])) throw new Error(`Duplicate token ${match[1]}`);
      tokens.set(match[1], match[2].trim());
    }
    this.tokens = tokens;
  }

  value(name: string): string {
    const value = this.tokens.get(name);
    if (!value) throw new Error(`Missing theme token ${name}`);
    return value;
  }

  static contrast(first: string, second: string): number {
    const luminances = [first, second].map((hex) => {
      if (!/^#[\da-f]{6}$/i.test(hex))
        throw new Error(`Expected an opaque sRGB color: ${hex}`);
      const [r, g, b] = [1, 3, 5].map((offset) => {
        const channel = parseInt(hex.slice(offset, offset + 2), 16) / 255;
        return channel <= 0.04045
          ? channel / 12.92
          : ((channel + 0.055) / 1.055) ** 2.4;
      });
      return 0.2126 * r + 0.7152 * g + 0.0722 * b;
    });
    return (Math.max(...luminances) + 0.05) / (Math.min(...luminances) + 0.05);
  }

  matrix() {
    return contrastPairs.map((pair) => {
      const foreground = this.value(pair.foreground);
      const background = this.value(pair.background);
      const ratio = ThemeContract.contrast(foreground, background);
      return {
        ...pair,
        foreground,
        background,
        ratio,
        passes: pair.minimum === null || ratio >= pair.minimum,
      };
    });
  }

  report(): string {
    const rows = this.matrix();
    if (rows.some((row) => !row.passes))
      throw new Error(
        `Contrast failed: ${rows
          .filter((row) => !row.passes)
          .map((row) => row.id)
          .join(", ")}`,
      );
    return [
      "# Want Keep: contrast matrix / Матрица контраста",
      "",
      "Generated from CSS tokens by `node web/scripts/write-contrast-matrix.ts`. Do not edit manually. / Сгенерировано из CSS-токенов; не редактировать вручную.",
      "",
      "Thresholds use unrounded sRGB ratios. Exemptions apply only to the documented roles. / Порог проверяется без округления; исключения относятся только к указанным ролям.",
      "",
      "| Pair / Пара | FG | BG | Ratio | Minimum | Purpose / Назначение |",
      "| --- | --- | --- | --- | --- | --- |",
      ...rows.map(
        (row) =>
          `| ${row.id} | ${row.foreground} | ${row.background} | ${row.ratio.toFixed(3)}:1 | ${row.minimum ?? "exempt"} | ${row.purpose} |`,
      ),
      "",
    ].join("\n");
  }
}
