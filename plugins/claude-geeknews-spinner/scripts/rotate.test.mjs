import assert from "node:assert/strict";
import test from "node:test";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { parseSessionId, startRotator, stopRotator } from "./rotate.mjs";

test("reads the Claude Code session identifier from hook input", () => {
  assert.equal(parseSessionId('{"session_id":"session-1"}'), "session-1");
  assert.throws(() => parseSessionId('{}'), /did not include a session_id/);
});

test("rotator updates the selected headline and stops when requested", async (t) => {
  const directory = await mkdtemp(join(tmpdir(), "claude-geeknews-spinner-"));
  const stateFile = join(directory, "rotator.json");
  t.after(() => rm(directory, { force: true, recursive: true }));
  const selected = [];
  let releaseDelay;
  let markFirstRefresh;
  const firstRefresh = new Promise((resolve) => {
    markFirstRefresh = resolve;
  });
  const delay = () => new Promise((resolve) => {
    releaseDelay = resolve;
  });

  const running = startRotator("session-1", {
    stateFile,
    refresh: async (index) => {
      selected.push(index);
      markFirstRefresh();
    },
    delay,
  });

  await firstRefresh;
  assert.deepEqual(selected, [0]);
  const state = JSON.parse(await readFile(stateFile, "utf8"));
  assert.equal(state.sessionId, "session-1");
  assert.equal(typeof state.token, "string");
  while (!releaseDelay) await new Promise((resolve) => setImmediate(resolve));

  await stopRotator("session-1", stateFile);
  releaseDelay();
  await running;
  assert.deepEqual(selected, [0]);
});
