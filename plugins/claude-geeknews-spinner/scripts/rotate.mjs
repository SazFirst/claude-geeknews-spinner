import { mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { homedir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { randomUUID } from "node:crypto";

import { refreshOne } from "./refresh.mjs";

const ROTATION_INTERVAL_MS = 20_000;

export async function startRotator(sessionId, options = {}) {
  const refresh = options.refresh ?? refreshOne;
  const delay = options.delay ?? wait;
  const stateFile = options.stateFile ?? rotatorStatePath();
  const token = randomUUID();

  await writeState(stateFile, { sessionId, token });

  let index = 0;
  while (await isCurrentRotator(stateFile, sessionId, token)) {
    try {
      await refresh(index);
      index += 1;
    } catch (error) {
      console.error(`claude-geeknews-spinner: ${error.message}`);
    }
    await delay(ROTATION_INTERVAL_MS);
  }
}

export async function stopRotator(sessionId, stateFile = rotatorStatePath()) {
  try {
    const state = JSON.parse(await readFile(stateFile, "utf8"));
    if (state.sessionId === sessionId) {
      await rm(stateFile, { force: true });
    }
  } catch (error) {
    if (error.code !== "ENOENT") throw error;
  }
}

async function isCurrentRotator(stateFile, sessionId, token) {
  try {
    const state = JSON.parse(await readFile(stateFile, "utf8"));
    return state.sessionId === sessionId && state.token === token;
  } catch (error) {
    if (error.code === "ENOENT") return false;
    throw error;
  }
}

async function writeState(stateFile, state) {
  await mkdir(dirname(stateFile), { recursive: true, mode: 0o700 });
  await writeFile(stateFile, `${JSON.stringify(state)}\n`, { mode: 0o600 });
}

function rotatorStatePath() {
  const configDir = process.env.CLAUDE_CONFIG_DIR || join(homedir(), ".claude");
  return join(configDir, "claude-geeknews-spinner", "rotator.json");
}

function wait(durationMs) {
  return new Promise((resolveDelay) => setTimeout(resolveDelay, durationMs));
}

async function readHookInput() {
  const input = await readFile(0, "utf8");
  const { session_id: sessionId } = JSON.parse(input);
  if (typeof sessionId !== "string" || sessionId.length === 0) {
    throw new Error("hook input did not include a session_id");
  }
  return sessionId;
}

async function main() {
  const action = process.argv[2];
  const sessionId = await readHookInput();
  if (action === "start") {
    await startRotator(sessionId);
    return;
  }
  if (action === "stop") {
    await stopRotator(sessionId);
    return;
  }
  throw new Error(`unknown rotator action: ${action}`);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main().catch((error) => {
    console.error(`claude-geeknews-spinner: ${error.message}`);
    process.exitCode = 1;
  });
}
