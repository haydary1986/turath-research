/*
 * Service worker for the Al-Turath University scientific-affairs portal.
 *
 * - Precaches a small app shell (offline page, icons, crest, manifest).
 * - Serves hashed build assets (/_next/static) and brand images cache-first.
 * - Page navigations always go to the network; when offline they fall back to
 *   /offline.html. Pages are never stored, so no signed-in screen is cached.
 * - Never touches /api: API responses (public or authenticated) are not cached.
 * - No push notifications.
 */
const VERSION = "v1";
const SHELL_CACHE = `turath-shell-${VERSION}`;
const STATIC_CACHE = `turath-static-${VERSION}`;
const OFFLINE_URL = "/offline.html";

const SHELL_FILES = [
  OFFLINE_URL,
  "/manifest.webmanifest",
  "/brand/crest.png",
  "/icons/icon-192.png",
  "/icons/icon-512.png",
  "/icons/favicon-32.png",
];

const STATIC_PREFIXES = ["/_next/static/", "/icons/", "/brand/"];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches
      .open(SHELL_CACHE)
      .then((cache) => cache.addAll(SHELL_FILES))
      .then(() => self.skipWaiting())
  );
});

self.addEventListener("activate", (event) => {
  const keep = new Set([SHELL_CACHE, STATIC_CACHE]);
  event.waitUntil(
    caches
      .keys()
      .then((names) => Promise.all(names.filter((name) => !keep.has(name)).map((name) => caches.delete(name))))
      .then(() => self.clients.claim())
  );
});

function isStaticAsset(url) {
  return url.pathname === "/favicon.ico" || STATIC_PREFIXES.some((prefix) => url.pathname.startsWith(prefix));
}

async function cacheFirst(request) {
  const cached = await caches.match(request);
  if (cached) return cached;
  const response = await fetch(request);
  if (response.ok && response.type === "basic") {
    const copy = response.clone();
    caches.open(STATIC_CACHE).then((cache) => cache.put(request, copy));
  }
  return response;
}

async function networkWithOfflineFallback(request) {
  try {
    return await fetch(request);
  } catch (error) {
    const offline = await caches.match(OFFLINE_URL);
    if (offline) return offline;
    throw error;
  }
}

self.addEventListener("fetch", (event) => {
  const { request } = event;
  if (request.method !== "GET") return;

  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;
  // API data is always live and never stored, public or not.
  if (url.pathname.startsWith("/api/")) return;
  if (request.headers.has("Authorization")) return;

  if (request.mode === "navigate") {
    event.respondWith(networkWithOfflineFallback(request));
    return;
  }
  if (isStaticAsset(url)) {
    event.respondWith(cacheFirst(request));
  }
  // Everything else (React Server Component payloads, images via /_next/image…) goes to the network untouched.
});
