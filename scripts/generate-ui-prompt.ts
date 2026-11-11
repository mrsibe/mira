import { writeFileSync } from "node:fs";
import { resolve } from "node:path";
import { genUIPrompt } from "../src/genui/catalog";

const target = resolve(import.meta.dir, "../internal/prompt/ui_catalog.txt");
writeFileSync(target, `${genUIPrompt.trim()}\n`, "utf8");
console.log(`Wrote ${target}`);
