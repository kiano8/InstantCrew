/* ─────────────────────────────────────────────────────────────
   session.js — idle sign-out for logged-in pages
   Include on employer.html and applicant.html (NOT login/index).

   - Does nothing for guests. Tracking starts only when the server says
     someone is logged in, or when the page's own login/signup modal
     succeeds (detected from the /api/auth/login|signup response).
   - Counts real user input (mouse, keys, touch, scroll) as activity and
     tells the server (throttled) so its idle clock matches reality.
   - Warns 60s before sign-out. On timeout (or a 401 while logged in) it
     logs out, clears the browser-side session, reloads the SAME page as a
     guest, and opens that page's own login popup (#headerLoginBtn).
     It never navigates to login.html.
   ───────────────────────────────────────────────────────────── */
(function () {
  'use strict';

  var DEFAULT_IDLE_SECONDS = 15 * 60;   // overwritten by /api/auth/me
  var WARN_SECONDS = 60;
  var PING_EVERY_MS = 30 * 1000;
  var CHECK_EVERY_MS = 5 * 1000;
  var STORE_KEY = 'ic_last_activity';   // shared across tabs (localStorage)
  var FLAG_KEY = 'ic_idle_signout';     // survives the reload (sessionStorage)
  var SESSION_KEYS = ['ic_worker_session', 'ic_employer_session'];
  var NOTICE_TEXT = 'You were signed out because of inactivity. Please log in again.';

  var origFetch = window.fetch.bind(window);
  var idleMs = DEFAULT_IDLE_SECONDS * 1000;
  var tracking = false;     // true only while someone is logged in
  var timer = null;
  var lastActivity = Date.now();
  var lastPing = 0;
  var warnEl = null;

  function noop() {}

  /* ── Fetch wrapper: learn about login / logout / expiry ──── */
  function describe(input, init) {
    var url = '', method = 'GET';
    try {
      url = typeof input === 'string' ? input : (input && input.url) || '';
      method = String((init && init.method) || (input && input.method) || 'GET').toUpperCase();
      return { path: new URL(url, location.origin).pathname, method: method };
    } catch (e) {
      return { path: '', method: method };
    }
  }

  window.fetch = function (input, init) {
    var info = describe(input, init);
    return origFetch(input, init).then(function (res) {
      try {
        if ((info.path === '/api/auth/login' || info.path === '/api/auth/signup') &&
            info.method === 'POST' && res.ok) {
          startTracking();                       // logged in via the popup
        } else if (info.path === '/api/auth/logout') {
          stopTracking();                        // person logged out themselves
        } else if (res.status === 401 && tracking && info.path.indexOf('/api/') === 0) {
          expire();                              // server says the session is gone
        }
      } catch (e) { /* never break the app over this */ }
      return res;
    });
  };

  /* ── Start / stop ────────────────────────────────────────── */
  function startTracking() {
    origFetch('/api/auth/me', { credentials: 'same-origin' })
      .then(function (res) { return res.ok ? res.json() : null; })
      .then(function (data) {
        if (!data || !data.user) return;         // guest: nothing to track
        if (typeof data.idle_timeout_seconds === 'number') {
          idleMs = data.idle_timeout_seconds * 1000;   // 0 = idle timeout off
        }
        tracking = true;
        lastActivity = Date.now();
        lastPing = lastActivity;                 // login/me just counted as activity
        writeShared(lastActivity);
        if (!timer) timer = setInterval(check, CHECK_EVERY_MS);
      })
      .catch(noop);
  }

  function stopTracking() {
    tracking = false;
    if (timer) { clearInterval(timer); timer = null; }
    hideWarning();
  }

  /* ── Activity tracking ───────────────────────────────────── */
  function readShared() {
    try {
      var v = parseInt(localStorage.getItem(STORE_KEY), 10);
      return isNaN(v) ? 0 : v;
    } catch (e) { return 0; }
  }

  function writeShared(ts) {
    try { localStorage.setItem(STORE_KEY, String(ts)); } catch (e) { /* ignore */ }
  }

  function markActive() {
    if (!tracking) return;
    var now = Date.now();
    lastActivity = now;
    writeShared(now);
    hideWarning();
    if (idleMs > 0 && now - lastPing >= PING_EVERY_MS) {
      lastPing = now;
      origFetch('/api/auth/activity', {
        method: 'POST', credentials: 'same-origin', keepalive: true
      }).then(function (res) {
        if (res.status === 401) expire();
      }).catch(noop);                            // offline: next ping retries
    }
  }

  ['pointerdown', 'keydown', 'touchstart', 'wheel', 'scroll', 'mousemove']
    .forEach(function (evt) {
      window.addEventListener(evt, markActive, { passive: true, capture: true });
    });

  /* ── Idle check ──────────────────────────────────────────── */
  function check() {
    if (!tracking || idleMs <= 0) return;
    var last = Math.max(lastActivity, readShared());   // other tabs count too
    var remainingMs = idleMs - (Date.now() - last);
    if (remainingMs <= 0) {
      expire();
    } else if (remainingMs <= WARN_SECONDS * 1000) {
      showWarning(Math.ceil(remainingMs / 1000));
    } else {
      hideWarning();
    }
  }

  /* ── Expire: log out, reset to guest, reopen the login popup ─ */
  function expire() {
    if (!tracking) return;
    stopTracking();
    try {
      sessionStorage.setItem(FLAG_KEY, '1');
      SESSION_KEYS.forEach(function (k) { sessionStorage.removeItem(k); });
    } catch (e) { /* ignore */ }
    origFetch('/api/auth/logout', {
      method: 'POST', credentials: 'same-origin', keepalive: true
    }).catch(noop).then(function () { location.reload(); });
  }

  /* After the reload: open the page's own login popup. */
  function showLoginPopupIfFlagged() {
    var flagged = false;
    try {
      flagged = sessionStorage.getItem(FLAG_KEY) === '1';
      sessionStorage.removeItem(FLAG_KEY);
    } catch (e) { /* ignore */ }
    if (!flagged) return;

    function open() {
      setTimeout(function () {
        var btn = document.getElementById('headerLoginBtn');  // wired to the page's login modal
        if (btn) btn.click();
        showNotice(NOTICE_TEXT, 9000);
      }, 250);
    }
    if (document.readyState === 'complete') open();
    else window.addEventListener('load', open);
  }

  /* ── Small UI pieces ─────────────────────────────────────── */
  var BASE_STYLE =
    'position:fixed;left:50%;transform:translateX(-50%);z-index:2147483647;' +
    'max-width:calc(100vw - 32px);padding:12px 18px;border-radius:12px;' +
    'background:#1f2937;color:#fff;font:600 14px/1.4 system-ui,sans-serif;' +
    'box-shadow:0 8px 24px rgba(0,0,0,.3);text-align:center;';

  function showWarning(secondsLeft) {
    if (!warnEl) {
      warnEl = document.createElement('div');
      warnEl.setAttribute('role', 'alert');
      warnEl.style.cssText = BASE_STYLE + 'bottom:20px;';
      document.body.appendChild(warnEl);
    }
    warnEl.textContent = 'Still there? You will be signed out in ' + secondsLeft +
      's for inactivity. Tap anywhere to stay signed in.';
  }

  function hideWarning() {
    if (warnEl) { warnEl.remove(); warnEl = null; }
  }

  function showNotice(text, ms) {
    var el = document.createElement('div');
    el.setAttribute('role', 'status');
    el.style.cssText = BASE_STYLE + 'top:16px;';
    el.textContent = text;
    document.body.appendChild(el);
    setTimeout(function () { el.remove(); }, ms);
  }

  /* ── Boot ────────────────────────────────────────────────── */
  showLoginPopupIfFlagged();
  startTracking();   // no-op for guests (server answers 401 and we stay idle)
})();
