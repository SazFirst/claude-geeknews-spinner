import assert from "node:assert/strict";
import test from "node:test";

import { cleanText, formatHeadline, parseTopics } from "./refresh.mjs";

test("parses the latest GeekNews topics", () => {
  const headlines = parseTopics(`
    <div class="topic_row" data-topic-state-id="101">
      <h2 class="topic-title-heading">First &amp; Best</h2>
      <div class="topicdesc">A short <b>summary</b></div>
    </div>
    <div class="topic_row" data-topic-state-id="100">
      <h2 class="topic-title-heading">Second</h2>
    </div>`);
  assert.equal(headlines.length, 2);
  assert.match(headlines[0], /First & Best - A short summary/);
  assert.match(headlines[0], /https:\/\/news\.hada\.io\/topic\?id=101/);
});

test("removes terminal and bidi controls", () => {
  assert.equal(cleanText("  제목\x1b  사이\u202e  공백  "), "제목 사이 공백");
});

test("uses plain text when a link is not HTTP", () => {
  assert.equal(formatHeadline({ title: "Title", summary: "", url: "ftp://example.com" }), "Title");
});
