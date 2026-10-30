/* ─────────────────────────────────────────────────────────────
   shared-data.js — UI helpers for job icons, maps and toast notices.
   Account, job, assignment and notification data is owned by the API.
   All icons use Tabler / Heroicons vector SVG specifications.
   ───────────────────────────────────────────────────────────── */

(function (window) {
    'use strict';


    /* ── Tabler / Heroicons Vector SVG Definitions ───────────── */
    const ICONS = {
        kitchen: (size = 20) => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon-tabler-chef-hat"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M12 3c1.918 0 3.52 1.35 3.91 3.151a4 4 0 0 1 2.09 7.723l0 7.126h-12v-7.126a4 4 0 1 1 2.092 -7.723a4 4 0 0 1 3.908 -3.151" /><path d="M6.161 17.009l11.839 -.009" /></svg>`,
        delivery: (size = 20) => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon-tabler-moped"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M16 17a2 2 0 1 0 4 0a2 2 0 1 0 -4 0" /><path d="M5 16v1a2 2 0 0 0 4 0v-5h-3a3 3 0 0 0 -3 3v1h10a6 6 0 0 1 5 -4v-5a2 2 0 0 0 -2 -2h-1" /><path d="M6 9l3 0" /></svg>`,
        helpers: (size = 20) => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon-tabler-users"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M9 7m-4 0a4 4 0 1 0 8 0a4 4 0 1 0 -8 0" /><path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2" /><path d="M16 3.13a4 4 0 0 1 0 7.75" /><path d="M21 21v-2a4 4 0 0 0 -3 -3.85" /></svg>`,
        package: (size = 20) => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon-tabler-package"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M12 3l8 4.5l0 9l-8 4.5l-8 -4.5l0 -9l8 -4.5" /><path d="M12 12l8 -4.5" /><path d="M12 12l0 9" /><path d="M12 12l-8 -4.5" /><path d="M16 5.25l-8 4.5" /></svg>`,
        tent: (size = 20) => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon-tabler-tent"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M11 14l4 6h6l-9 -16l-9 16h6l4 -6" /></svg>`,
        briefcase: (size = 20) => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon-tabler-briefcase"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M3 7m0 2a2 2 0 0 1 2 -2h14a2 2 0 0 1 2 2v9a2 2 0 0 1 -2 2h-14a2 2 0 0 1 -2 -2z" /><path d="M8 7v-2a2 2 0 0 1 2 -2h4a2 2 0 0 1 2 2v2" /><path d="M12 12l0 .01" /><path d="M3 13a20 20 0 0 0 18 0" /></svg>`,
        star: (size = 14, color = '#F59E0B') => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="${color}" stroke="${color}" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" class="icon-tabler-star"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M12 17.75l-6.172 3.245l1.179 -6.873l-5 -4.867l6.9 -1l3.086 -6.253l3.086 6.253l6.9 1l-5 4.867l1.179 6.873z" /></svg>`,
        check: (size = 18, color = '#10B981') => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="icon-tabler-check"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M5 12l5 5l10 -10" /></svg>`,
        checkCircle: (size = 20, color = '#10B981') => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon-tabler-circle-check"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M12 12m-9 0a9 9 0 1 0 18 0a9 9 0 1 0 -18 0" /><path d="M9 12l2 2l4 -4" /></svg>`,
        alert: (size = 18, color = '#F59E0B') => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon-tabler-alert-triangle"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M12 9v4" /><path d="M12 16v.01" /><path d="M12 3l9 17h-18l9 -17" /></svg>`,
        bolt: (size = 14, color = '#F59E0B') => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="${color}" stroke="${color}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon-tabler-bolt"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M13 3l-7 10h6l-2 8l9 -12h-6l2 -6" /></svg>`,
        bell: (size = 20) => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon-tabler-bell"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M10 5a2 2 0 1 1 4 0a7 7 0 0 1 4 6v3a4 4 0 0 0 2 3h-16a4 4 0 0 0 2 -3v-3a7 7 0 0 1 4 -6" /><path d="M9 17v1a3 3 0 0 0 6 0v-1" /></svg>`,
        clipboard: (size = 20) => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon-tabler-clipboard-text"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M9 5h-2a2 2 0 0 0 -2 2v12a2 2 0 0 0 2 2h10a2 2 0 0 0 2 -2v-12a2 2 0 0 0 -2 -2h-2" /><path d="M9 3m0 2a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2v0a2 2 0 0 1 -2 2h-2a2 2 0 0 1 -2 -2z" /><path d="M9 12h6" /><path d="M9 16h6" /></svg>`,
        party: (size = 20, color = '#10B981') => `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="icon-tabler-confetti"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M4 5h2" /><path d="M5 4v2" /><path d="M11.5 4l-.5 2" /><path d="M18 5h2" /><path d="M19 4v2" /><path d="M15 9l-1 1" /><path d="M18 13l2 -.5" /><path d="M18 19h2" /><path d="M19 18v2" /><path d="M14 16.518l-6.518 -6.518l-4.482 11l11 -4.482z" /></svg>`
    };

    function getRoleIcon(itemOrCategory, size = 20) {
        if (typeof itemOrCategory === 'string' && itemOrCategory.trim().startsWith('<svg')) {
            return itemOrCategory;
        }

        let cat = '';
        let role = '';
        let title = '';

        if (typeof itemOrCategory === 'object' && itemOrCategory !== null) {
            cat = (itemOrCategory.category || '').toLowerCase();
            role = (itemOrCategory.role || '').toLowerCase();
            title = (itemOrCategory.title || itemOrCategory.roleLabel || '').toLowerCase();
            if (itemOrCategory.emoji && typeof itemOrCategory.emoji === 'string' && itemOrCategory.emoji.trim().startsWith('<svg')) {
                return itemOrCategory.emoji;
            }
        } else if (typeof itemOrCategory === 'string') {
            cat = itemOrCategory.toLowerCase();
            role = itemOrCategory.toLowerCase();
        }

        const combined = `${cat} ${role} ${title}`;

        if (cat === 'kitchen' || combined.includes('cook') || combined.includes('prep') || combined.includes('dish') || combined.includes('chef') || combined.includes('kitchen') || combined.includes('server') || combined.includes('barista')) {
            return ICONS.kitchen(size);
        }
        if (cat === 'delivery' || combined.includes('delivery') || combined.includes('rider') || combined.includes('courier') || combined.includes('motorcycle') || combined.includes('driver')) {
            return ICONS.delivery(size);
        }
        if (combined.includes('warehouse') || combined.includes('logistics') || combined.includes('package') || combined.includes('staging') || combined.includes('stock')) {
            return ICONS.package(size);
        }
        if (combined.includes('event') || combined.includes('banquet') || combined.includes('setup') || combined.includes('staging') || combined.includes('booth')) {
            return ICONS.tent(size);
        }
        if (cat === 'helpers' || combined.includes('helper') || combined.includes('handyman') || combined.includes('cleaner') || combined.includes('crew') || combined.includes('labor')) {
            return ICONS.helpers(size);
        }

        return ICONS.briefcase(size);
    }

    function getNotificationIcon(type, size = 20) {
        if (type === 'accept') return ICONS.party(size, '#10B981');
        if (type === 'end') return ICONS.clipboard(size);
        if (type === 'match') return ICONS.bolt(size, '#F59E0B');
        if (type === 'transit' || type === 'on_the_way') return ICONS.delivery(size);
        if (type === 'arrived') return ICONS.checkCircle(size, '#10B981');
        return ICONS.bell(size);
    }

    function formatGoogleMapsUrl(locationStr, existingUrl) {
        if (existingUrl && typeof existingUrl === 'string' && (existingUrl.startsWith('http://') || existingUrl.startsWith('https://'))) {
            return existingUrl;
        }
        if (locationStr && typeof locationStr === 'string' && (locationStr.startsWith('http://') || locationStr.startsWith('https://'))) {
            return locationStr;
        }
        const query = (locationStr && typeof locationStr === 'string') ? locationStr.trim() : 'Cebu City';
        return `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(query)}`;
    }

    function initJobs() {}

    function getJobs() { return []; } // server APIs own jobs

    function saveJobs() { /* no browser-side job writes */ }

    // Add employer booking as an available job in the pool
    function addEmployerJob(booking) {
        const jobs = getJobs();
        const roleName = booking.roleLabel || booking.role || 'Crew Member';
        const cleanTitle = (booking.roleLabel || booking.role || 'Crew').replace(/\s*\((Full-Time|Part-Time)\)/gi, '').trim();
        const empType = booking.employmentType || 'full-time';
        const loc = booking.location || 'Cebu City';
        const mapLoc = booking.mapLocation || booking.mapAddress || `${loc} (Work Venue)`;
        const mapsUrl = formatGoogleMapsUrl(mapLoc, booking.googleMapsUrl);

        const newJob = {
            id: 'emp-shift-' + (booking.id || Date.now()),
            category: booking.category || 'delivery',
            role: roleName,
            title: cleanTitle,
            emoji: getRoleIcon(booking),
            venue: booking.venue || 'Instant Crew Employer',
            distance: booking.distance || '0.9 km away',
            commute: booking.commute || '6 min commute',
            timing: booking.timing || 'ASAP',
            rate: booking.offeredRate ? `₱${booking.offeredRate}` : (booking.crewRate || '₱85'),
            notes: `${loc} · Minimum shift`,
            location: loc,
            mapLocation: mapLoc,
            googleMapsUrl: mapsUrl,
            employmentType: empType,
            neededCrew: Number(booking.count) || 1,
            acceptedCount: 0,
            acceptedCrew: [],
            status: 'open',
            employerName: booking.name || 'Employer',
            employerEmail: booking.email || '',
            bookingRefId: booking.id,
            paid: booking.paid !== undefined ? booking.paid : true,
            paymentMethod: booking.paymentMethod || 'GCash',
            paymentAmount: booking.paymentAmount || null,
            escrowStatus: booking.escrowStatus || 'Secured in Escrow'
        };

        // Add to front of jobs pool
        jobs.unshift(newJob);
        saveJobs(jobs);

        // Notify worker side about the matched shift
        addNotification('worker', {
            id: Date.now(),
            title: 'New Matched Employer!',
            message: `${newJob.venue} is hiring for ${newJob.role} · ${empType === 'full-time' ? 'Full-Time' : 'Part-Time'}.`,
            time: 'Just now',
            type: 'match',
            jobId: newJob.id,
            role: newJob.role,
            employmentType: empType
        });

        return newJob;
    }

    // Rejected jobs by worker (vanish from that worker's feed)
    function getRejectedJobIds() {
        return [];
    }

    function rejectJob(jobId) {
        // Use POST /api/applicant/jobs/{id}/reject.
    }

    // Default seed contracts
    // Active contracts for worker
    function getActiveContracts() { return []; } // read assignments from server APIs

    function saveActiveContracts() { /* no browser-side contract writes */ }

    // Accept a job
    function acceptJob(jobId, workerName) {
        const jobs = getJobs();
        const job = jobs.find(j => j.id === jobId);
        if (!job) return { success: false, reason: 'Job not found' };

        const contracts = getActiveContracts().filter(c => c.status === 'active');
        const hasFullTime = contracts.some(c => c.employmentType === 'full-time');
        const partTimeCount = contracts.filter(c => c.employmentType === 'part-time').length;

        // Enforce capacity constraints:
        // 1. If worker has 1 active full-time contract, cannot accept any other jobs.
        // 2. If worker has 1 part-time contract, cannot accept a full-time contract.
        // 3. If worker has 2 part-time contracts, cannot accept any further jobs.
        if (hasFullTime) {
            return {
                success: false,
                reason: 'You currently have an active Full-Time contract. You cannot accept other jobs.'
            };
        }

        if (job.employmentType === 'full-time' && partTimeCount > 0) {
            return {
                success: false,
                reason: 'You have an active Part-Time job. You cannot accept a Full-Time job concurrently.'
            };
        }

        if (job.employmentType === 'part-time' && partTimeCount >= 2) {
            return {
                success: false,
                reason: 'You have reached the maximum of 2 Part-Time contracts.'
            };
        }

        // Increment acceptedCount
        job.acceptedCount = (job.acceptedCount || 0) + 1;
        if (!job.acceptedCrew) job.acceptedCrew = [];
        job.acceptedCrew.push(workerName);

        if (job.acceptedCount >= job.neededCrew) {
            job.status = 'filled';
        }
        saveJobs(jobs);

        // Add to active contracts
        const contract = {
            id: 'contract-' + Date.now(),
            jobId: job.id,
            bookingRefId: job.bookingRefId || null,
            title: (job.title || job.role || 'Crew').replace(/\s*\((Full-Time|Part-Time)\)/gi, '').trim(),
            role: job.role,
            category: job.category,
            emoji: getRoleIcon(job),
            venue: job.venue,
            rate: job.rate,
            timing: job.timing,
            employmentType: job.employmentType,
            status: 'active',
            startedAt: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
            employerName: job.employerName || 'Employer',
            workerName: workerName
        };

        const allContracts = getActiveContracts();
        allContracts.unshift(contract);
        saveActiveContracts(allContracts);

        // Send push notification to Employer
        addNotification('employer', {
            id: Date.now(),
            title: 'Offer Accepted!',
            message: `${workerName} has accepted your ${job.role} shift offer!`,
            time: 'Just now',
            type: 'accept',
            jobId: job.id,
            workerName: workerName
        });

        return { success: true, contract };
    }

    // End work / contract (invoked by either Employee or Employer)
    function endContract(contractOrJobId, endedByRole, endedByName) {
        const contracts = getActiveContracts();
        let targetContract = null;

        contracts.forEach(c => {
            if ((c.id === contractOrJobId || c.jobId === contractOrJobId || (c.bookingRefId && String(c.bookingRefId) === String(contractOrJobId))) && c.status === 'active') {
                c.status = 'ended';
                c.endedAt = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
                c.endedBy = endedByRole;
                targetContract = c;
            }
        });

        saveActiveContracts(contracts);

        // Also update any matching job in the pool
        const jobs = getJobs();
        jobs.forEach(j => {
            if (j.id === contractOrJobId || (targetContract && j.id === targetContract.jobId)) {
                j.contractStatus = 'ended';
            }
        });
        saveJobs(jobs);

        // Dispatch notifications to the other party
        if (endedByRole === 'worker') {
            addNotification('employer', {
                id: Date.now(),
                title: 'Contract Concluded',
                message: `${endedByName || 'Crew member'} has completed and ended the contract for ${targetContract ? targetContract.title : 'the shift'}.`,
                time: 'Just now',
                type: 'end'
            });
        } else {
            addNotification('worker', {
                id: Date.now(),
                title: 'Contract Concluded',
                message: `Employer ${endedByName || ''} has concluded the contract for ${targetContract ? targetContract.title : 'your shift'}.`,
                time: 'Just now',
                type: 'end'
            });
        }

        return { success: true, targetContract };
    }

    // Update contract transit status (on the way / arrived) & notify employer
    function updateContractTransit(contractOrJobId, transitStatus, workerName) {
        const contracts = getActiveContracts();
        let targetContract = null;
        const nowTime = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

        contracts.forEach(c => {
            if ((c.id === contractOrJobId || c.jobId === contractOrJobId || (c.bookingRefId && String(c.bookingRefId) === String(contractOrJobId))) && c.status === 'active') {
                c.transitStatus = transitStatus;
                if (transitStatus === 'on_the_way') {
                    c.departedAt = nowTime;
                } else if (transitStatus === 'arrived') {
                    c.arrivedAt = nowTime;
                }
                targetContract = c;
            }
        });

        if (targetContract) {
            saveActiveContracts(contracts);

            // Also sync matching job in the jobs pool
            const jobs = getJobs();
            jobs.forEach(j => {
                if (j.id === contractOrJobId || j.id === targetContract.jobId || (j.bookingRefId && String(j.bookingRefId) === String(contractOrJobId))) {
                    j.transitStatus = transitStatus;
                    if (transitStatus === 'on_the_way') j.departedAt = nowTime;
                    if (transitStatus === 'arrived') j.arrivedAt = nowTime;
                }
            });
            saveJobs(jobs);

            const crewName = workerName || targetContract.workerName || 'Crew member';
            const roleName = targetContract.role || targetContract.title || 'crew';
            const venueName = targetContract.venue || 'your location';

            // Send instant notification to Employer
            if (transitStatus === 'on_the_way') {
                addNotification('employer', {
                    id: Date.now(),
                    title: 'Crew is On the Way',
                    message: `${crewName} has departed and is on the way to ${venueName} for the ${roleName} shift.`,
                    time: 'Just now',
                    type: 'transit',
                    jobId: targetContract.jobId,
                    contractId: targetContract.id,
                    departedAt: nowTime,
                    workerName: crewName
                });
            } else if (transitStatus === 'arrived') {
                addNotification('employer', {
                    id: Date.now(),
                    title: 'Crew Has Arrived',
                    message: `${crewName} has arrived at ${venueName} for the ${roleName} shift.`,
                    time: 'Just now',
                    type: 'arrived',
                    jobId: targetContract.jobId,
                    contractId: targetContract.id,
                    arrivedAt: nowTime,
                    workerName: crewName
                });
            }

            notifyStateChange('transit');
            return { success: true, contract: targetContract };
        }

        return { success: false, reason: 'Active contract not found' };
    }

    // Notifications Store
    function getNotifications() { return []; }
    function addNotification() { /* notifications persist through /api/notifications */ }
    function markAllNotificationsRead() { /* use POST /api/notifications/read */ }
    function clearNotifications() { /* use DELETE /api/notifications */ }

    // In-app push toasts are disabled so alerts are delivered exclusively via
    // external phone/desktop system notifications without duplicate in-app popups.
    function showPushToast() {
        const existing = document.querySelector('.db-push-toast');
        if (existing) existing.remove();
    }

    // System (OS-level) notifications. Mobile browsers reject `new Notification()`
    // and only show notifications through a service worker registration, so the
    // worker is registered here and the constructor is the desktop fallback.
    let notificationRegistration = null;
    function getNotificationRegistration() {
        if (!('serviceWorker' in navigator)) return Promise.resolve(null);
        if (!notificationRegistration) {
            notificationRegistration = navigator.serviceWorker.register('/sw.js')
                .then(() => navigator.serviceWorker.ready)
                .catch(() => null);
        }
        return notificationRegistration;
    }

    // Registers the worker and asks for permission on the first tap, since
    // browsers ignore permission prompts that are not tied to a user gesture.
    // With `push`, a granted permission also subscribes this device to Web Push.
    function enableSystemNotifications({ push = false } = {}) {
        if (!('Notification' in window)) return;
        getNotificationRegistration();
        if (Notification.permission === 'granted') {
            if (push) syncPushSubscription();
            return;
        }
        if (Notification.permission !== 'default') return;
        document.addEventListener('click', () => {
            try {
                Promise.resolve(Notification.requestPermission()).then((result) => {
                    if (push && result === 'granted') syncPushSubscription();
                });
            } catch { }
        }, { once: true });
    }

    // Web Push lets the server reach this device while the app is closed (on iOS
    // only once the site is added to the Home Screen). The subscription belongs
    // to the logged-in employer, so logout removes it again.
    let pushActive = false;
    function base64UrlToBytes(value) {
        const base64 = (value + '='.repeat((4 - value.length % 4) % 4)).replace(/-/g, '+').replace(/_/g, '/');
        return Uint8Array.from(atob(base64), (c) => c.charCodeAt(0));
    }

    function sameKey(buffer, bytes) {
        const current = new Uint8Array(buffer);
        return current.length === bytes.length && current.every((b, i) => b === bytes[i]);
    }

    function pushRoleQuery() {
        const p = (window.location && window.location.pathname) || '';
        if (/applicant\.html/i.test(p)) return '?role=applicant';
        if (/employer\.html/i.test(p)) return '?role=employer';
        return '';
    }

    async function syncPushSubscription() {
        if (!('PushManager' in window) || !('Notification' in window) || Notification.permission !== 'granted') return false;
        try {
            const registration = await getNotificationRegistration();
            if (!registration || !registration.pushManager) return false;
            const roleQuery = pushRoleQuery();
            const keyResponse = await fetch('/api/push/key' + roleQuery, { credentials: 'same-origin' });
            if (!keyResponse.ok) return false; // logged out, or push not configured on the server
            const { public_key: publicKey } = await keyResponse.json();
            const serverKey = base64UrlToBytes(publicKey);
            let subscription = await registration.pushManager.getSubscription();
            // A subscription made with an older server key can no longer receive pushes.
            const existingKey = subscription && subscription.options && subscription.options.applicationServerKey;
            if (existingKey && !sameKey(existingKey, serverKey)) {
                await subscription.unsubscribe();
                subscription = null;
            }
            if (!subscription) {
                subscription = await registration.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: serverKey });
            }
            const saved = await fetch('/api/push/subscribe' + roleQuery, {
                method: 'POST',
                credentials: 'same-origin',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(subscription.toJSON())
            });
            pushActive = saved.ok;
        } catch (error) {
            console.warn('[push]', error.message);
            pushActive = false;
        }
        return pushActive;
    }

    // Call before logging out: the server only accepts this while the session is valid.
    async function removePushSubscription() {
        pushActive = false;
        try {
            const registration = await getNotificationRegistration();
            const subscription = registration && registration.pushManager && await registration.pushManager.getSubscription();
            if (!subscription) return;
            await fetch('/api/push/unsubscribe' + pushRoleQuery(), {
                method: 'POST',
                credentials: 'same-origin',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ endpoint: subscription.endpoint })
            });
            await subscription.unsubscribe();
        } catch { }
    }

    function isPushActive() { return pushActive; }

    async function showSystemNotification(title, body, { tag, url } = {}) {
        if (!('Notification' in window) || Notification.permission !== 'granted') return;
        const options = {
            body,
            icon: '/images/icon-192.png',
            badge: '/images/Instalogo.png',
            tag: tag || undefined,
            renotify: Boolean(tag),
            vibrate: [120, 60, 120],
            data: { url: url || window.location.href }
        };
        try {
            const registration = await getNotificationRegistration();
            if (registration && registration.showNotification) {
                await registration.showNotification(title, options);
                return;
            }
        } catch { }
        try { new Notification(title, options); } catch { }
    }

    // Broadcast helper
    function notifyStateChange(detail) {
        window.dispatchEvent(new CustomEvent('ic_state_change', { detail }));
    }

    // 1-hour arrival limit countdown. Elements with data-arrival-deadline (epoch ms,
    // derived from the server's seconds remaining) are refreshed every second.
    function formatArrivalCountdown(deadlineMs) {
        const left = Math.max(0, Math.floor((deadlineMs - Date.now()) / 1000));
        if (left === 0) return '1-Hr Arrival Limit: time is up';
        return `1-Hr Arrival Limit: ${Math.floor(left / 60)}m ${left % 60}s left`;
    }

    function arrivalDeadlineFromSeconds(seconds) {
        const secs = Number(seconds || 0);
        return secs > 0 ? Date.now() + secs * 1000 : 0;
    }

    function tickArrivalCountdowns() {
        document.querySelectorAll('[data-arrival-deadline]').forEach(el => {
            const deadline = Number(el.dataset.arrivalDeadline);
            if (deadline > 0) el.textContent = formatArrivalCountdown(deadline);
        });
    }
    setInterval(tickArrivalCountdowns, 1000);

    // Escape text another user wrote (names, venues, notifications) before it goes into innerHTML.
    // GCash can't send the user back to a web page after paying, so once they
    // switch back from the app we point them at the receipt step. The returned
    // function is true once per "Open GCash" tap, letting the caller auto-submit.
    function watchGcashReturn(wrap) {
        const link = wrap && wrap.querySelector('a[href="gcash://"]');
        const upload = wrap && wrap.querySelector('.db-proof-upload-section');
        if (!link || !upload) return () => false;
        let away = false;
        let armed = false;
        link.addEventListener('click', () => { away = true; armed = true; });
        document.addEventListener('visibilitychange', () => {
            if (document.visibilityState !== 'visible' || !away) return;
            away = false;
            upload.scrollIntoView({ behavior: 'smooth', block: 'center' });
            upload.classList.remove('db-proof-attention');
            void upload.offsetWidth; // restart the highlight animation
            upload.classList.add('db-proof-attention');
        });
        return () => {
            const was = armed;
            armed = false;
            return was;
        };
    }

    // In the installed (standalone) app, following a download link opens the
    // image full-screen with no way back, so save the QR without leaving the page.
    document.addEventListener('click', async (e) => {
        const link = e.target.closest && e.target.closest('.db-gcash-actions a[download]');
        if (!link) return;
        e.preventDefault();
        try {
            const blob = await (await fetch(link.href)).blob();
            const file = new File([blob], link.getAttribute('download'), { type: blob.type || 'image/png' });
            const isIOS = /iPad|iPhone|iPod/.test(navigator.userAgent) || (navigator.platform === 'MacIntel' && navigator.maxTouchPoints > 1);
            if (isIOS && navigator.canShare && navigator.canShare({ files: [file] })) {
                await navigator.share({ files: [file] }); // iOS share sheet offers "Save Image"
                return;
            }
            const url = URL.createObjectURL(blob);
            const a = Object.assign(document.createElement('a'), { href: url, download: file.name });
            document.body.appendChild(a);
            a.click();
            a.remove();
            setTimeout(() => URL.revokeObjectURL(url), 1000);
        } catch (err) {
            if (err && err.name === 'AbortError') return; // user closed the share sheet
            console.warn('Save QR failed', err);
        }
    });

    function escHtml(s) {
        return String(s == null ? '' : s).replace(/[&<>"']/g, function (c) {
            return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
        });
    }

    // Export to global window object
    window.InstantCrewShared = {
        escHtml,
        watchGcashReturn,
        formatArrivalCountdown,
        arrivalDeadlineFromSeconds,
        tickArrivalCountdowns,
        initJobs,
        getJobs,
        saveJobs,
        addEmployerJob,
        getRejectedJobIds,
        rejectJob,
        getActiveContracts,
        saveActiveContracts,
        acceptJob,
        endContract,
        updateContractTransit,
        getNotifications,
        addNotification,
        markAllNotificationsRead,
        clearNotifications,
        showPushToast,
        enableSystemNotifications,
        showSystemNotification,
        syncPushSubscription,
        removePushSubscription,
        isPushActive,
        formatGoogleMapsUrl,
        ICONS,
        getRoleIcon,
        getNotificationIcon
    };

    // Initialize default jobs on load
    initJobs();

})(window);
