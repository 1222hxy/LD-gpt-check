import { cp, mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const root = resolve(here, "..");
const source = resolve(root, "../dashboard/dist");
const target = resolve(root, "dist/dashboard");

await rm(target, { force: true, recursive: true });
await mkdir(target, { recursive: true });
await cp(source, target, { recursive: true });
await writeFile(resolve(root, "dist/dashboard.html"), await readFile(resolve(target, "index.html")));
