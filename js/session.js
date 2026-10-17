/* ─────────────────────────────────────────────────────────────
   session.js — idle sign-out for logged-in pages
   Include on employer.html and applicant.html (NOT login/index).

   - Counts real user input (mouse, keys, touch, scroll) as activity.
   - Tells the server about activity (throttled) so the server-side
     idle clock matches what the person is actually doing.
   - Warns 60s before sign-out, then logs out and returns to login.
   - Any 401 from /api/* sends the person back to login.
   ───────────────────────────────────────────────────────────── */
(function () {
  'use strict';

  var LOGIN_URL = '/login.html?reason=idle';
  var DEFAULT_IDLE_SECONDS = 15 * 60;   // overwritten by /api/auth/me
  var WARN_SECONDS = 60;
  var PING_EVERY_MS = 30 * 1000;
  var CHECK_EVERY_MS = 5 * 1000;
  var STORE_KEY = 'ic_last_activity';   // shared across tabs

  var origFetch = window.fetch.bind(window);
  var idleMs = DEFAULT_IDLE_SECONDS * 1000;
  var lastActivity = Date.now();
  var lastPing = 0;
  var banner = null;
  var signingOut = false;

  /* ── 401 handling: wrap fetch once ───────────────────────── */
  window.fetch = function (input, init) {
    return origFetch(input, init).then(function (res) {
      try {
        var url = typeof input === 'string' ? input : (input && input.url) || '';
        var path = new URL(url, location.origin).pathname;
        var isAuthForm = path === '/api/auth/login' || path === '/api/auth/signup';
        if (res.status === 401 && path.indexOf('/api/') === 0 && !isAuthForm) {
          goToLogin();
        }
      } catch (e) { /* never break the app over this */ }
      return res;
    });
  };

  function goToLogin() {
    if (signingOut) return;
    signingOut = true;
    location.replace(LOGIN_URL);
  }

  /* ── Activity tracking ───────────────────────────────────── */
  function readShared() {
    try {
      var v = parseInt(localStorage.getItem(STORE_KEY), 10);
      return isNaN(v) ? 0 : v;
    } catch (e) { return 0; }
  }

  function markActive() {
    if (signingOut) return;
    var now = Date.now();
    lastActivity = now;
    try { localStorage.setItem(STORE_KEY, String(now)); } catch (e) { /* ignore */ }
    hideBanner();
    if (now - lastPing >= PING_EVERY_MS) {
      lastPing = now;
      origFetch('/api/auth/activity', {
        method: 'POST', credentials: 'same-origin', keepalive: true
      }).then(function (res) {
        if (res.status === 401) goToLogin();
      }).catch(function () { /* offline: the next check/ping will retry */ });
    }
  }

  ['pointerdown', 'keydown', 'touchstart', 'wheel', 'scroll', 'mousemove']
    .forEach(function (evt) {
      window.addEventListener(evt, markActive, { passive: true, capture: true });
    });

  /* ── Warning banner ──────────────────────────────────────── */
  function showBanner(secondsLeft) {
    if (!banner) {
      banner = document.createElement('div');
      banner.setAttribute('role', 'alert');
      banner.style.cssText =
        'position:fixed;left:50%;bottom:20px;transform:translateX(-50%);z-index:2147483647;' +
        'max-width:calc(100vw - 32px);padding:12px 18px;border-radius:12px;' +
        'background:#1f2937;color:#fff;font:600 14px/1.4 system-ui,sans-serif;' +
        'box-shadow:0 8px 24px rgba(0,0,0,.3);text-align:center;';
      document.body.appendChild(banner);
    }
    banner.textContent =
      'Still there? You will be signed out in ' + secondsLeft +
      's for inactivity. Tap anywhere to stay signed in.';
  }

  function hideBanner() {
    if (banner) { banner.remove(); banner = null; }
  }

  /* ── Idle check ──────────────────────────────────────────── */
  function check() {
    if (signingOut) return;
    // Activity in another tab counts too.
    var last = Math.max(lastActivity, readShared());
    var idleFor = Date.now() - last;
    var remainingMs = idleMs - idleFor;

    if (remainingMs <= 0) {
      signingOut = true;
      origFetch('/api/auth/logout', { method: 'POST', credentials: 'same-origin', keepalive: true })
        .catch(function () {})
        .then(function () { location.replace(LOGIN_URL); });
    } else if (remainingMs <= WARN_SECONDS * 1000) {
      showBanner(Math.ceil(remainingMs / 1000));
    } else {
      hideBanner();
    }
  }

  /* ── Start ───────────────────────────────────────────────── */
  origFetch('/api/auth/me', { credentials: 'same-origin' })
    .then(function (res) {
      if (res.status === 401) { goToLogin(); return null; }
      return res.ok ? res.json() : null;
    })
    .then(function (data) {
      if (data && data.idle_timeout_seconds > 0) {
        idleMs = data.idle_timeout_seconds * 1000;
      } else if (data && data.idle_timeout_seconds === 0) {
        return; // server has idle timeout turned off; nothing to enforce here
      }
      markActive();
      setInterval(check, CHECK_EVERY_MS);
    })
    .catch(function () {
      markActive();
      setInterval(check, CHECK_EVERY_MS);
    });
})();
