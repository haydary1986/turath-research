#!/usr/bin/env node
/**
 * Real-browser i18n check: opens every public page and every portal page in both
 * locales and fails when
 *   - a page crashes (HTTP >= 500, an uncaught exception, the Next error screen),
 *   - the browser logs a console error or a next-intl MISSING_MESSAGE,
 *   - text that looks like an untranslated message key is rendered,
 *   - <html lang/dir> does not match the locale,
 *   - on /en pages, UI chrome (headings, buttons, nav, labels, aria-labels,
 *     placeholders...) contains Arabic letters,
 *   - on /ar pages, UI chrome contains Latin words outside the allowlist.
 * Text marked as data (inside an element with `data-bidi` or a `lang` attribute
 * other than <html>) is never treated as chrome: that is how the app labels names,
 * titles and other database text (see src/components/i18n/bidi.tsx).
 *
 * Usage (from frontend/):
 *   I18N_ADMIN_EMAIL=admin@example.iq I18N_ADMIN_PASSWORD=... npm run test:i18n
 * Environment:
 *   I18N_BASE_URL       site to test (default http://localhost:3001)
 *   I18N_ADMIN_EMAIL    admin account used for the portal pages (required)
 *   I18N_ADMIN_PASSWORD its password (required)
 *   CHROME_PATH         Chrome executable (default: the macOS system Chrome)
 *   I18N_ONLY           only check paths containing this substring
 *   I18N_HEADED=1       show the browser
 */
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright-core";

const BASE = (process.env.I18N_BASE_URL || "http://localhost:3001").replace(/\/+$/, "");
const EMAIL = process.env.I18N_ADMIN_EMAIL;
const PASSWORD = process.env.I18N_ADMIN_PASSWORD;
const CHROME = process.env.CHROME_PATH || "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";
const ONLY = process.env.I18N_ONLY || "";
const LOCALES = ["ar", "en"];
const NAV_TIMEOUT = 45_000;
const SETTLE_MS = 600;

/** A path that matches no page: it must render the localized 404 page with status 404. */
const NOT_FOUND_PATH = "/journals/no-such-page";

const PUBLIC_PATHS = [
  "/",
  "/news",
  "/news?type=circular",
  "/journals",
  "/journals/check",
  "/journals/predatory",
  "/journals/predatory/list",
  "/journals/predatory/list/journals",
  "/journals/predatory/list/publishers",
  "/journals/predatory/list/hijacked",
  "/journals/predatory/list/journals?letter=a",
  "/journals/guide/warning-signs",
  "/journals/guide/hijacked-journals",
  "/journals/guide/verify-indexing",
  "/journals/guide/predatory-conferences",
  NOT_FOUND_PATH,
  "/auth/login",
];

const PORTAL_PATHS = [
  "/portal",
  "/portal/requests",
  "/portal/new/research_plan",
  "/portal/new/journal_verification",
  "/portal/new/appreciation_letter",
  "/portal/activities",
  "/portal/audit",
  "/portal/import",
  "/portal/journal-check",
  "/portal/journals",
  "/portal/predatory-journals",
  "/portal/profile",
  "/portal/reports",
  "/portal/responsibilities",
  "/portal/site",
  "/portal/templates",
  "/portal/users",
  "/dashboard",
  "/insights/trends",
  "/publications",
  "/researchers",
  "/search",
];

/**
 * Latin terms allowed in Arabic UI chrome: product, index and standard names that
 * have no Arabic form in Iraqi academic usage, plus units and file formats.
 */
const AR_ALLOWLIST = [
  "Web of Science", "Google Scholar", "Google Workspace", "Stop Predatory Journals", "FS 003", "X (Twitter)",
  "ORCID", "iD", "DOI", "ISSN", "Scopus", "Google", "ResearchGate", "LinkedIn", "Excel", "PDF", "PWA",
  "OpenAlex", "Crossref", "DOAJ", "Gmail", "CiteScore", "h-index", "Beall", "Beall's",
  "PNG", "JPG", "JPEG", "xlsx", "KB", "MB", "Q1", "Q2", "Q3", "Q4",
  "Facebook", "Instagram", "YouTube", "Telegram", "TikTok", "WhatsApp",
  // Names of external sources and terms glossed in parentheses after their Arabic form.
  "Predatory Journals", "Beall's list", "Quartile", "Impact Factor", "07XXXXXXXXX",
  "Hijacked Journal", "Coverage", "Sources", "Source list", "Master Journal List", "Journal Citation Reports",
  "Elsevier", "Clarivate", "SCIE", "SSCI", "AHCI", "ESCI", "Yahoo", "DeepSeek", "IP",
];

const ARABIC = /[ء-يٱ-ۓ]/;
// A rendered string is an untranslated key when it is "<namespace>.<key>..." for a real namespace.
const MESSAGES_DIR = join(dirname(fileURLToPath(import.meta.url)), "../src/messages");
const NAMESPACES = new Set(
  ["ar.json", "public/ar.json"].flatMap((f) => Object.keys(JSON.parse(readFileSync(join(MESSAGES_DIR, f), "utf8"))))
);
const MESSAGE_KEY = /^([a-zA-Z]+)(\.[a-zA-Z0-9_]+)+$/;
const ERROR_SCREEN = /Application error|This page couldn.t load|Internal Server Error|Unhandled Runtime Error/i;

function fail(message) {
  console.error(message);
  process.exit(2);
}

async function signIn() {
  const res = await fetch(`${BASE}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email: EMAIL, password: PASSWORD }),
  });
  const json = await res.json().catch(() => null);
  if (!res.ok || !json?.data?.token) fail(`Sign-in failed (HTTP ${res.status}): ${json?.error ?? "no token"}`);
  return json.data;
}

async function apiGet(path, token) {
  const res = await fetch(`${BASE}${path}`, { headers: { Authorization: `Bearer ${token}` } });
  if (!res.ok) return null;
  return (await res.json().catch(() => null))?.data ?? null;
}

/** Adds the record pages (a request, its letter, a researcher, an announcement, a journal entry) that exist locally. */
async function dynamicPaths(token) {
  const publicPaths = [];
  const portalPaths = [];
  const announcements = await apiGet("/api/v1/public/announcements?limit=1", token);
  if (announcements?.[0]) publicPaths.push(`/news/${announcements[0].id}`);
  const entries = await apiGet("/api/v1/public/journals/entries?type=hijacked&limit=1", token);
  const entry = Array.isArray(entries) ? entries[0] : entries?.items?.[0];
  if (entry?.slug) publicPaths.push(`/journals/predatory/${entry.slug}`);

  const requests = (await apiGet("/api/v1/portal/requests?limit=50", token)) ?? [];
  const withKind = (kind) => requests.find((r) => r.kind === kind);
  for (const r of [withKind("research_plan"), withKind("journal_verification"), withKind("appreciation_letter")]) {
    if (r) portalPaths.push(`/portal/requests/${r.id}`, `/portal/requests/${r.id}/print`);
  }
  const editable = requests.find((r) => r.status === "draft" || r.status === "returned");
  if (editable) portalPaths.push(`/portal/requests/${editable.id}/edit`);
  for (const r of requests.filter((x) => x.status === "issued").slice(0, 1)) {
    const detail = await apiGet(`/api/v1/portal/requests/${r.id}`, token);
    if (detail?.letter_id) {
      portalPaths.push(`/portal/letters/${detail.letter_id}`);
      publicPaths.push(`/verify/${detail.letter_id}`);
    }
  }
  const researchers = (await apiGet("/api/v1/researchers?limit=1", token)) ?? [];
  if (researchers[0]) portalPaths.push(`/researchers/${researchers[0].id}`, `/researchers/${researchers[0].id}/insights`);
  return { publicPaths, portalPaths };
}

function escapeRegExp(s) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

const ALLOW_RE = new RegExp(
  `(?<![A-Za-z])(${[...AR_ALLOWLIST].sort((a, b) => b.length - a.length).map(escapeRegExp).join("|")})(?![A-Za-z])`,
  "g"
);

/** Latin words left in an Arabic chrome string once URLs, emails and allowlisted terms are removed. */
function latinWords(text) {
  const stripped = text
    .replace(/https?:\/\/\S*/g, " ")
    .replace(/\b[\w.+-]+@[\w-]+(\.[\w-]+)+\b/g, " ")
    .replace(/\b(?:[a-z0-9-]+\.)+[a-z]{2,}(?:\/\S*)?/gi, " ")
    .replace(/(^|[\s(])\/[\w\-/]+/g, " ")
    .replace(/@[\w-]+(\.[\w-]+)+/g, " ")
    .replace(/\b[\w-]+(\.[\w-]+)*\.(iq|com|org|net|edu)\b/g, " ")
    .replace(ALLOW_RE, " ");
  return stripped.match(/[A-Za-z][A-Za-z'-]*/g) ?? [];
}

/** Runs in the page: every chrome text (text nodes and labelling attributes) outside data elements. */
function collectChrome() {
  const DATA = "script,style,noscript,textarea,template,svg,[data-bidi],[lang]:not(html)";
  const ATTRS = ["aria-label", "placeholder", "title", "alt"];
  const describe = (el) => {
    const parts = [];
    for (let e = el; e && e !== document.body && parts.length < 3; e = e.parentElement) {
      const cls = typeof e.className === "string" && e.className ? `.${e.className.split(/\s+/).slice(0, 2).join(".")}` : "";
      parts.unshift(`${e.tagName.toLowerCase()}${e.id ? `#${e.id}` : ""}${cls}`);
    }
    return parts.join(" > ");
  };
  const out = [];
  const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
  while (walker.nextNode()) {
    const node = walker.currentNode;
    const text = (node.nodeValue || "").replace(/\s+/g, " ").trim();
    const el = node.parentElement;
    if (!text || !el || el.closest(DATA)) continue;
    if (typeof el.checkVisibility === "function" && !el.checkVisibility() && !el.closest(".sr-only,option")) continue;
    out.push({ text, where: describe(el) });
  }
  for (const el of document.body.querySelectorAll(ATTRS.map((a) => `[${a}]`).join(","))) {
    if (el.closest(DATA)) continue;
    for (const attr of ATTRS) {
      const value = (el.getAttribute(attr) || "").trim();
      if (value) out.push({ text: value, where: `${describe(el)} @${attr}` });
    }
  }
  return {
    lang: document.documentElement.getAttribute("lang"),
    dir: document.documentElement.getAttribute("dir"),
    bodyText: document.body.innerText.slice(0, 2000),
    chrome: out,
  };
}

async function checkPage(context, locale, path) {
  const url = `${BASE}/${locale}${path === "/" ? "" : path}`;
  const problems = [];
  const expect404 = path === NOT_FOUND_PATH;
  const page = await context.newPage();
  page.on("console", (msg) => {
    const text = msg.text();
    // The browser logs the 404 of the document itself on the not-found check.
    if (expect404 && /status of 404/.test(text) && msg.location()?.url?.startsWith(url)) return;
    if (msg.type() === "error" || /MISSING_MESSAGE/.test(text)) problems.push(`console ${msg.type()}: ${text.slice(0, 300)}`);
  });
  page.on("pageerror", (err) => problems.push(`uncaught exception: ${String(err.message || err).slice(0, 300)}`));
  try {
    const res = await page.goto(url, { waitUntil: "networkidle", timeout: NAV_TIMEOUT });
    if (!res) problems.push("no response");
    else if (res.status() >= 500) problems.push(`HTTP ${res.status()}`);
    else if (expect404 && res.status() !== 404) problems.push(`expected HTTP 404, got ${res.status()}`);
    await page.waitForTimeout(SETTLE_MS);
    const snap = await page.evaluate(collectChrome);
    if (ERROR_SCREEN.test(snap.bodyText)) problems.push(`error screen: ${snap.bodyText.slice(0, 200)}`);
    const wantDir = locale === "ar" ? "rtl" : "ltr";
    if (snap.lang !== locale) problems.push(`<html lang="${snap.lang}"> expected "${locale}"`);
    if (snap.dir !== wantDir) problems.push(`<html dir="${snap.dir}"> expected "${wantDir}"`);
    const seen = new Set();
    for (const { text, where } of snap.chrome) {
      let issue = null;
      const keyMatch = MESSAGE_KEY.exec(text);
      if (keyMatch && NAMESPACES.has(keyMatch[1])) {
        issue = "looks like an untranslated message key";
      } else if (locale === "en" && ARABIC.test(text)) {
        issue = "Arabic text in English chrome";
      } else if (locale === "ar") {
        const words = latinWords(text);
        if (words.length) issue = `Latin words in Arabic chrome (${[...new Set(words)].join(", ")})`;
      }
      if (!issue) continue;
      const key = `${issue}|${text}`;
      if (seen.has(key)) continue;
      seen.add(key);
      problems.push(`${issue}: "${text.slice(0, 120)}" at ${where}`);
    }
  } catch (err) {
    problems.push(`navigation failed: ${String(err.message || err).split("\n")[0]}`);
  } finally {
    await page.close();
  }
  return { url, problems };
}

async function main() {
  if (!EMAIL || !PASSWORD) fail("Set I18N_ADMIN_EMAIL and I18N_ADMIN_PASSWORD (an admin account of the site under test).");
  const session = await signIn();
  const extra = await dynamicPaths(session.token);
  const publicPaths = [...PUBLIC_PATHS, ...extra.publicPaths].filter((p) => p.includes(ONLY));
  const portalPaths = [...PORTAL_PATHS, ...extra.portalPaths].filter((p) => p.includes(ONLY));

  const browser = await chromium.launch({ executablePath: CHROME, headless: process.env.I18N_HEADED !== "1" });
  const anonymous = await browser.newContext({ serviceWorkers: "block" });
  const signedIn = await browser.newContext({ serviceWorkers: "block" });
  await signedIn.addInitScript(({ token, user }) => {
    localStorage.setItem("turath_token", token);
    localStorage.setItem("turath_user", JSON.stringify(user));
  }, session);

  const results = [];
  const report = (r) => {
    console.log(`${r.problems.length ? "FAIL" : "ok  "} ${r.url}`);
    for (const p of r.problems) console.log(`       - ${p}`);
    results.push(r);
  };
  for (const locale of LOCALES) {
    for (const path of publicPaths) report(await checkPage(anonymous, locale, path));
    for (const path of portalPaths) report(await checkPage(signedIn, locale, path));
  }
  await browser.close();

  const failed = results.filter((r) => r.problems.length);
  console.log(`\n${results.length - failed.length}/${results.length} pages passed the i18n check.`);
  process.exit(failed.length ? 1 : 0);
}

main().catch((err) => fail(err?.stack || String(err)));
