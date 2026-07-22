import assert from "node:assert/strict";
import test from "node:test";

import { cleanText, fetchHeadlines, formatHeadline, headlineAt, parseTopics, selectCandidateTopics } from "./refresh.mjs";

test("parses the latest GeekNews topics", () => {
  const headlines = parseTopics(`
    <div class="topic_row" data-topic-state-id="101">
      <h2 class="topic-title-heading">First &amp; Best</h2>
      <div class="topicdesc">A short <b>summary</b></div>
      <div class="topicinfo"><span id="tp101">12</span> points <time data-timestamp="1700000000">now</time></div>
    </div>
    <div class="topic_row" data-topic-state-id="100">
      <h2 class="topic-title-heading">Second</h2>
      <div class="topicinfo"><span id="tp100">1</span> point <time data-timestamp="1699999999">now</time></div>
    </div>`);
  assert.equal(headlines.length, 2);
  assert.equal(headlines[0].points, 12);
  assert.match(formatHeadline(headlines[0]), /\[12p\] First & Best - A short summary/);
  assert.match(formatHeadline(headlines[0]), /https:\/\/news\.hada\.io\/topic\?id=101/);
});

test("removes terminal and bidi controls", () => {
  assert.equal(cleanText("  제목\x1b  사이\u202e  공백  "), "제목 사이 공백");
});

test("uses plain text when a link is not HTTP", () => {
  assert.equal(formatHeadline({ title: "Title", summary: "", points: 10, url: "ftp://example.com" }), "[10p] Title");
});

test("selects a headline by cycling through the latest topics", () => {
  const headlines = ["first", "second", "third"];
  assert.equal(headlineAt(headlines, 0), "first");
  assert.equal(headlineAt(headlines, 4), "second");
  assert.throws(() => headlineAt([], 0), /no usable topics/);
});

test("keeps every recent topic and backfills to ten candidates", () => {
  const now = 1_700_000_000_000;
  const cutoff = now - 24 * 60 * 60 * 1_000;
  const topics = Array.from({ length: 12 }, (_, index) => ({
    title: `Topic ${index}`,
    timestamp: index < 3 ? now - index * 60_000 : cutoff - (index - 2) * 60_000,
  }));
  assert.equal(selectCandidateTopics(topics, cutoff).length, 10);

  const recentTopics = Array.from({ length: 11 }, (_, index) => ({
    title: `Recent ${index}`,
    timestamp: now - index * 60_000,
  }));
  assert.equal(selectCandidateTopics(recentTopics, cutoff).length, 11);
});

test("loads another page to include every topic from the last day", async () => {
  const now = 1_700_000_000_000;
  const firstPage = Array.from({ length: 20 }, (_, index) => topicHtml(
    100 + index,
    Math.floor((now - index * 60_000) / 1_000),
  )).join("");
  const secondPage = topicHtml(200, Math.floor((now - 25 * 60 * 60 * 1_000) / 1_000));
  const requests = [];
  const fetchImpl = async (url) => {
    requests.push(url);
    return htmlResponse(url.endsWith("?page=2") ? secondPage : firstPage);
  };

  const headlines = await fetchHeadlines(fetchImpl, now);
  assert.equal(headlines.length, 20);
  assert.equal(requests.length, 2);
  assert.match(requests[1], /\?page=2$/);
});

function topicHtml(id, timestamp) {
  return `<div class="topic_row" data-topic-state-id="${id}">
    <h2 class="topic-title-heading">Topic ${id}</h2>
    <div class="topicdesc">Summary ${id}</div>
    <div class="topicinfo"><span id="tp${id}">10</span> points <time data-timestamp="${timestamp}">now</time></div>
  </div>`;
}

function htmlResponse(html) {
  return {
    ok: true,
    headers: { get: () => null },
    arrayBuffer: async () => Buffer.from(html),
  };
}
