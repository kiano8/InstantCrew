/* Payment history / activity log, shared by employer.html and applicant.html.
   Employer: search fees paid.  Applicant: free hires, top-ups, hires used. */
(function () {
    'use strict';

    var isEmployer = /employer/i.test(location.pathname);
    var ENDPOINT = isEmployer ? '/api/employer/payments' : '/api/applicant/payments';
    var TITLE = isEmployer ? 'Payment history' : 'Payments & hires log';

    var CSS = '\
.ph-btn{display:inline-flex;align-items:center;gap:6px;padding:8px 14px;border:1px solid #E2E8F0;border-radius:999px;background:#fff;color:#1E293B;font:600 .8rem inherit;cursor:pointer;box-shadow:0 1px 2px rgba(0,0,0,.06)}\
.ph-btn:hover{background:#F8FAFC}\
.ph-btn.ph-float{position:fixed;right:16px;bottom:16px;z-index:900;box-shadow:0 4px 14px rgba(0,0,0,.18)}\
.ph-overlay{position:fixed;inset:0;background:rgba(15,23,42,.55);display:none;align-items:center;justify-content:center;z-index:2000;padding:16px}\
.ph-overlay.open{display:flex}\
.ph-card{background:#fff;color:#0F172A;width:min(680px,100%);max-height:86vh;display:flex;flex-direction:column;border-radius:16px;box-shadow:0 20px 50px rgba(0,0,0,.3);overflow:hidden;font-family:inherit}\
.ph-head{display:flex;align-items:center;justify-content:space-between;padding:16px 18px;border-bottom:1px solid #E2E8F0}\
.ph-head h3{margin:0;font-size:1.05rem}\
.ph-close{border:0;background:transparent;font-size:1.4rem;line-height:1;cursor:pointer;color:#475569}\
.ph-filters{display:flex;gap:6px;padding:10px 18px;border-bottom:1px solid #F1F5F9;flex-wrap:wrap}\
.ph-chip-btn{border:1px solid #E2E8F0;background:#fff;border-radius:999px;padding:4px 12px;font-size:.75rem;font-weight:600;cursor:pointer;color:#475569}\
.ph-chip-btn.active{background:#0F172A;color:#fff;border-color:#0F172A}\
.ph-body{overflow:auto;padding:6px 18px 16px}\
.ph-row{display:flex;gap:12px;justify-content:space-between;padding:12px 0;border-bottom:1px solid #F1F5F9}\
.ph-row:last-child{border-bottom:0}\
.ph-main{min-width:0}\
.ph-title{font-weight:700;font-size:.9rem}\
.ph-desc{color:#475569;font-size:.8rem;margin-top:2px;word-break:break-word}\
.ph-meta{color:#94A3B8;font-size:.72rem;margin-top:4px}\
.ph-side{text-align:right;flex-shrink:0}\
.ph-amt{font-weight:800;font-size:.95rem}\
.ph-status{display:inline-block;margin-top:4px;padding:2px 9px;border-radius:999px;font-size:.68rem;font-weight:700;text-transform:capitalize}\
.ph-s-verified,.ph-s-granted{background:#DCFCE7;color:#166534}\
.ph-s-pending{background:#DBEAFE;color:#1E40AF}\
.ph-s-rejected{background:#FEE2E2;color:#991B1B}\
.ph-s-used{background:#FEF3C7;color:#92400E}\
.ph-link{color:#2563EB;font-size:.72rem;text-decoration:underline;margin-left:8px}\
.ph-empty,.ph-err{padding:28px 0;text-align:center;color:#64748B;font-size:.85rem}\
.ph-err{color:#B91C1C}\
@media (prefers-color-scheme:dark){.ph-card{background:#0F172A;color:#E2E8F0}.ph-head,.ph-filters,.ph-row{border-color:#1E293B}.ph-btn,.ph-chip-btn{background:#1E293B;color:#E2E8F0;border-color:#334155}.ph-chip-btn.active{background:#E2E8F0;color:#0F172A}.ph-desc{color:#94A3B8}}';

    function esc(s) {
        return String(s == null ? '' : s).replace(/[&<>"']/g, function (c) {
            return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
        });
    }
    function when(ts) {
        try { return new Date(ts * 1000).toLocaleString([], { dateStyle: 'medium', timeStyle: 'short' }); }
        catch (e) { return ''; }
    }
    function safeProof(u) {
        return typeof u === 'string' && (u.indexOf('/uploads/') === 0 || u.indexOf('data:image/') === 0) ? u : '';
    }

    var overlay, bodyEl, filterEl, items = [], filter = 'all';

    function filtersFor() {
        return isEmployer
            ? [['all', 'All'], ['pending', 'Pending'], ['verified', 'Verified'], ['rejected', 'Rejected']]
            : [['all', 'All'], ['payments', 'Payments'], ['hires', 'Hires']];
    }

    function matches(e) {
        if (filter === 'all') return true;
        if (isEmployer) return e.status === filter;
        if (filter === 'payments') return e.kind === 'hiring_pass';
        if (filter === 'hires') return e.kind === 'hire_used' || e.kind === 'free_hires';
        return true;
    }

    function render() {
        filterEl.innerHTML = filtersFor().map(function (f) {
            return '<button type="button" class="ph-chip-btn' + (filter === f[0] ? ' active' : '') + '" data-f="' + f[0] + '">' + f[1] + '</button>';
        }).join('');
        var list = items.filter(matches);
        if (!list.length) {
            bodyEl.innerHTML = '<div class="ph-empty">' + (items.length ? 'Nothing matches this filter.' : 'No payment activity yet.') + '</div>';
            return;
        }
        bodyEl.innerHTML = list.map(function (e) {
            var proof = safeProof(e.proof_image);
            var amount = e.amount_centavos > 0 ? esc(e.amount_pesos) : (e.kind === 'free_hires' ? 'Free' : '');
            var method = e.method ? ' · ' + esc(e.method.toUpperCase()) : '';
            var ref = e.ref ? ' · ' + esc(e.ref) : '';
            var reason = e.status === 'rejected' && e.rejection_reason ? '<div class="ph-desc" style="color:#B91C1C">Reason: ' + esc(e.rejection_reason) + '</div>' : '';
            return '<div class="ph-row"><div class="ph-main">' +
                '<div class="ph-title">' + esc(e.title) + '</div>' +
                '<div class="ph-desc">' + esc(e.description) + '</div>' + reason +
                '<div class="ph-meta">' + esc(when(e.created_at)) + method + ref +
                (proof ? '<a class="ph-link" href="' + esc(proof) + '" target="_blank" rel="noopener">View receipt</a>' : '') +
                '</div></div>' +
                '<div class="ph-side"><div class="ph-amt">' + amount + '</div>' +
                '<span class="ph-status ph-s-' + esc(e.status) + '">' + esc(e.status) + '</span></div></div>';
        }).join('');
    }

    function load() {
        bodyEl.innerHTML = '<div class="ph-empty">Loading…</div>';
        fetch(ENDPOINT, { credentials: 'same-origin', cache: 'no-store' })
            .then(function (r) {
                return r.json().catch(function () { return {}; }).then(function (d) {
                    if (!r.ok) throw new Error(d.error || 'Could not load payment history.');
                    return d;
                });
            })
            .then(function (d) { items = d.payments || []; render(); })
            .catch(function (err) { bodyEl.innerHTML = '<div class="ph-err">' + esc(err.message) + '</div>'; });
    }

    function open() { overlay.classList.add('open'); load(); }
    function close() { overlay.classList.remove('open'); }

    function build() {
        var style = document.createElement('style');
        style.textContent = CSS;
        document.head.appendChild(style);

        overlay = document.createElement('div');
        overlay.className = 'ph-overlay';
        overlay.setAttribute('role', 'dialog');
        overlay.setAttribute('aria-modal', 'true');
        overlay.innerHTML = '<div class="ph-card"><div class="ph-head"><h3>' + TITLE +
            '</h3><button type="button" class="ph-close" aria-label="Close">&times;</button></div>' +
            '<div class="ph-filters"></div><div class="ph-body"></div></div>';
        document.body.appendChild(overlay);
        filterEl = overlay.querySelector('.ph-filters');
        bodyEl = overlay.querySelector('.ph-body');

        overlay.addEventListener('click', function (e) { if (e.target === overlay) close(); });
        overlay.querySelector('.ph-close').addEventListener('click', close);
        filterEl.addEventListener('click', function (e) {
            var b = e.target.closest('[data-f]');
            if (b) { filter = b.getAttribute('data-f'); render(); }
        });
        document.addEventListener('keydown', function (e) { if (e.key === 'Escape') close(); });

        var btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'ph-btn';
        btn.textContent = TITLE;
        btn.addEventListener('click', open);

        // Applicants: sit next to the hiring-pass widget. Everyone else: floating button.
        var host = document.getElementById('appHiringPassWidget');
        if (host) { btn.style.margin = '10px 0 0'; host.appendChild(btn); }
        else { btn.classList.add('ph-float'); document.body.appendChild(btn); }

        window.openPaymentHistory = open;
    }

    if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', build);
    else build();
})();
