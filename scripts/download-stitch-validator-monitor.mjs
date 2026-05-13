#!/usr/bin/env node
/**
 * Downloads HTML (and screenshot when available) for Stitch project
 * "Validator Monitor List" into docs/stitch-validator-monitor-list/.
 *
 * Usage:
 *   export STITCH_API_KEY=...   # or reads from .cursor/mcp.json X-Goog-Api-Key
 *   node scripts/download-stitch-validator-monitor.mjs
 */
import { mkdir, writeFile, access } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { readFile } from "node:fs/promises";
import { stitch } from "@google/stitch-sdk";

const __dirname = dirname(fileURLToPath(import.meta.url));

async function findRepoRoot(startDir) {
  if (process.env.PAULI_REPO_ROOT) {
    return process.env.PAULI_REPO_ROOT;
  }
  let dir = startDir;
  for (let i = 0; i < 12; i++) {
    const marker = join(dir, "go.mod");
    try {
      await access(marker);
      return dir;
    } catch {
      const parent = dirname(dir);
      if (parent === dir) break;
      dir = parent;
    }
  }
  return join(__dirname, "..");
}

const REPO_ROOT = await findRepoRoot(__dirname);
const OUT_DIR = join(REPO_ROOT, "docs", "stitch-validator-monitor-list");
const PROJECT_ID = "13758647080274697556";

async function loadApiKey() {
  if (process.env.STITCH_API_KEY?.trim()) {
    return process.env.STITCH_API_KEY.trim();
  }
  const mcpPath = join(REPO_ROOT, ".cursor", "mcp.json");
  const raw = JSON.parse(await readFile(mcpPath, "utf8"));
  const key = raw?.mcpServers?.stitch?.headers?.["X-Goog-Api-Key"];
  if (!key) throw new Error("No STITCH_API_KEY and no X-Goog-Api-Key in .cursor/mcp.json");
  return String(key).trim();
}

function downloadUrl(obj) {
  if (!obj || typeof obj !== "object") return null;
  const u = obj.downloadUrl;
  return typeof u === "string" && u.startsWith("http") ? u : null;
}

async function fetchBinary(url, apiKey) {
  const res = await fetch(url, {
    headers: { "X-Goog-Api-Key": apiKey },
    redirect: "follow",
  });
  if (!res.ok) {
    const t = await res.text().catch(() => "");
    throw new Error(`GET ${url.slice(0, 80)}… → ${res.status}: ${t.slice(0, 200)}`);
  }
  return Buffer.from(await res.arrayBuffer());
}

async function main() {
  const apiKey = await loadApiKey();
  process.env.STITCH_API_KEY = apiKey;

  await mkdir(OUT_DIR, { recursive: true });

  const projects = await stitch.projects();
  const meta = projects.find((p) => p.projectId === PROJECT_ID);
  const projectTitle = meta?.data?.title ?? "Validator Monitor List";

  const project = stitch.project(PROJECT_ID);
  const screens = await project.screens();

  const manifest = {
    projectId: PROJECT_ID,
    projectTitle,
    downloadedAt: new Date().toISOString(),
    screens: [],
  };

  for (const screen of screens) {
    const id = screen.screenId;
    const d = screen.data ?? {};
    const title = d.title ?? d.name ?? id;
    const htmlUrl = downloadUrl(d.htmlCode);
    let htmlPath = null;
    let pngPath = null;

    if (htmlUrl) {
      const buf = await fetchBinary(htmlUrl, apiKey);
      htmlPath = `${id}.html`;
      await writeFile(join(OUT_DIR, htmlPath), buf);
    }

    let shot = d.screenshot;
    if (shot && typeof shot === "object" && !shot.downloadUrl) {
      shot = null;
    }
    const pngUrl = downloadUrl(shot);
    if (pngUrl) {
      const buf = await fetchBinary(pngUrl, apiKey);
      pngPath = `${id}.png`;
      await writeFile(join(OUT_DIR, pngPath), buf);
    }

    manifest.screens.push({
      screenId: id,
      title,
      name: d.name ?? null,
      deviceType: d.deviceType ?? null,
      width: d.width ?? null,
      height: d.height ?? null,
      htmlFile: htmlPath,
      pngFile: pngPath,
    });
  }

  await writeFile(join(OUT_DIR, "manifest.json"), JSON.stringify(manifest, null, 2), "utf8");
  console.log(`Wrote ${manifest.screens.length} screens to ${OUT_DIR}`);
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
