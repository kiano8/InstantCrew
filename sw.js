// Instant Crew - Service Worker for PWA Add-to-Home-Screen Support
const CACHE_NAME = 'instantcrew-v2';
const CORE_ASSETS = [
    './',
    './index.html',
    './css/styles.css',
    './images/icon-192.png',
    './images/icon-512.png',
    './images/icon-maskable-192.png',
    './images/icon-maskable-512.png',
    './images/apple-touch-icon.png',
    './images/Instalogo.png',
    './images/logo.png',
    './images/instadelivery.png',
    './manifest.json'
];

self.addEventListener('install', (event) => {
    event.waitUntil(
        caches.open(CACHE_NAME).then((cache) => {
            return cache.addAll(CORE_ASSETS).catch((err) => {
                console.warn('[SW] Pre-caching asset warning:', err);
            });
        }).then(() => self.skipWaiting())
    );
});

self.addEventListener('activate', (event) => {
    event.waitUntil(
        caches.keys().then((keys) => {
            return Promise.all(
                keys.map((key) => {
                    if (key !== CACHE_NAME) {
                        return caches.delete(key);
                    }
                })
            );
        }).then(() => self.clients.claim())
    );
});

// Web Push from the server (internal/push) arrives here even when no InstantCrew
// tab is open. Its tag matches the open page's own notification for the same
// event, so the two replace each other instead of stacking.
self.addEventListener('push', (event) => {
    let data = {};
    try { data = event.data ? event.data.json() : {}; } catch (_) { }
    event.waitUntil(
        self.registration.showNotification(data.title || 'Instant Crew', {
            body: data.body || '',
            icon: '/images/logo.png',
            badge: '/images/Instalogo.png',
            tag: data.tag || undefined,
            renotify: Boolean(data.tag),
            vibrate: [120, 60, 120],
            data: { url: data.url || '/employer.html' }
        })
    );
});

// Tapping a system notification focuses an open InstantCrew tab, or opens the
// page that raised it. An open tab is told the URL so it can show the shift.
self.addEventListener('notificationclick', (event) => {
    event.notification.close();
    const target = new URL((event.notification.data && event.notification.data.url) || './', self.location.origin);
    event.waitUntil(
        self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clients) => {
            const samePage = clients.find((client) => new URL(client.url).pathname === target.pathname);
            if (samePage) {
                samePage.postMessage({ type: 'notification-click', url: target.href });
                return samePage.focus();
            }
            return self.clients.openWindow(target.href);
        })
    );
});

self.addEventListener('fetch', (event) => {
    // Only handle GET requests and http/https schemes
    if (event.request.method !== 'GET' || !event.request.url.startsWith('http')) {
        return;
    }
    // Let long-lived event streams (live admin updates) bypass the worker.
    if ((event.request.headers.get('Accept') || '').includes('text/event-stream')) {
        return;
    }

    event.respondWith(
        fetch(event.request)
            .then((networkResponse) => {
                return networkResponse;
            })
            .catch(() => {
                return caches.match(event.request).then((cachedResponse) => {
                    if (cachedResponse) {
                        return cachedResponse;
                    }
                    if (event.request.mode === 'navigate') {
                        return caches.match('./index.html');
                    }
                });
            })
    );
});
