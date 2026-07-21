import { mkdir, readFile, writeFile } from "node:fs/promises";
import { homedir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const SOURCE_URL = "https://news.hada.io/new";
const HEADLINE_COUNT = 10;
const MAX_RESPONSE_BYTES = 2 * 1024 * 1024;

export async function refresh(fetchImpl = fetch) {
  const response = await fetchImpl(SOURCE_URL, {
    headers: {
      Accept: "text/html",
      "User-Agent": "claude-geeknews-spinner",
    },
    signal: AbortSignal.timeout(10_000),
  });
  if (!response.ok) {
    throw new Error(`GeekNews returned HTTP ${response.status}`);
  }
  const contentLength = Number(response.headers.get("content-length"));
  if (contentLength > MAX_RESPONSE_BYTES) {
    throw new Error("GeekNews response exceeds 2 MiB");
  }
  const body = Buffer.from(await response.arrayBuffer());
  if (body.length > MAX_RESPONSE_BYTES) {
    throw new Error("GeekNews response exceeds 2 MiB");
  }
  const headlines = parseTopics(body.toString("utf8"));
  if (headlines.length === 0) {
    throw new Error("GeekNews latest page contained no usable topics");
  }

  const path = settingsPath();
  const settings = await readSettings(path);
  settings.spinnerVerbs = { mode: "append", verbs: headlines };
  await mkdir(dirname(path), { recursive: true, mode: 0o700 });
  await writeFile(path, `${JSON.stringify(settings, null, 2)}\n`, { mode: 0o600 });
  return headlines.length;
}

export function parseTopics(html) {
  const starts = [...html.matchAll(/<div\b[^>]*\bclass\s*=\s*["'][^"']*\btopic_row\b[^"']*["'][^>]*>/gi)];
  const headlines = [];
  for (let index = 0; index < starts.length && headlines.length < HEADLINE_COUNT; index += 1) {
    const start = starts[index];
    const row = html.slice(start.index, starts[index + 1]?.index);
    const id = row.match(/\bdata-topic-state-id\s*=\s*["']([^"']+)["']/i)?.[1];
    const title = findText(row, "h2", "topic-title-heading");
    if (!id || !title) continue;
    const summary = findText(row, "div", "topicdesc");
    headlines.push(formatHeadline({
      title,
      summary,
      url: `https://news.hada.io/topic?id=${encodeURIComponent(id)}`,
    }));
  }
  return headlines;
}

export function formatHeadline({ title, summary, url }) {
  const text = summary ? `${title} - ${summary}` : title;
  let link;
  try {
    link = new URL(url);
  } catch {
    return text;
  }
  if (link.protocol !== "https:" && link.protocol !== "http:") return text;
  return `\x1b]8;;${link}\x07${text}\x1b]8;;\x07`;
}

export function cleanText(value) {
  return decodeEntities(value)
    .replace(/<[^>]*>/g, " ")
    .replace(/[\u0000-\u001F\u007F-\u009F\u202A-\u202E\u2066-\u2069]/g, "")
    .replace(/[\u00B7\u318D]/g, "-")
    .replace(/\s+/g, " ")
    .trim();
}

function findText(row, tag, className) {
  const pattern = new RegExp(`<${tag}\\b[^>]*\\bclass\\s*=\\s*["'][^"']*\\b${className}\\b[^"']*["'][^>]*>([\\s\\S]*?)<\\/${tag}>`, "i");
  const match = row.match(pattern);
  return match ? cleanText(match[1]) : "";
}

function decodeEntities(value) {
  return value.replace(/&(#x[\da-f]+|#\d+|amp|lt|gt|quot|apos);/gi, (_, entity) => {
    if (entity[0] === "#") {
      const hexadecimal = entity[1].toLowerCase() === "x";
      const codePoint = Number.parseInt(entity.slice(hexadecimal ? 2 : 1), hexadecimal ? 16 : 10);
      return Number.isInteger(codePoint) && codePoint >= 0 && codePoint <= 0x10FFFF
        ? String.fromCodePoint(codePoint)
        : "";
    }
    return { amp: "&", lt: "<", gt: ">", quot: '"', apos: "'" }[entity.toLowerCase()];
  });
}

function settingsPath() {
  return join(process.env.CLAUDE_CONFIG_DIR || homedir(), process.env.CLAUDE_CONFIG_DIR ? "settings.json" : ".claude/settings.json");
}

async function readSettings(path) {
  try {
    const data = await readFile(path, "utf8");
    if (Buffer.byteLength(data) > 5 * 1024 * 1024) {
      throw new Error("Claude settings exceed 5 MiB");
    }
    const settings = JSON.parse(data);
    if (!settings || Array.isArray(settings) || typeof settings !== "object") {
      throw new Error("Claude settings must be a JSON object");
    }
    return settings;
  } catch (error) {
    if (error.code === "ENOENT") return {};
    throw error;
  }
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  refresh().catch((error) => {
    console.error(`claude-geeknews-spinner: ${error.message}`);
    process.exitCode = 1;
  });
}
