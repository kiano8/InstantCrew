/* ═══════════════════════════════════════════════════════════════════
   Instant Crew — Automated PWA Install & Add-to-Home-Screen
   Unified Controller for Android Native Prompt + iOS Safari Share Sheet
   ═══════════════════════════════════════════════════════════════════ */

(function () {
    'use strict';

    // 1. Register Service Worker
    if ('serviceWorker' in navigator) {
        window.addEventListener('load', () => {
            navigator.serviceWorker.register('./sw.js').catch((err) => {
                console.log('[SW] Note:', err);
            });
        });
    }

    let deferredPrompt = window.deferredPrompt || null;
    let toastTimeout = null;
    let pendingPromptWaitTimeout = null;
    let isIosGuideActive = false;

    // 2. Intercept beforeinstallprompt for instant 1-tap Android & Chrome prompt
    window.addEventListener('beforeinstallprompt', (e) => {
        e.preventDefault();
        deferredPrompt = e;
        window.deferredPrompt = e;
    });

    // 3. App installed event
    window.addEventListener('appinstalled', () => {
        deferredPrompt = null;
        window.deferredPrompt = null;
        updateInstalledButtons();
        showInstallToast('Instant Crew was successfully added to your Home Screen!');
    });

    // 4. Platform Detection Helpers
    function getUA() {
        return (navigator.userAgent || navigator.vendor || window.opera || '').toLowerCase();
    }

    function isIOS() {
        const ua = getUA();
        return /iphone|ipad|ipod/.test(ua) || (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1);
    }

    function isChromeIOS() {
        return isIOS() && /crios/i.test(getUA());
    }

    function isAndroid() {
        return /android/.test(getUA());
    }

    function isInAppBrowser() {
        const ua = getUA();
        return /fban|fbav|instagram|messenger|line|micromessenger|tiktok|snapchat|gsa\/|twitter|threads|linkedin/i.test(ua);
    }

    function isStandaloneMode() {
        return window.matchMedia('(display-mode: standalone)').matches || window.navigator.standalone === true;
    }

    // 5. Ensure Toast & Modal DOM elements exist
    function ensureInstallDOM() {
        if (!document.getElementById('installToast')) {
            const toast = document.createElement('div');
            toast.className = 'install-toast';
            toast.id = 'installToast';
            toast.setAttribute('role', 'status');
            toast.setAttribute('aria-live', 'polite');
            toast.innerHTML = `
                <svg class="install-toast-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
                    <polyline points="22 4 12 14.01 9 11.01"></polyline>
                </svg>
                <span id="installToastMessage">Instant Crew added to Home Screen!</span>
            `;
            document.body.appendChild(toast);
        }

        if (!document.getElementById('iosShareGuideBanner')) {
            const banner = document.createElement('div');
            banner.className = 'ios-share-guide-banner';
            banner.id = 'iosShareGuideBanner';
            banner.innerHTML = `
                <div class="ios-share-guide-icon">
                    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                        <line x1="12" y1="5" x2="12" y2="19"></line>
                        <polyline points="19 12 12 19 5 12"></polyline>
                    </svg>
                </div>
                <div class="ios-share-guide-text">
                    <div class="ios-share-guide-title">Look for the Share Button Below 👇</div>
                    <div class="ios-share-guide-sub">Tap <strong style="color:#FFCC00;">[↑] Share</strong> in Safari's toolbar, then scroll down to <strong>"Add to Home Screen"</strong></div>
                </div>
                <button type="button" class="ios-share-guide-close" id="iosShareGuideCloseBtn" aria-label="Dismiss">
                    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <line x1="18" y1="6" x2="6" y2="18"></line>
                        <line x1="6" y1="6" x2="18" y2="18"></line>
                    </svg>
                </button>
            `;
            document.body.appendChild(banner);

            const closeBtn = document.getElementById('iosShareGuideCloseBtn');
            if (closeBtn) {
                closeBtn.addEventListener('click', hideIosGuideBanner);
            }
        }

        if (!document.getElementById('installModalOverlay')) {
            const modal = document.createElement('div');
            modal.className = 'install-modal-overlay';
            modal.id = 'installModalOverlay';
            modal.setAttribute('aria-hidden', 'true');
            modal.setAttribute('role', 'dialog');
            modal.setAttribute('aria-modal', 'true');
            modal.setAttribute('aria-labelledby', 'installModalTitle');
            modal.innerHTML = `
                <div class="install-modal-backdrop" id="installModalBackdrop"></div>
                <div class="install-modal-dialog">
                    <button type="button" class="install-modal-close" id="installModalCloseBtn" aria-label="Close dialog">
                        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
                            <line x1="18" y1="6" x2="6" y2="18"></line>
                            <line x1="6" y1="6" x2="18" y2="18"></line>
                        </svg>
                    </button>
                    <img src="images/logo.svg" alt="Instant Crew" class="install-modal-app-icon">
                    <h2 class="install-modal-title" id="installModalTitle">Install Instant Crew</h2>
                    <p class="install-modal-subtitle" id="installModalSubtitle">Add to your Home Screen for fast, 1-tap access anytime.</p>
                    <div class="install-steps-list" id="installStepsList"></div>
                    <button type="button" class="btn-install-modal-action" id="btnInstallModalClose">Got it</button>
                </div>
            `;
            document.body.appendChild(modal);
        }

        // Bind close handlers once, whether the modal was created here or is
        // already present in the page markup (e.g. index.html).
        const modal = document.getElementById('installModalOverlay');
        if (modal && !modal.dataset.closeBound) {
            modal.dataset.closeBound = 'true';
            const closeBtn = document.getElementById('installModalCloseBtn');
            const backdrop = document.getElementById('installModalBackdrop');
            const gotItBtn = document.getElementById('btnInstallModalClose');

            if (closeBtn) closeBtn.addEventListener('click', closeInstallModal);
            if (backdrop) backdrop.addEventListener('click', closeInstallModal);
            if (gotItBtn) gotItBtn.addEventListener('click', closeInstallModal);
        }
    }

    function showInstallToast(message) {
        ensureInstallDOM();
        const toast = document.getElementById('installToast');
        const msgSpan = document.getElementById('installToastMessage');
        if (!toast) return;
        if (msgSpan) msgSpan.textContent = message;
        toast.classList.add('show');
        clearTimeout(toastTimeout);
        toastTimeout = setTimeout(() => {
            toast.classList.remove('show');
        }, 4000);
    }

    function showIosGuideBanner() {
        ensureInstallDOM();
        const banner = document.getElementById('iosShareGuideBanner');
        if (banner) {
            banner.classList.add('show');
            setTimeout(() => {
                hideIosGuideBanner();
            }, 8000);
        }
    }

    function hideIosGuideBanner() {
        const banner = document.getElementById('iosShareGuideBanner');
        if (banner) banner.classList.remove('show');
    }

    function openInstallModal(customSubtitle, customStepsHtml, customTitle) {
        ensureInstallDOM();
        const modal = document.getElementById('installModalOverlay');
        const title = document.getElementById('installModalTitle');
        const sub = document.getElementById('installModalSubtitle');
        const list = document.getElementById('installStepsList');

        if (customTitle && title) title.textContent = customTitle;
        if (customSubtitle && sub) sub.textContent = customSubtitle;
        if (customStepsHtml && list) list.innerHTML = customStepsHtml;

        if (modal) {
            modal.classList.add('open');
            modal.setAttribute('aria-hidden', 'false');
            document.body.style.overflow = 'hidden';
        }
    }

    function closeInstallModal() {
        const modal = document.getElementById('installModalOverlay');
        if (modal) {
            modal.classList.remove('open');
            modal.setAttribute('aria-hidden', 'true');
            document.body.style.overflow = '';
        }
        if (isIosGuideActive && isIOS() && !isStandaloneMode()) {
            isIosGuideActive = false;
            showIosGuideBanner();
        }
    }

    function updateInstalledButtons() {
        document.querySelectorAll('.nav-install-btn').forEach(btn => {
            btn.classList.add('installed');
            const txt = btn.querySelector('.install-btn-text');
            if (txt) txt.textContent = 'Installed';
            btn.setAttribute('title', 'Instant Crew is installed on your Home Screen');
            btn.setAttribute('aria-label', 'Instant Crew is installed');
        });
    }

    // 6. Master Install Trigger Function
    window.triggerAppInstall = async function (e) {
        if (e && e.preventDefault) e.preventDefault();
        ensureInstallDOM();

        // Check if already in standalone PWA mode
        if (isStandaloneMode()) {
            showInstallToast('Instant Crew is already installed on your Home Screen!');
            return;
        }

        // ── Case A: Android ──
        if (isAndroid()) {
            // Check for in-app browser (e.g. Facebook, Instagram, Messenger)
            if (isInAppBrowser()) {
                const chromeIntentUrl = `intent://${location.host}${location.pathname}${location.search}#Intent;scheme=https;package=com.android.chrome;end`;
                const inAppSteps = `
                    <div class="install-step-item">
                        <span class="install-step-num">1</span>
                        <div>Tap the <strong>three dots (⋮)</strong> in the top-right corner</div>
                    </div>
                    <div class="install-step-item">
                        <span class="install-step-num">2</span>
                        <div>Select <strong>'Open in Chrome'</strong></div>
                    </div>
                    <div class="install-step-item">
                        <span class="install-step-num">3</span>
                        <div>Tap <strong>'Install'</strong> to automatically add Instant Crew to your Home Screen</div>
                    </div>
                    <div style="margin-top:14px;">
                        <a href="${chromeIntentUrl}" class="btn-install-modal-action" style="display:block; text-align:center; text-decoration:none;">Open in Google Chrome</a>
                    </div>
                `;
                openInstallModal('Open in Google Chrome to install directly to your Home Screen:', inAppSteps);
                return;
            }

            // If prompt is ready, immediately trigger the native Android prompt dialog
            if (deferredPrompt) {
                try {
                    deferredPrompt.prompt();
                    const choice = await deferredPrompt.userChoice;
                    if (choice && choice.outcome === 'accepted') {
                        showInstallToast('Adding Instant Crew to your Home Screen...');
                    }
                    deferredPrompt = null;
                    window.deferredPrompt = null;
                } catch (err) {
                    console.error('Android prompt error:', err);
                }
                return;
            }

            // If deferredPrompt hasn't fired yet, wait briefly (up to 1000ms)
            showInstallToast('Opening installation prompt…');
            let promptFired = false;
            const promptCheckInterval = setInterval(async () => {
                if (deferredPrompt && !promptFired) {
                    promptFired = true;
                    clearInterval(promptCheckInterval);
                    clearTimeout(pendingPromptWaitTimeout);
                    try {
                        deferredPrompt.prompt();
                        const choice = await deferredPrompt.userChoice;
                        if (choice && choice.outcome === 'accepted') {
                            showInstallToast('Adding Instant Crew to your Home Screen...');
                        }
                        deferredPrompt = null;
                        window.deferredPrompt = null;
                    } catch (err) {
                        console.error('Android prompt error:', err);
                    }
                }
            }, 100);

            pendingPromptWaitTimeout = setTimeout(() => {
                clearInterval(promptCheckInterval);
                if (!promptFired && !deferredPrompt) {
                    const androidSteps = `
                        <div class="install-step-item">
                            <span class="install-step-num">1</span>
                            <div>Tap the <strong>menu icon (⋮)</strong> at the top right of Chrome</div>
                        </div>
                        <div class="install-step-item">
                            <span class="install-step-num">2</span>
                            <div>Tap <strong>'Install app'</strong> or <strong>'Add to Home screen'</strong></div>
                        </div>
                        <div class="install-step-item">
                            <span class="install-step-num">3</span>
                            <div>Tap <strong>'Install'</strong> to add Instant Crew to your device</div>
                        </div>
                    `;
                    openInstallModal('Add Instant Crew directly to your Android Home Screen:', androidSteps);
                }
            }, 1000);
            return;
        }

        // ── Case B: iPhone / iOS ──
        if (isIOS()) {
            if (isInAppBrowser()) {
                const iosInAppSteps = `
                    <div class="install-step-item">
                        <span class="install-step-num">1</span>
                        <div class="install-step-body">
                            <div class="install-step-title">Tap the menu in the corner</div>
                            <div class="install-step-desc">Tap the <strong>•••</strong> (three dots) or <strong>Share</strong> button at the top-right or bottom-right of this in-app browser.</div>
                        </div>
                    </div>
                    <div class="install-step-item">
                        <span class="install-step-num">2</span>
                        <div class="install-step-body">
                            <div class="install-step-title">Select <strong>'Open in Safari'</strong></div>
                            <div class="install-step-desc">In-app browsers (Instagram, Messenger, etc.) do not allow adding apps to the Home Screen. Safari is required.</div>
                        </div>
                    </div>
                    <div class="install-step-item">
                        <span class="install-step-num">3</span>
                        <div class="install-step-body">
                            <div class="install-step-title">Tap <strong>'Install'</strong> once Safari opens</div>
                            <div class="install-step-desc">You can then add Instant Crew straight to your Home Screen in 1 tap.</div>
                        </div>
                    </div>
                `;
                openInstallModal('Open in Safari first to add Instant Crew to your iPhone Home Screen:', iosInAppSteps, 'Open in Safari to Install');
                return;
            }

            if (isChromeIOS()) {
                const chromeIosSteps = `
                    <div class="install-step-item">
                        <span class="install-step-num">1</span>
                        <div class="install-step-body">
                            <div class="install-step-title">Tap the <strong>Share</strong> icon in Chrome</div>
                            <div class="install-step-desc">
                                Tap the
                                <span class="install-step-inline-icon">
                                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3">
                                        <path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/>
                                        <polyline points="16 6 12 2 8 6"/>
                                        <line x1="12" y1="2" x2="12" y2="15"/>
                                    </svg>
                                </span>
                                Share button at the top-right of the address bar, or tap the three dots (<strong>⋯</strong>) at the bottom.
                            </div>
                        </div>
                    </div>
                    <div class="install-step-item">
                        <span class="install-step-num">2</span>
                        <div class="install-step-body">
                            <div class="install-step-title">Scroll down and tap <strong>'Add to Home Screen'</strong></div>
                            <div class="install-step-desc">
                                In the options list, scroll down and select
                                <span class="install-step-inline-icon">
                                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3">
                                        <rect x="3" y="3" width="18" height="18" rx="3"/>
                                        <line x1="12" y1="8" x2="12" y2="16"/>
                                        <line x1="8" y1="12" x2="16" y2="12"/>
                                    </svg>
                                </span>
                                <strong>Add to Home Screen</strong>.
                            </div>
                        </div>
                    </div>
                    <div class="install-step-item">
                        <span class="install-step-num">3</span>
                        <div class="install-step-body">
                            <div class="install-step-title">Tap <strong>'Add'</strong> in the top-right corner</div>
                            <div class="install-step-desc">Instant Crew will appear on your Home Screen as an app.</div>
                        </div>
                    </div>
                `;
                openInstallModal('Add Instant Crew to your iPhone Home Screen using Google Chrome:', chromeIosSteps, 'Add to Home Screen');
                return;
            }

            // Safari on iPhone / iPad (Apple does not support programmatic install or navigator.share for Home Screen)
            const isIPad = /ipad/.test(getUA()) || (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1);
            const toolbarLocation = isIPad ? 'at the top-right of Safari' : 'in Safari\'s toolbar at the bottom of your screen';
            const toolbarHint = isIPad ? '' : '<div class="install-step-hint">💡 <em>Toolbar hidden? Tap near the bottom of your screen to show it.</em></div>';

            const safariSteps = `
                <div class="install-step-item">
                    <span class="install-step-num">1</span>
                    <div class="install-step-body">
                        <div class="install-step-title">Tap the <strong>Share</strong> button in Safari</div>
                        <div class="install-step-desc">
                            Look for the
                            <span class="install-step-inline-icon" title="Share button">
                                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3">
                                    <path d="M4 12v8a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-8"/>
                                    <polyline points="16 6 12 2 8 6"/>
                                    <line x1="12" y1="2" x2="12" y2="15"/>
                                </svg>
                            </span>
                            Share icon ${toolbarLocation}.
                            ${toolbarHint}
                        </div>
                    </div>
                </div>
                <div class="install-step-item">
                    <span class="install-step-num">2</span>
                    <div class="install-step-body">
                        <div class="install-step-title">Scroll down and tap <strong>'Add to Home Screen'</strong></div>
                        <div class="install-step-desc">
                            In the menu sheet, scroll past your contacts &amp; apps until you see
                            <span class="install-step-inline-icon" title="Add to Home Screen">
                                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.3">
                                    <rect x="3" y="3" width="18" height="18" rx="3"/>
                                    <line x1="12" y1="8" x2="12" y2="16"/>
                                    <line x1="8" y1="12" x2="16" y2="12"/>
                                </svg>
                            </span>
                            <strong>Add to Home Screen</strong>.
                        </div>
                    </div>
                </div>
                <div class="install-step-item">
                    <span class="install-step-num">3</span>
                    <div class="install-step-body">
                        <div class="install-step-title">Tap <strong>'Add'</strong> in the top-right corner</div>
                        <div class="install-step-desc">Instant Crew will be added to your Home Screen with our app icon!</div>
                    </div>
                </div>
            `;
            isIosGuideActive = !isIPad;
            openInstallModal(
                'Follow these 3 quick steps in Safari to add Instant Crew to your Home Screen:',
                safariSteps,
                'Add to iPhone Home Screen'
            );
            return;
        }

        // ── Case C: Desktop (Chrome, Edge, etc.) ──
        if (deferredPrompt) {
            try {
                deferredPrompt.prompt();
                const choice = await deferredPrompt.userChoice;
                if (choice && choice.outcome === 'accepted') {
                    showInstallToast('Instant Crew added to your apps!');
                }
                deferredPrompt = null;
                window.deferredPrompt = null;
            } catch (err) {
                console.error('Desktop prompt error:', err);
            }
            return;
        }

        const desktopSteps = `
            <div class="install-step-item">
                <span class="install-step-num">1</span>
                <div>Click the <strong>Install</strong> icon <span class="install-step-inline-icon"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg></span> on the right side of the address bar</div>
            </div>
            <div class="install-step-item">
                <span class="install-step-num">2</span>
                <div>Or open the browser menu (<strong>⋮</strong> or <strong>⋯</strong>) and select <strong>'Install Instant Crew'</strong></div>
            </div>
            <div class="install-step-item">
                <span class="install-step-num">3</span>
                <div>Click <strong>'Install'</strong> to create a dedicated shortcut on your desktop</div>
            </div>
        `;
        openInstallModal('Add Instant Crew to your computer:', desktopSteps);
    };

    // 7. Auto-bind buttons on DOM ready
    function initPwaInstall() {
        if (isStandaloneMode()) {
            updateInstalledButtons();
        }

        document.querySelectorAll('.nav-install-btn').forEach(btn => {
            btn.removeEventListener('click', window.triggerAppInstall);
            btn.addEventListener('click', window.triggerAppInstall);
        });
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initPwaInstall);
    } else {
        initPwaInstall();
    }
})();
