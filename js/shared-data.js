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

    // Push Toast Helper using pure SVG vector icons
    function showPushToast(title, message, iconOrType, onClick) {
        // Remove existing toast if any
        const existing = document.querySelector('.db-push-toast');
        if (existing) existing.remove();

        let iconMarkup = '';
        if (typeof iconOrType === 'string' && iconOrType.trim().startsWith('<svg')) {
            iconMarkup = iconOrType;
        } else if (iconOrType === 'accept' || iconOrType === 'party') {
            iconMarkup = ICONS.party(20, '#10B981');
        } else if (iconOrType === 'end' || iconOrType === 'clipboard') {
            iconMarkup = ICONS.clipboard(20);
        } else if (iconOrType === 'match' || iconOrType === 'bolt') {
            iconMarkup = ICONS.bolt(20, '#F59E0B');
        } else if (iconOrType === 'transit' || iconOrType === 'on_the_way') {
            iconMarkup = ICONS.delivery(20);
        } else if (iconOrType === 'arrived') {
            iconMarkup = ICONS.checkCircle(20, '#10B981');
        } else if (iconOrType === 'check' || iconOrType === '✓') {
            iconMarkup = ICONS.checkCircle(20, '#10B981');
        } else if (iconOrType === 'alert') {
            iconMarkup = ICONS.alert(20, '#F59E0B');
        } else {
            iconMarkup = ICONS.bell(20);
        }

        const toast = document.createElement('div');
        toast.className = 'db-push-toast';
        if (onClick) {
            toast.style.cursor = 'pointer';
            toast.setAttribute('title', 'Click to view offer');
            toast.addEventListener('click', (e) => {
                onClick(e);
            });
        }
        toast.innerHTML = `
            <span class="db-push-toast-icon">${iconMarkup}</span>
            <div class="db-push-toast-body">
                <p class="db-push-toast-title">${title}</p>
                <p class="db-push-toast-msg">${message}</p>
            </div>
            <button type="button" class="db-push-toast-close" aria-label="Dismiss">&times;</button>
        `;

        toast.querySelector('.db-push-toast-close').addEventListener('click', (e) => {
            e.stopPropagation();
            toast.remove();
        });

        document.body.appendChild(toast);

        // Play gentle notification chime
        try {
            const AudioCtx = window.AudioContext || window.webkitAudioContext;
            if (AudioCtx) {
                const ctx = new AudioCtx();
                if (ctx.state === 'suspended') {
                    ctx.resume().catch(() => { });
                }
                const now = ctx.currentTime;
                const osc = ctx.createOscillator();
                const gain = ctx.createGain();
                osc.type = 'sine';
                osc.frequency.setValueAtTime(587.33, now); // D5
                osc.frequency.setValueAtTime(880, now + 0.08); // A5
                gain.gain.setValueAtTime(0.08, now);
                gain.gain.exponentialRampToValueAtTime(0.001, now + 0.3);
                osc.connect(gain);
                gain.connect(ctx.destination);
                osc.start(now);
                osc.stop(now + 0.3);
            }
        } catch { }

        // Auto remove after 5 seconds
        setTimeout(() => {
            if (toast.parentNode) {
                toast.style.animation = 'pushToastIn 0.3s reverse forwards';
                setTimeout(() => toast.remove(), 300);
            }
        }, 5000);
    }

    // Broadcast helper
    function notifyStateChange(detail) {
        window.dispatchEvent(new CustomEvent('ic_state_change', { detail }));
    }

    // Export to global window object
    window.InstantCrewShared = {
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
        formatGoogleMapsUrl,
        ICONS,
        getRoleIcon,
        getNotificationIcon
    };

    // Initialize default jobs on load
    initJobs();

})(window);
