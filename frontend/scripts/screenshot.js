import { chromium } from "playwright";
import { mkdir } from "node:fs/promises";
import path from "node:path";

const target = process.env.URL || "http://127.0.0.1:5173/";
const outDir = path.resolve("screenshots");

const viewports = [
  { name: "desktop", width: 1440, height: 1100 },
  { name: "mobile", width: 390, height: 1100 },
];

await mkdir(outDir, { recursive: true });

const browser = await chromium.launch();
for (const viewport of viewports) {
  const page = await browser.newPage({
    viewport: { width: viewport.width, height: viewport.height },
    deviceScaleFactor: 1,
  });
  await page.goto(target, { waitUntil: "networkidle" });
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({
    path: path.join(outDir, `${viewport.name}.png`),
    fullPage: true,
  });
  await page.close();
}
await browser.close();

console.log(`Saved screenshots to ${outDir}`);
