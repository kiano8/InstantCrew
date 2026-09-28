/* ─────────────────────────────────────────────────────────────
   shared-data.js — Instant Crew Shared State & Notifications
   Synchronizes Jobs, Bookings, Contracts & Push Notifications
   across Employer and Applicant portals using localStorage.
   All icons use Tabler / Heroicons vector SVG specifications.
   ───────────────────────────────────────────────────────────── */

(function (window) {
    'use strict';

    const STORAGE_KEY_JOBS = 'ic_shared_jobs';
    const STORAGE_KEY_NOTIFS_WORKER = 'ic_worker_notifications';
    const STORAGE_KEY_NOTIFS_EMPLOYER = 'ic_employer_notifications';
    const STORAGE_KEY_REJECTED = 'ic_worker_rejected_jobs';
    const STORAGE_KEY_ACTIVE_CONTRACTS = 'ic_active_contracts';

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

    // Initial seed shifts
    const DEFAULT_SHIFTS = [
        {
            id: 'shift-1',
            category: 'kitchen',
            role: 'Line Cook',
            title: 'Line Cook / Prep Assistant',
            emoji: ICONS.kitchen(20),
            venue: 'Bistro Moderne — Downtown',
            distance: '1.2 km away',
            commute: '8 min commute',
            timing: 'Today, 1:00 PM – 5:00 PM (4 hrs)',
            rate: '₱95',
            notes: 'Kitchen attire & apron provided on site',
            employmentType: 'part-time',
            neededCrew: 2,
            acceptedCount: 0,
            acceptedCrew: [],
            status: 'open',
            employerName: 'Bistro Moderne'
        },
        {
            id: 'shift-1b',
            category: 'kitchen',
            role: 'Line Cook',
            title: 'Line Cook / Station Specialist',
            emoji: ICONS.kitchen(20),
            venue: 'Grand Plaza Trattoria — Midtown',
            distance: '1.8 km away',
            commute: '12 min commute',
            timing: 'Tonight, 6:00 PM – 10:00 PM (4 hrs)',
            rate: '₱100',
            notes: 'Short-order grill & sautee station',
            employmentType: 'part-time',
            neededCrew: 2,
            acceptedCount: 0,
            acceptedCrew: [],
            status: 'open',
            employerName: 'Grand Plaza Trattoria'
        },
        {
            id: 'shift-1c',
            category: 'kitchen',
            role: 'Line Cook',
            title: 'Full-Time Lead Line Cook',
            emoji: ICONS.kitchen(20),
            venue: 'Heritage Grill & Smokehouse',
            distance: '2.4 km away',
            commute: '15 min commute',
            timing: 'Full-time Day Shift (8 hrs / day)',
            rate: '₱110',
            notes: 'Full-time kitchen team member',
            employmentType: 'full-time',
            neededCrew: 2,
            acceptedCount: 0,
            acceptedCrew: [],
            status: 'open',
            employerName: 'Heritage Grill'
        },
        {
            id: 'shift-1m',
            category: 'kitchen',
            role: 'Line Cook',
            title: 'BGC Bistro Line Cook',
            emoji: ICONS.kitchen(20),
            venue: 'Makati & BGC Gourmet Bistro',
            location: 'Manila',
            mapLocation: 'BGC High Street, Manila',
            distance: '2.1 km away',
            commute: '15 min commute',
            timing: 'Full-time Day Shift (8 hrs / day)',
            rate: '₱120',
            notes: 'Full-time hot line cook',
            employmentType: 'full-time',
            neededCrew: 2,
            acceptedCount: 0,
            acceptedCrew: [],
            status: 'open',
            employerName: 'BGC Gourmet Bistro'
        },
        {
            id: 'shift-2',
            category: 'delivery',
            role: 'Motorcycle Rider',
            title: 'Express Delivery Rider',
            emoji: ICONS.delivery(20),
            venue: 'Metro Logistics Express Hub',
            distance: '0.8 km away',
            commute: '4 min commute',
            timing: 'Full-time Day Shift (8 hrs / day)',
            rate: '₱85',
            notes: 'Motorcycle & valid license required',
            employmentType: 'full-time',
            neededCrew: 1,
            acceptedCount: 0,
            acceptedCrew: [],
            status: 'open',
            employerName: 'Metro Logistics'
        },
        {
            id: 'shift-2b',
            category: 'delivery',
            role: 'Motorcycle Rider',
            title: 'Afternoon Parcel Courier',
            emoji: ICONS.delivery(20),
            venue: 'SwiftCourier Express Hub',
            distance: '1.5 km away',
            commute: '10 min commute',
            timing: 'Today, 2:00 PM – 6:00 PM (4 hrs)',
            rate: '₱90',
            notes: 'Rapid parcel delivery route',
            employmentType: 'part-time',
            neededCrew: 2,
            acceptedCount: 0,
            acceptedCrew: [],
            status: 'open',
            employerName: 'SwiftCourier'
        },
        {
            id: 'shift-2c',
            category: 'delivery',
            role: 'Motorcycle Rider',
            title: 'Evening Express Delivery Rider',
            emoji: ICONS.delivery(20),
            venue: 'CityDash Parcel Service',
            location: 'Manila',
            mapLocation: 'Bonifacio Global City, Manila',
            distance: '1.9 km away',
            commute: '12 min commute',
            timing: 'Tonight, 5:00 PM – 9:00 PM (4 hrs)',
            rate: '₱88',
            notes: 'Gas allowance & thermal bag provided',
            employmentType: 'part-time',
            neededCrew: 2,
            acceptedCount: 0,
            acceptedCrew: [],
            status: 'open',
            employerName: 'CityDash'
        },
        {
            id: 'shift-3',
            category: 'helpers',
            role: 'Event Helper',
            title: 'Event Setup & Banquet Helper',
            emoji: ICONS.tent(20),
            venue: 'Grand Ballroom & Pavilion',
            distance: '2.3 km away',
            commute: '14 min commute',
            timing: 'Tomorrow, 9:00 AM – 2:00 PM (5 hrs)',
            rate: '₱75',
            notes: 'Staging, seating & guest support',
            employmentType: 'part-time',
            neededCrew: 3,
            acceptedCount: 1,
            acceptedCrew: ['Beth T.'],
            status: 'open',
            employerName: 'Grand Ballroom'
        },
        {
            id: 'shift-4',
            category: 'kitchen',
            role: 'Dishwasher',
            title: 'Kitchen Steward / Dishwasher',
            emoji: ICONS.kitchen(20),
            venue: 'Harbor Seafood Grill',
            distance: '1.7 km away',
            commute: '10 min commute',
            timing: 'Tonight, 6:00 PM – 11:00 PM (5 hrs)',
            rate: '₱80',
            notes: 'Sanitation equipment provided',
            employmentType: 'part-time',
            neededCrew: 1,
            acceptedCount: 0,
            acceptedCrew: [],
            status: 'open',
            employerName: 'Harbor Seafood Grill'
        },
        {
            id: 'shift-5',
            category: 'helpers',
            role: 'Warehouse Helper',
            title: 'Warehouse Logistics Staging',
            emoji: ICONS.package(20),
            venue: 'Central Distribution Center',
            distance: '3.1 km away',
            commute: '18 min commute',
            timing: 'Full-time Morning Shift (8 hrs / day)',
            rate: '₱90',
            notes: 'Safe lifting & package sorting',
            employmentType: 'full-time',
            neededCrew: 2,
            acceptedCount: 0,
            acceptedCrew: [],
            status: 'open',
            employerName: 'Central Distribution'
        }
    ];

    // Seed jobs if not yet in localStorage
    function initJobs() {
        try {
            const raw = localStorage.getItem(STORAGE_KEY_JOBS);
            if (!raw) {
                localStorage.setItem(STORAGE_KEY_JOBS, JSON.stringify(DEFAULT_SHIFTS));
            } else {
                // Ensure default shifts exist in jobs pool
                const existing = JSON.parse(raw) || [];
                let modified = false;
                DEFAULT_SHIFTS.forEach(ds => {
                    if (!existing.some(j => j.id === ds.id)) {
                        existing.push(ds);
                        modified = true;
                    }
                });
                if (modified) {
                    localStorage.setItem(STORAGE_KEY_JOBS, JSON.stringify(existing));
                }
            }
        } catch (e) {
            console.warn('Storage not accessible:', e);
        }
    }

    function getJobs() {
        initJobs();
        try {
            const jobs = JSON.parse(localStorage.getItem(STORAGE_KEY_JOBS)) || DEFAULT_SHIFTS;
            let updated = false;
            // Upgrade any raw emoji strings to Tabler SVGs & clean titles
            jobs.forEach(j => {
                if (!j.emoji || !j.emoji.trim().startsWith('<svg')) {
                    j.emoji = getRoleIcon(j);
                    updated = true;
                }
                if (j.title && /\s*\((Full-Time|Part-Time)\)/i.test(j.title)) {
                    j.title = j.title.replace(/\s*\((Full-Time|Part-Time)\)/gi, '').trim();
                    updated = true;
                }
                if (!j.location) {
                    j.location = 'Cebu City';
                    updated = true;
                }
                if (!j.mapLocation) {
                    j.mapLocation = j.location || (j.notes ? j.notes.split('·')[0].trim() : 'Cebu City');
                    updated = true;
                }
                if (!j.googleMapsUrl) {
                    j.googleMapsUrl = formatGoogleMapsUrl(j.mapLocation);
                    updated = true;
                }
            });
            if (updated) {
                localStorage.setItem(STORAGE_KEY_JOBS, JSON.stringify(jobs));
            }
            return jobs;
        } catch {
            return DEFAULT_SHIFTS;
        }
    }

    function saveJobs(jobs) {
        try {
            localStorage.setItem(STORAGE_KEY_JOBS, JSON.stringify(jobs));
            notifyStateChange('jobs');
        } catch (e) {
            console.error('Failed to save jobs:', e);
        }
    }

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
            employerName: booking.name || 'Sample Employer',
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
        try {
            return JSON.parse(localStorage.getItem(STORAGE_KEY_REJECTED)) || [];
        } catch {
            return [];
        }
    }

    function rejectJob(jobId) {
        const rejected = getRejectedJobIds();
        if (!rejected.includes(jobId)) {
            rejected.push(jobId);
            localStorage.setItem(STORAGE_KEY_REJECTED, JSON.stringify(rejected));
            notifyStateChange('rejected');
        }
    }

    // Default seed contracts
    const DEFAULT_CONTRACTS = [
        {
            id: 'contract-demo-1',
            jobId: 'shift-3',
            title: 'Event Setup & Banquet Helper',
            role: 'Event Helper',
            category: 'helpers',
            emoji: ICONS.tent(20),
            venue: 'Grand Ballroom & Pavilion',
            rate: '₱75',
            timing: 'Tomorrow, 9:00 AM – 2:00 PM (5 hrs)',
            employmentType: 'part-time',
            status: 'ended',
            endedAt: 'Yesterday, 2:00 PM',
            employerName: 'Grand Ballroom'
        },
        {
            id: 'contract-demo-2',
            jobId: 'shift-1',
            title: 'Line Cook / Prep Assistant',
            role: 'Line Cook',
            category: 'kitchen',
            emoji: ICONS.kitchen(20),
            venue: 'Bistro Moderne — Downtown',
            rate: '₱95',
            timing: 'Today, 1:00 PM – 5:00 PM (4 hrs)',
            employmentType: 'part-time',
            status: 'ended',
            endedAt: '5:00 PM',
            employerName: 'Bistro Moderne'
        },
        {
            id: 'contract-demo-3',
            jobId: 'shift-2',
            title: 'Express Delivery Rider',
            role: 'Motorcycle Rider',
            category: 'delivery',
            emoji: ICONS.delivery(20),
            venue: 'Metro Logistics Express Hub',
            rate: '₱85',
            timing: 'Full-time Day Shift (8 hrs / day)',
            employmentType: 'full-time',
            status: 'ended',
            endedAt: 'Yesterday',
            employerName: 'Metro Logistics'
        }
    ];

    // Active contracts for worker
    function getActiveContracts() {
        try {
            const raw = localStorage.getItem(STORAGE_KEY_ACTIVE_CONTRACTS);
            if (!raw) {
                localStorage.setItem(STORAGE_KEY_ACTIVE_CONTRACTS, JSON.stringify(DEFAULT_CONTRACTS));
                return DEFAULT_CONTRACTS;
            }
            const list = JSON.parse(raw) || [];
            let changed = false;
            list.forEach(c => {
                // Ensure contract-demo-1 is ended so applicant starts clean
                if (c.id === 'contract-demo-1' && c.status === 'active') {
                    c.status = 'ended';
                    c.endedAt = 'Yesterday, 2:00 PM';
                    changed = true;
                }
                if (c.status === 'completed') {
                    c.status = 'ended';
                    changed = true;
                }
                // Upgrade any raw emoji strings to Tabler SVGs & clean titles
                if (!c.emoji || !c.emoji.trim().startsWith('<svg')) {
                    c.emoji = getRoleIcon(c);
                    changed = true;
                }
                if (c.title && /\s*\((Full-Time|Part-Time)\)/i.test(c.title)) {
                    c.title = c.title.replace(/\s*\((Full-Time|Part-Time)\)/gi, '').trim();
                    changed = true;
                }
            });
            if (changed) {
                localStorage.setItem(STORAGE_KEY_ACTIVE_CONTRACTS, JSON.stringify(list));
            }
            return list;
        } catch {
            return DEFAULT_CONTRACTS;
        }
    }

    function saveActiveContracts(contracts) {
        try {
            localStorage.setItem(STORAGE_KEY_ACTIVE_CONTRACTS, JSON.stringify(contracts));
            notifyStateChange('contracts');
        } catch (e) {
            console.error('Failed to save contracts:', e);
        }
    }

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

    // Notifications Store
    function getNotifications(role) {
        const key = role === 'employer' ? STORAGE_KEY_NOTIFS_EMPLOYER : STORAGE_KEY_NOTIFS_WORKER;
        try {
            return JSON.parse(localStorage.getItem(key)) || [];
        } catch {
            return [];
        }
    }

    function addNotification(role, notif) {
        const key = role === 'employer' ? STORAGE_KEY_NOTIFS_EMPLOYER : STORAGE_KEY_NOTIFS_WORKER;
        const list = getNotifications(role);
        list.unshift({
            id: notif.id || Date.now(),
            title: notif.title || 'Notification',
            message: notif.message || '',
            time: notif.time || 'Just now',
            type: notif.type || 'info',
            read: false,
            ...notif
        });
        // Keep max 30 notifications
        if (list.length > 30) list.length = 30;
        try {
            localStorage.setItem(key, JSON.stringify(list));
            notifyStateChange('notif_' + role);
        } catch (e) {
            console.error('Error saving notification:', e);
        }
    }

    function markAllNotificationsRead(role) {
        const key = role === 'employer' ? STORAGE_KEY_NOTIFS_EMPLOYER : STORAGE_KEY_NOTIFS_WORKER;
        const list = getNotifications(role);
        list.forEach(n => n.read = true);
        try {
            localStorage.setItem(key, JSON.stringify(list));
            notifyStateChange('notif_' + role);
        } catch { }
    }

    function clearNotifications(role) {
        const key = role === 'employer' ? STORAGE_KEY_NOTIFS_EMPLOYER : STORAGE_KEY_NOTIFS_WORKER;
        try {
            localStorage.setItem(key, JSON.stringify([]));
            notifyStateChange('notif_' + role);
        } catch { }
    }

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
        } else if (iconOrType === 'end' || iconOrType === 'clipboard' || iconOrType === '📋') {
            iconMarkup = ICONS.clipboard(20);
        } else if (iconOrType === 'match' || iconOrType === 'bolt' || iconOrType === '⚡') {
            iconMarkup = ICONS.bolt(20, '#F59E0B');
        } else if (iconOrType === 'check' || iconOrType === '✓') {
            iconMarkup = ICONS.checkCircle(20, '#10B981');
        } else if (iconOrType === 'alert' || iconOrType === '⚠️') {
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
