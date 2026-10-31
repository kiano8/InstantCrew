(function () {
  'use strict';

  var POLL_MS = 30000;
  var range = '24h';
  var last = null;
  var lastOkAt = 0;
  var pollSeq = 0;
  var $ = function (id) { return document.getElementById(id); };
  var num = function (n) { return Number(n || 0).toLocaleString('en-US'); };
  var TZ = 'Asia/Manila';

  function esc(v) {
    return String(v == null ? '' : v).replace(/[&<>"']/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
    });
  }

  function fmtTime(t, step) {
    var d = new Date(t * 1000);
    if (step >= 86400) return d.toLocaleDateString('en-US', { timeZone: TZ, month: 'short', day: 'numeric' });
    var time = d.toLocaleTimeString('en-US', { timeZone: TZ, hour: 'numeric', minute: '2-digit' });
    if (step >= 3600) return d.toLocaleDateString('en-US', { timeZone: TZ, weekday: 'short' }) + ' ' + time;
    return time;
  }
  function fmtFull(t) {
    return new Date(t * 1000).toLocaleString('en-US', { timeZone: TZ, month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' });
  }
  function fmtDur(s) {
    if (s < 3600) return Math.max(1, Math.round(s / 60)) + ' min';
    if (s < 86400) return (s / 3600).toFixed(1) + ' hr';
    return (s / 86400).toFixed(1) + ' days';
  }
  function niceMax(v) {
    if (v <= 4) return 4;
    var p = Math.pow(10, Math.floor(Math.log10(v)));
    // Divisible by 4 so all four gridlines get whole-number labels.
    var n = [2, 4, 8, 12].find(function (m) { return m * p >= v; });
    return n * p;
  }

  // opts: points [{t, ...}], series [{key, name, color}], step, bars, unit
  function drawChart(el, opts) {
    var showLegend = opts.series.length > 1;
    el.classList.toggle('has-legend', showLegend);
    var W = el.clientWidth || 300, H = 200;
    var L = 44, R = 8, T = 8, B = 22;
    var pts = opts.points, n = pts.length;
    var peak = 0;
    pts.forEach(function (p) { opts.series.forEach(function (s) { peak = Math.max(peak, p[s.key] || 0); }); });
    var top = niceMax(peak);
    var slot = (W - L - R) / Math.max(n, 1);
    var x = function (i) { return L + slot * (i + 0.5); };
    var y = function (v) { return T + (H - T - B) * (1 - v / top); };

    var svg = '';
    for (var g = 0; g <= 4; g++) {
      var val = top * g / 4;
      svg += '<line x1="' + L + '" x2="' + (W - R) + '" y1="' + y(val) + '" y2="' + y(val) + '" stroke="var(--tel-grid)"/>';
      svg += '<text x="' + (L - 6) + '" y="' + (y(val) + 4) + '" text-anchor="end">' + num(val) + '</text>';
    }
    var labels = Math.max(2, Math.floor((W - L - R) / 95));
    var every = Math.max(1, Math.ceil(n / labels));
    for (var i = 0; i < n; i += every) {
      svg += '<text x="' + x(i) + '" y="' + (H - 6) + '" text-anchor="middle">' + esc(fmtTime(pts[i].t, opts.step)) + '</text>';
    }
    opts.series.forEach(function (s) {
      if (opts.bars) {
        var bw = Math.max(1, Math.min(slot - 2, 22));
        pts.forEach(function (p, i) {
          var v = p[s.key] || 0;
          if (v > 0) svg += '<rect x="' + (x(i) - bw / 2) + '" y="' + y(v) + '" width="' + bw + '" height="' + (y(0) - y(v)) + '" rx="' + Math.min(4, bw / 2) + '" fill="' + s.color + '"/>';
        });
      } else {
        var d = pts.map(function (p, i) { return (i ? 'L' : 'M') + x(i).toFixed(1) + ' ' + y(p[s.key] || 0).toFixed(1); }).join('');
        svg += '<path d="' + d + '" fill="none" stroke="' + s.color + '" stroke-width="2" stroke-linejoin="round"/>';
      }
    });
    svg += '<line class="cross" y1="' + T + '" y2="' + y(0) + '" stroke="var(--adm-text-muted)" visibility="hidden"/>';

    var summary = opts.series.map(function (s) {
      return s.name + ' peak ' + num(Math.max.apply(null, pts.map(function (p) { return p[s.key] || 0; })));
    }).join(', ');
    var legend = showLegend ? '<div class="tel-legend">' + opts.series.map(function (s) {
      return '<span><i style="background:' + s.color + '"></i>' + esc(s.name) + '</span>';
    }).join('') + '</div>' : '';
    el.innerHTML = legend + '<svg role="img" aria-label="' + esc(summary) + '" viewBox="0 0 ' + W + ' ' + H + '" preserveAspectRatio="none">' + svg + '</svg><div class="tel-tip" hidden></div>';

    var svgEl = el.querySelector('svg'), tip = el.querySelector('.tel-tip'), cross = el.querySelector('.cross');
    svgEl.addEventListener('pointermove', function (e) {
      var box = svgEl.getBoundingClientRect();
      var i = Math.min(n - 1, Math.max(0, Math.floor((e.clientX - box.left - L) / slot)));
      var p = pts[i];
      if (!p) return;
      cross.setAttribute('x1', x(i)); cross.setAttribute('x2', x(i)); cross.setAttribute('visibility', 'visible');
      tip.innerHTML = '<b>' + esc(fmtTime(p.t, opts.step)) + '</b>' + opts.series.map(function (s) {
        return esc(s.name) + ': ' + num(p[s.key]) + (opts.unit || '');
      }).join('<br>');
      tip.hidden = false;
      tip.style.left = Math.min(x(i) + 10, Math.max(0, W - tip.offsetWidth - 2)) + 'px';
    });
    svgEl.addEventListener('pointerleave', function () { tip.hidden = true; cross.setAttribute('visibility', 'hidden'); });
  }

  function drawStrip(data) {
    var r = data.report, step = data.step;
    var html = r.series.map(function (p) {
      var end = p.t + step;
      var outage = r.outages.find(function (o) { return o.start < end && o.end > p.t; });
      var state = outage ? 'down' : (end <= r.monitoring_since || !r.monitoring_since ? 'none' : 'up');
      var label = { up: 'Up', down: 'Down', none: 'Not monitored' }[state];
      return '<span class="' + state + '" title="' + esc(fmtTime(p.t, step) + ': ' + label) + '"></span>';
    }).join('');
    $('strip').innerHTML = html;
    $('outages').innerHTML = r.outages.length
      ? '<ul>' + r.outages.map(function (o) {
          return '<li><strong>Down for ' + esc(fmtDur(o.seconds)) + '</strong>, ' + esc(fmtFull(o.start)) + ' to ' + esc(fmtFull(o.end)) + '</li>';
        }).join('') + '</ul>'
      : 'No outages in this period.';
  }

  function render(data) {
    var r = data.report, s = r.series, step = data.step;
    var sum = function (k) { return s.reduce(function (a, p) { return a + (p[k] || 0); }, 0); };
    var total = sum('requests'), errs = sum('server_errors');
    var weighted = s.reduce(function (a, p) { return a + p.avg_ms * p.requests; }, 0);

    $('kStatus').textContent = 'Up'; $('kStatus').className = 'adm-metric-val tel-up';
    $('kStatusSub').textContent = 'Running since ' + fmtFull(data.started_at);
    $('kUptime').textContent = r.uptime_percent.toFixed(2) + '%';
    $('kUptimeSub').textContent = r.outages.length + (r.outages.length === 1 ? ' outage' : ' outages');
    $('kRequests').textContent = num(total);
    $('kRequestsSub').textContent = num(sum('client_errors')) + ' with 4xx responses';
    $('kErrors').textContent = num(errs);
    $('kErrorsSub').textContent = (total ? (100 * errs / total).toFixed(2) : '0.00') + '% of requests';
    $('kLatency').textContent = num(total ? Math.round(weighted / total) : 0) + ' ms';
    $('kLatencySub').textContent = num(sum('slow')) + ' took over 1s';
    var today = r.visitors.length ? r.visitors[r.visitors.length - 1] : null;
    var dayStart = Math.floor((data.now + 28800) / 86400) * 86400 - 28800;
    $('kVisitors').textContent = num(today && today.t === dayStart ? today.visitors : 0);
    $('kVisitorsSub').textContent = 'Unique browsers';

    drawStrip(data);
    drawChart($('cTraffic'), { points: s, step: step, series: [{ key: 'requests', name: 'Requests', color: 'var(--tel-1)' }] });
    drawChart($('cErrors'), { points: s, step: step, series: [
      { key: 'client_errors', name: '4xx client', color: 'var(--tel-1)' },
      { key: 'server_errors', name: '5xx server', color: 'var(--tel-2)' }
    ] });
    drawChart($('cLatency'), { points: s, step: step, unit: ' ms', series: [{ key: 'avg_ms', name: 'Average', color: 'var(--tel-1)' }] });
    drawChart($('cActivity'), { points: s, step: step, series: [
      { key: 'signups', name: 'Signups', color: 'var(--tel-1)' },
      { key: 'jobs', name: 'Jobs posted', color: 'var(--tel-2)' },
      { key: 'payments', name: 'Payments', color: 'var(--tel-3)' }
    ] });
    var byDay = {};
    r.visitors.forEach(function (d) { byDay[d.t] = d.visitors; });
    var days = [];
    for (var i = 29; i >= 0; i--) days.push({ t: dayStart - i * 86400, visitors: byDay[dayStart - i * 86400] || 0 });
    drawChart($('cVisitors'), { points: days, step: 86400, bars: true, series: [{ key: 'visitors', name: 'Visitors', color: 'var(--tel-1)' }] });

    var facts = [
      ['Signed-in employers', num(r.signed_in.employer)],
      ['Signed-in crew', num(r.signed_in.applicant)],
      ['Database ping', num(r.db_ping_ms) + ' ms'],
      ['Database connections', num(r.db_in_use_conns) + ' in use / ' + num(r.db_open_conns) + ' open'],
      ['App memory', num(data.memory_mb) + ' MB'],
      ['Goroutines', num(data.goroutines)]
    ];
    $('facts').innerHTML = facts.map(function (f) { return '<dt>' + esc(f[0]) + '</dt><dd>' + esc(f[1]) + '</dd>'; }).join('');

    $('routes').innerHTML = r.routes.length ? r.routes.map(function (x) {
      return '<tr><td><code>' + esc(x.route) + '</code></td><td>' + num(x.requests) + '</td><td>' + num(x.client_errors) + '</td><td>' + num(x.server_errors) + '</td><td>' + num(x.avg_ms) + ' ms</td><td>' + num(x.slow) + '</td></tr>';
    }).join('') : '<tr><td colspan="6">No requests in this period.</td></tr>';
  }

  // This page is served by the app it monitors, so a failed poll is itself the signal.
  function showUnreachable(msg) {
    $('kStatus').textContent = 'Down'; $('kStatus').className = 'adm-metric-val tel-down';
    $('kStatusSub').textContent = msg;
    $('banner').hidden = false;
    $('banner').textContent = msg + (lastOkAt ? ' Last good reading: ' + fmtFull(lastOkAt) + '. Charts below are from then.' : '');
  }

  async function poll() {
    var seq = ++pollSeq; // a slow reply must not overwrite a newer range choice
    try {
      var res = await fetch('/api/admin/telemetry?range=' + range, { credentials: 'same-origin', headers: { 'X-Auth-Role': 'admin' } });
      if (seq !== pollSeq) return;
      if (res.status === 401 || res.status === 403) { location.replace('login.html?role=admin'); return; }
      if (!res.ok) { showUnreachable('The app is running but cannot reach its database.'); return; }
      var data = await res.json();
      if (seq !== pollSeq) return;
      last = data; lastOkAt = data.now;
      $('banner').hidden = true;
      render(data);
      $('updated').textContent = 'Updated ' + fmtFull(data.now) + ' (Philippine time). Refreshes every 30 seconds.';
    } catch (e) {
      if (seq === pollSeq) showUnreachable('Cannot reach the server.');
    }
  }

  document.querySelectorAll('[data-range]').forEach(function (b) {
    b.setAttribute('aria-pressed', String(b.dataset.range === range));
    b.addEventListener('click', function () {
      range = b.dataset.range;
      document.querySelectorAll('[data-range]').forEach(function (o) {
        o.classList.toggle('active', o === b);
        o.setAttribute('aria-pressed', String(o === b));
      });
      poll();
    });
  });

  var resizeTimer;
  window.addEventListener('resize', function () {
    clearTimeout(resizeTimer);
    resizeTimer = setTimeout(function () { if (last) render(last); }, 150);
  });

  // Real input keeps the admin session alive, same as session.js on other portals.
  var lastPing = 0;
  ['pointerdown', 'keydown', 'scroll'].forEach(function (ev) {
    window.addEventListener(ev, function () {
      if (Date.now() - lastPing < 30000) return;
      lastPing = Date.now();
      fetch('/api/auth/activity?role=admin', { method: 'POST', credentials: 'same-origin' }).catch(function () {});
    }, { passive: true });
  });

  poll();
  setInterval(function () { if (document.visibilityState === 'visible') poll(); }, POLL_MS);
  document.addEventListener('visibilitychange', function () { if (document.visibilityState === 'visible') poll(); });
})();
