import { mkdir, readFile, writeFile } from "node:fs/promises";
import { homedir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const SOURCE_URL = "https://news.hada.io/new";
const LOOKBACK_MS = 24 * 60 * 60 * 1_000;
const MIN_HEADLINE_COUNT = 10;
const MAX_RESPONSE_BYTES = 2 * 1024 * 1024;

export async function refresh(fetchImpl = fetch) {
  const headlines = await fetchHeadlines(fetchImpl);
  await writeSpinnerVerbs(headlines);
  return headlines.length;
}

export async function refreshOne(index, fetchImpl = fetch) {
  const headlines = await fetchHeadlines(fetchImpl);
  const headline = headlineAt(headlines, index);
  await writeSpinnerVerbs([headline]);
  return headline;
}

export function headlineAt(headlines, index) {
  if (headlines.length === 0) {
    throw new Error("GeekNews latest page contained no usable topics");
  }
  return headlines[index % headlines.length];
}

export async function fetchHeadlines(fetchImpl = fetch, now = Date.now()) {
  const cutoff = now - LOOKBACK_MS;
  const topics = [];

  for (let page = 1; ; page += 1) {
    const pageTopics = await fetchTopicsPage(page, fetchImpl);
    if (pageTopics.length === 0) break;
    topics.push(...pageTopics);

    const candidates = selectCandidateTopics(topics, cutoff);
    const hasOlderTopic = pageTopics.some((topic) => topic.timestamp < cutoff);
    if (hasOlderTopic && candidates.length >= MIN_HEADLINE_COUNT) {
      return candidates.map(formatHeadline);
    }
  }

  const candidates = selectCandidateTopics(topics, cutoff);
  headlineAt(candidates, 0);
  return candidates.map(formatHeadline);
}

export function selectCandidateTopics(topics, cutoff) {
  const recentCount = topics.filter((topic) => topic.timestamp >= cutoff).length;
  return topics.slice(0, Math.max(recentCount, MIN_HEADLINE_COUNT));
}

async function fetchTopicsPage(page, fetchImpl) {
  const url = page === 1 ? SOURCE_URL : `${SOURCE_URL}?page=${page}`;
  const response = await fetchImpl(url, {
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
  return parseTopicRows(body.toString("utf8"));
}

async function writeSpinnerVerbs(verbs) {
  const path = settingsPath();
  const settings = await readSettings(path);
  settings.spinnerVerbs = { mode: "replace", verbs };
  await mkdir(dirname(path), { recursive: true, mode: 0o700 });
  await writeFile(path, `${JSON.stringify(settings, null, 2)}\n`, { mode: 0o600 });
}

export function parseTopics(html) {
  const starts = [...html.matchAll(/<div\b[^>]*\bclass\s*=\s*["'][^"']*\btopic_row\b[^"']*["'][^>]*>/gi)];
  const topics = [];
  for (let index = 0; index < starts.length; index += 1) {
    const start = starts[index];
    const row = html.slice(start.index, starts[index + 1]?.index);
    const id = row.match(/\bdata-topic-state-id\s*=\s*["']([^"']+)["']/i)?.[1];
    const title = findText(row, "h2", "topic-title-heading");
    if (!id || !title) continue;
    const summary = findText(row, "div", "topicdesc");
    const points = Number.parseInt(
      row.match(/<span\b[^>]*\bid\s*=\s*["']tp[^"']+["'][^>]*>\s*(\d+)\s*<\/span>\s*points?\b/i)?.[1] ?? "0",
      10,
    );
    const timestamp = Number.parseInt(
      row.match(/<time\b[^>]*\bdata-timestamp\s*=\s*["'](\d+)["']/i)?.[1] ?? "0",
      10,
    ) * 1_000;
    if (!Number.isSafeInteger(timestamp) || timestamp <= 0) continue;
    topics.push({
      title,
      summary,
      points: Number.isSafeInteger(points) && points >= 0 ? points : 0,
      timestamp,
      url: `https://news.hada.io/topic?id=${encodeURIComponent(id)}`,
    });
  }
  return topics;
}

export function parseTopicRows(html) {
  return parseTopics(html);
}

export function formatHeadline({ title, summary, points = 0, url }) {
  const text = summary ? `[${points}p] ${title} - ${summary}` : `[${points}p] ${title}`;
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
