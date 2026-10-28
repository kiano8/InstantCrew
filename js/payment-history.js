/* Payment history / activity log, shared by employer.html and applicant.html.
   Employer: search fees paid.  Applicant: free hires, top-ups, hires used. */
(function () {
    'use strict';

    var isEmployer = /employer/i.test(location.pathname);
    var ENDPOINT = isEmployer ? '/api/employer/payments' : '/api/applicant/payments';
    var TITLE = isEmployer ? 'Payment history' : 'Payments & hires log';
    var SUBTITLE = isEmployer ? 'Search fees you have paid' : 'Your free hires, top-ups and hires used';
    var RECEIPT_SVG = '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 2v20l2-1 2 1 2-1 2 1 2-1 2 1 2-1 2 1V2l-2 1-2-1-2 1-2-1-2 1-2-1-2 1Z"/><path d="M16 8h-6a2 2 0 1 0 0 4h4a2 2 0 1 1 0 4H8"/><path d="M12 17.5v-11"/></svg>';

    var CSS = '\
.ph-btn{display:inline-flex;align-items:center;justify-content:center;gap:7px;height:38px;padding:0 16px 0 13px;border:1.5px solid #E6B800;border-radius:9999px;background:#FFCC00;color:#1C1C1E;font-family:inherit;font-size:.8rem;font-weight:700;letter-spacing:-.005em;cursor:pointer;white-space:nowrap;line-height:1;text-decoration:none;-webkit-tap-highlight-color:transparent;box-shadow:0 1px 2px rgba(28,28,30,.08),0 2px 8px rgba(255,204,0,.28);transition:background .14s ease,transform .14s ease,box-shadow .14s ease}\
.ph-btn:hover{background:#FFD633;transform:translateY(-1px);box-shadow:0 2px 4px rgba(28,28,30,.1),0 6px 14px rgba(255,204,0,.38)}\
.ph-btn:active{background:#E6B800;transform:translateY(0);box-shadow:0 1px 2px rgba(28,28,30,.1)}\
.ph-btn:focus-visible{outline:2px solid #1C1C1E;outline-offset:2px}\
.ph-btn-icon{display:inline-flex;flex-shrink:0}\
@media (max-width:520px){.ph-btn{height:34px;padding:0 12px 0 10px;font-size:.74rem;gap:5px}}\
@media (max-width:420px){.ph-btn{width:38px;height:38px;padding:0;gap:0}.ph-btn-label{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0);white-space:nowrap}}\
.ph-overlay{position:fixed;inset:0;background:rgba(28,28,30,.45);-webkit-backdrop-filter:blur(4px);backdrop-filter:blur(4px);display:none;align-items:center;justify-content:center;z-index:2000;padding:16px}\
.ph-overlay.open{display:flex}\
.ph-card{background:#FFFFFF;color:#1C1C1E;width:min(560px,100%);max-height:86vh;display:flex;flex-direction:column;border:1px solid #E5E5EA;border-radius:20px;box-shadow:0 25px 50px -12px rgba(0,0,0,.25);overflow:hidden;font-family:"Plus Jakarta Sans",-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;animation:ph-pop .24s cubic-bezier(.16,1,.3,1)}\
@keyframes ph-pop{from{opacity:0;transform:scale(.96) translateY(10px)}to{opacity:1;transform:none}}\
.ph-head{display:flex;align-items:center;justify-content:space-between;gap:12px;padding:18px 20px 14px}\
.ph-head-left{display:flex;align-items:center;gap:12px;min-width:0}\
.ph-head-icon{width:42px;height:42px;border-radius:12px;background:#FFCC00;color:#1C1C1E;display:flex;align-items:center;justify-content:center;flex-shrink:0}\
.ph-head h3{margin:0;font-size:1.05rem;font-weight:800;letter-spacing:-.02em;color:#1C1C1E}\
.ph-head p{margin:3px 0 0;font-size:.78rem;color:#8E8E93}\
.ph-close{width:32px;height:32px;flex-shrink:0;border-radius:50%;border:1px solid #E5E5EA;background:#F5F5F7;color:#8E8E93;cursor:pointer;display:flex;align-items:center;justify-content:center;padding:0;transition:background .14s ease,color .14s ease}\
.ph-close:hover{background:#EFEFF4;color:#1C1C1E}\
.ph-close:focus-visible,.ph-chip-btn:focus-visible{outline:2px solid #1C1C1E;outline-offset:2px}\
.ph-filters{display:flex;gap:4px;margin:0 20px 6px;padding:4px;background:#F5F5F7;border-radius:9999px}\
.ph-chip-btn{flex:1;border:0;background:transparent;border-radius:9999px;padding:8px 12px;font-family:inherit;font-size:.8rem;font-weight:600;color:#8E8E93;cursor:pointer;transition:background .14s ease,color .14s ease,box-shadow .14s ease}\
.ph-chip-btn.active{background:#FFFFFF;color:#1C1C1E;box-shadow:0 1px 4px rgba(0,0,0,.08)}\
.ph-body{overflow:auto;padding:10px 20px 20px;display:flex;flex-direction:column;gap:10px}\
.ph-row{display:flex;gap:12px;justify-content:space-between;padding:14px;background:#FFFFFF;border:1px solid #E5E5EA;border-radius:14px}\
.ph-main{min-width:0}\
.ph-title{font-weight:800;font-size:.9rem;letter-spacing:-.01em;color:#1C1C1E}\
.ph-desc{color:rgba(28,28,30,.6);font-size:.8rem;margin-top:3px;word-break:break-word}\
.ph-reason{color:#C0392B}\
.ph-meta{color:#8E8E93;font-size:.72rem;margin-top:6px}\
.ph-side{display:flex;flex-direction:column;align-items:flex-end;gap:6px;text-align:right;flex-shrink:0}\
.ph-amt{font-weight:800;font-size:1rem;letter-spacing:-.02em;color:#1C1C1E}\
.ph-status{display:inline-flex;align-items:center;gap:6px;font-size:.66rem;font-weight:800;letter-spacing:.07em;text-transform:uppercase;color:#8E8E93;white-space:nowrap}\
.ph-status::before{content:"";width:7px;height:7px;border-radius:50%;background:currentColor}\
.ph-s-verified,.ph-s-granted{color:#1F9D48}\
.ph-s-pending{color:#B38F00}\
.ph-s-pending::before{background:#FFCC00}\
.ph-s-rejected{color:#C0392B}\
.ph-s-used{color:#8E8E93}\
.ph-link{color:#1C1C1E;font-weight:700;font-size:.72rem;text-decoration:underline;text-decoration-color:#FFCC00;text-decoration-thickness:2px;text-underline-offset:3px;margin-left:8px}\
.ph-empty,.ph-err{padding:32px 16px;text-align:center;color:#8E8E93;font-size:.85rem;background:#F5F5F7;border-radius:14px}\
.ph-err{color:#C0392B}';

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
            ? [['all', 'All'], ['verified', 'Verified'], ['rejected', 'Rejected']]
            : [['all', 'All'], ['payments', 'Payments'], ['hires', 'Hires']];
    }

    function matches(e) {
        if (e.status === 'pending') return false;
        if (filter === 'all' || filter === 'pending') return true;
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
            var reason = e.status === 'rejected' && e.rejection_reason ? '<div class="ph-desc ph-reason">Reason: ' + esc(e.rejection_reason) + '</div>' : '';
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
        overlay.innerHTML = '<div class="ph-card"><div class="ph-head"><div class="ph-head-left">' +
            '<span class="ph-head-icon" aria-hidden="true">' + RECEIPT_SVG + '</span>' +
            '<div><h3>' + TITLE + '</h3><p>' + SUBTITLE + '</p></div></div>' +
            '<button type="button" class="ph-close" aria-label="Close"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M18 6 6 18M6 6l12 12"/></svg></button></div>' +
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

        var btn = document.getElementById('phHeaderBtn');
        var notifWrap = document.querySelector('.db-notif-wrap, #headerNotifWrap');

        if (!btn) {
            btn = document.createElement('button');
            btn.type = 'button';
            btn.className = 'ph-btn';
            btn.id = 'phHeaderBtn';
            btn.textContent = TITLE;
            if (notifWrap && notifWrap.parentNode) {
                notifWrap.parentNode.insertBefore(btn, notifWrap);
            } else {
                var headerRight = document.querySelector('.db-header-inner > div:last-child, .app-header-inner > div:last-child');
                if (headerRight) headerRight.insertBefore(btn, headerRight.firstChild);
                else document.body.appendChild(btn);
            }
        } else {
            if (!btn.textContent.trim()) {
                btn.textContent = TITLE;
            }
        }
        // Receipt icon + label; the label is visually hidden on narrow phones.
        if (!btn.querySelector('.ph-btn-label')) {
            var label = btn.textContent.trim() || TITLE;
            btn.innerHTML = '<span class="ph-btn-icon" aria-hidden="true">' + RECEIPT_SVG.replace('width="20" height="20"', 'width="15" height="15"') + '</span><span class="ph-btn-label"></span>';
            btn.querySelector('.ph-btn-label').textContent = label;
            btn.title = label;
        }

        // Synchronize initial visibility with notifWrap (hidden if user is logged out)
        if (notifWrap) {
            var isHidden = notifWrap.style.display === 'none' || window.getComputedStyle(notifWrap).display === 'none';
            if (isHidden) {
                btn.style.display = 'none';
            }
        }

        btn.addEventListener('click', open);
        window.openPaymentHistory = open;
    }

    if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', build);
    else build();
})();
