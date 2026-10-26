/* ─────────────────────────────────────────────────────────────
   legal-nav.js — "Back to Home" routing for privacy.html / terms.html

   Guests keep the plain index.html links. Logged-in users are sent back
   to Step 1 of their own portal (employer.html or applicant.html).
   These pages usually open in a new tab, so sessionStorage is empty;
   the server cookie (/api/auth/me) is the source of truth.
   ───────────────────────────────────────────────────────────── */
(function () {
  'use strict';

  var PORTALS = { employer: 'employer.html', admin: 'employer.html', applicant: 'applicant.html' };

  // Prefer the portal the user came from (they may hold both role cookies).
  function referrerRole() {
    var ref = document.referrer || '';
    if (ref.indexOf('employer.html') !== -1) return 'employer';
    if (ref.indexOf('applicant.html') !== -1) return 'applicant';
    return '';
  }

  function sessionRole() {
    try {
      var now = Date.now();
      var worker = JSON.parse(sessionStorage.getItem('ic_worker_session') || 'null');
      var employer = JSON.parse(sessionStorage.getItem('ic_employer_session') || 'null');
      if (worker && worker.expiresAt > now) return 'applicant';
      if (employer && employer.expiresAt > now) return 'employer';
    } catch (e) { }
    return '';
  }

  function pointHomeLinksTo(url) {
    document.querySelectorAll('a[href="index.html"]').forEach(function (link) {
      link.setAttribute('href', url);
      if (link.getAttribute('aria-label') === 'Instant Crew Home') link.setAttribute('aria-label', 'Instant Crew Dashboard');
    });
  }

  var hint = referrerRole() || sessionRole();
  var url = '/api/auth/me' + (hint ? '?role=' + hint : '');

  fetch(url, { credentials: 'same-origin' })
    .then(function (res) { return res.ok ? res.json() : null; })
    .then(function (data) {
      var role = data && data.user && data.user.role;
      if (PORTALS[role]) pointHomeLinksTo(PORTALS[role]);
    })
    .catch(function () { /* offline or not logged in: stay on index.html */ });
})();
