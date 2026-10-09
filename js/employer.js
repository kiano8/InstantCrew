/* ─────────────────────────────────────────────────────────────
   dashboard.js  —  Instant Crew Employer Dashboard
   Pure Vanilla JS — no dependencies
   ───────────────────────────────────────────────────────────── */

'use strict';

/* ─── Sample Data ─────────────────────────────────────────── */

const ROLES = {
  kitchen: [
    { id: 'chef', label: 'Chef' },
    { id: 'sous-chef', label: 'Sous Chef' },
    { id: 'line-cook', label: 'Line Cook' },
    { id: 'prep-assistant', label: 'Prep Assistant' },
    { id: 'kitchen-helper', label: 'Kitchen Helper' },
    { id: 'dishwasher', label: 'Dishwasher' },
    { id: 'pastry-chef', label: 'Pastry Chef' },
    { id: 'grill-cook', label: 'Grill Cook' },
    { id: 'barista', label: 'Barista' },
    { id: 'bartender', label: 'Bartender' },
    { id: 'food-runner', label: 'Food Runner' },
    { id: 'buffet-server', label: 'Buffet Server' },
    { id: 'kitchen-steward', label: 'Kitchen Steward' },
    { id: 'catering-assistant', label: 'Catering Assistant' },
  ],
  delivery: [
    { id: 'motorcycle', label: 'Motorcycle Rider' },
    { id: 'food-delivery', label: 'Food Delivery Rider' },
    { id: 'car-driver', label: 'Car Driver' },
    { id: 'bicycle', label: 'Bicycle Courier' },
    { id: 'escooter', label: 'E-Scooter Courier' },
    { id: 'van-driver', label: 'Van Driver' },
    { id: 'express-messenger', label: 'Express Messenger' },
    { id: 'pickup-driver', label: 'Pickup Truck Driver' },
    { id: 'truck-driver', label: 'Heavy Truck Driver' },
    { id: 'document-courier', label: 'Document Courier' },
    { id: 'dispatch-driver', label: 'Warehouse Dispatch Driver' },
    { id: 'last-mile', label: 'Last-Mile Delivery Specialist' },
  ],
  helpers: [
    { id: 'event', label: 'Event Helper' },
    { id: 'banquet', label: 'Banquet Server' },
    { id: 'warehouse', label: 'Warehouse Helper' },
    { id: 'inventory', label: 'Inventory & Stock Assistant' },
    { id: 'loading-crew', label: 'Loading & Unloading Crew' },
    { id: 'office', label: 'Office Helper' },
    { id: 'moving', label: 'Moving Helper' },
    { id: 'cleaning', label: 'Cleaning Crew' },
    { id: 'sanitation', label: 'Sanitation Staff' },
    { id: 'staging', label: 'Staging & Booth Builder' },
    { id: 'general-laborer', label: 'General Laborer' },
    { id: 'retail-assistant', label: 'Retail Store Assistant' },
    { id: 'maintenance', label: 'Maintenance Assistant' },
    { id: 'usher-assistant', label: 'Security / Usher Assistant' },
  ],
};

const CREW_DATA = {
  kitchen: [
    { initials: 'JD', name: 'John D.', role: 'Line Cook', rating: 4.8, distance: '1.2 km', eta: 22, jobs: 147 },
    { initials: 'MS', name: 'Maria S.', role: 'Chef', rating: 4.9, distance: '1.4 km', eta: 25, jobs: 312 },
    { initials: 'RC', name: 'Ryan C.', role: 'Kitchen Helper', rating: 4.7, distance: '2.1 km', eta: 30, jobs: 89 },
    { initials: 'AL', name: 'Angela L.', role: 'Sous Chef', rating: 4.9, distance: '1.6 km', eta: 20, jobs: 240 },
    { initials: 'EB', name: 'Eric B.', role: 'Barista', rating: 4.8, distance: '0.9 km', eta: 15, jobs: 175 },
    { initials: 'PV', name: 'Paula V.', role: 'Pastry Chef', rating: 4.9, distance: '2.4 km', eta: 28, jobs: 198 },
    { initials: 'KT', name: 'Karl T.', role: 'Prep Assistant', rating: 4.7, distance: '1.8 km', eta: 24, jobs: 112 },
  ],
  delivery: [
    { initials: 'AL', name: 'Angelo L.', role: 'Motorcycle Rider', rating: 4.9, distance: '0.8 km', eta: 12, jobs: 523 },
    { initials: 'BT', name: 'Beth T.', role: 'Car Driver', rating: 4.6, distance: '1.5 km', eta: 18, jobs: 204 },
    { initials: 'MR', name: 'Mark R.', role: 'Van Driver', rating: 4.8, distance: '2.0 km', eta: 24, jobs: 178 },
    { initials: 'JN', name: 'Jared N.', role: 'Food Delivery Rider', rating: 4.9, distance: '0.6 km', eta: 10, jobs: 410 },
    { initials: 'SM', name: 'Sam M.', role: 'Express Messenger', rating: 4.8, distance: '1.3 km', eta: 16, jobs: 285 },
    { initials: 'CL', name: 'Chris L.', role: 'E-Scooter Courier', rating: 4.7, distance: '1.1 km', eta: 14, jobs: 160 },
    { initials: 'RH', name: 'Ronald H.', role: 'Heavy Truck Driver', rating: 4.9, distance: '3.2 km', eta: 32, jobs: 215 },
  ],
  helpers: [
    { initials: 'KM', name: 'Kevin M.', role: 'Event Helper', rating: 4.8, distance: '1.0 km', eta: 15, jobs: 96 },
    { initials: 'SC', name: 'Sara C.', role: 'Warehouse Helper', rating: 4.7, distance: '2.3 km', eta: 28, jobs: 141 },
    { initials: 'DP', name: 'Diego P.', role: 'Moving Helper', rating: 4.9, distance: '1.8 km', eta: 20, jobs: 63 },
    { initials: 'TR', name: 'Tina R.', role: 'Banquet Server', rating: 4.9, distance: '1.2 km', eta: 16, jobs: 184 },
    { initials: 'GL', name: 'Gary L.', role: 'Loading & Unloading Crew', rating: 4.8, distance: '2.0 km', eta: 22, jobs: 130 },
    { initials: 'NP', name: 'Nina P.', role: 'Inventory & Stock Assistant', rating: 4.9, distance: '1.7 km', eta: 19, jobs: 155 },
    { initials: 'RM', name: 'Rico M.', role: 'Sanitation Staff', rating: 4.8, distance: '1.4 km', eta: 17, jobs: 110 },
  ],
};

const RATES = {
  kitchen: { amount: '₱95', note: 'Minimum 4 hours' },
  delivery: { amount: '₱75', note: 'Minimum 3 hours' },
  helpers: { amount: '₱65', note: 'Minimum 4 hours' },
};

const CAT_LABELS = {
  kitchen: 'Kitchen',
  delivery: 'Delivery',
  helpers: 'Helpers',
};

const MONTH_NAMES = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December'
];

/* ─── State ───────────────────────────────────────────────── */

// Reference today as September 15, 2026
const TODAY = new Date(2026, 8, 15);
TODAY.setHours(0, 0, 0, 0);

const state = {
  category: null,
  role: null,
  roleLabel: null,
  employmentType: 'full-time',
  offeredRate: '85',
  ratePeriod: 'per hour',
  location: 'Cebu City',
  mapLocation: '',
  googleMapsUrl: '',
  geo: null,
  timing: 'now',
  scheduledDate: new Date(TODAY),
  scheduledTime: '08:00',
  count: 1,
  currentJobId: null,
  currentBooking: null,
  matchCheckInterval: null,
  paid: false,
  paymentMethod: 'GCash',
  paymentAmount: 0,
  pendingPayment: null,
  searchCountdown: 180,
  searchTimerInterval: null,
};

/* ─── DOM refs ────────────────────────────────────────────── */

const segs = document.querySelectorAll('.db-progress-seg');
const steps = document.querySelectorAll('.db-step');
const progress = document.getElementById('dbProgress');
const catCards = document.querySelectorAll('.db-cat-card');
const rolesWrap = document.getElementById('rolesWrap');
const rolesSelect = document.getElementById('rolesSelect');
const rolesLabel = document.getElementById('rolesLabel');
const rolesCountPill = document.getElementById('rolesCountPill');
const employerRoleDropdown = document.getElementById('employerRoleDropdown');
const roleDropdownTrigger = document.getElementById('roleDropdownTrigger');
const roleDropdownMenu = document.getElementById('roleDropdownMenu');
const roleTriggerIcon = document.getElementById('roleTriggerIcon');
const roleTriggerText = document.getElementById('roleTriggerText');
const roleSelectedBadge = document.getElementById('roleSelectedBadge');
const roleSearchInput = document.getElementById('roleSearchInput');
const roleSearchClear = document.getElementById('roleSearchClear');
const roleOptionsList = document.getElementById('roleOptionsList');
const roleEmptyMsg = document.getElementById('roleEmptyMsg');
const toStep2Btn = document.getElementById('toStep2');
const step1Hint = document.getElementById('step1Hint');

// Step 2 refs
const empToggles = document.querySelectorAll('.db-emp-toggle');
const offeredRateInput = document.getElementById('offeredRate');
const ratePeriodSelect = document.getElementById('ratePeriod');
const ratePeriodDropdown = document.getElementById('ratePeriodDropdown');
const ratePeriodTrigger = document.getElementById('ratePeriodTrigger');
const ratePeriodTriggerText = document.getElementById('ratePeriodTriggerText');
const ratePeriodMenu = document.getElementById('ratePeriodMenu');

const workLocationSelect = document.getElementById('workLocation');
const locationDropdown = document.getElementById('locationDropdown');
const locationDropdownTrigger = document.getElementById('locationDropdownTrigger');
const locationTriggerText = document.getElementById('locationTriggerText');
const locationDropdownMenu = document.getElementById('locationDropdownMenu');
const locationSearchInput = document.getElementById('locationSearchInput');
const locationOptionsList = document.getElementById('locationOptionsList');
const locationEmptyMsg = document.getElementById('locationEmptyMsg');

const useCurrentBtn = document.getElementById('useCurrentBtn');
const googleMapLocationInput = document.getElementById('googleMapLocationInput');
const previewGoogleMapBtn = document.getElementById('previewGoogleMapBtn');
const clearMapLocationBtn = document.getElementById('clearMapLocationBtn');
const dMapLocation = document.getElementById('dMapLocation');
const toStep3Btn = document.getElementById('toStep3');

// Step 3 refs
const countVal = document.getElementById('countVal');
const laterField = document.getElementById('laterField');
const toStep4Btn = document.getElementById('toStep4');
const openScheduleModalBtn = document.getElementById('openScheduleModalBtn');
const scheduleDisplayDate = document.getElementById('scheduleDisplayDate');
const scheduleDisplayTime = document.getElementById('scheduleDisplayTime');

// Step 4 refs
const crewList = document.getElementById('crewList');
const matchLoading = document.getElementById('matchLoading');
const matchResults = document.getElementById('matchResults');
const rateAmount = document.getElementById('rateAmount');
const confirmBooking = document.getElementById('confirmBooking');
const simulateAcceptBtn = document.getElementById('simulateAcceptBtn');
const resultsHeading = document.getElementById('resultsHeading');
const resultsAcceptedNote = document.getElementById('resultsAcceptedNote');

// Step 5 refs
const detailsToggle = document.getElementById('detailsToggle');
const detailsBox = document.getElementById('detailsBox');
const bookAnother = document.getElementById('bookAnother');

// Scheduler Modal refs
const scheduleModal = document.getElementById('scheduleModal');
const calMonthLabel = document.getElementById('calMonthLabel');
const calPrevBtn = document.getElementById('calPrevBtn');
const calNextBtn = document.getElementById('calNextBtn');
const calDaysGrid = document.getElementById('calDaysGrid');
const calTimeChips = document.querySelectorAll('.db-time-chip');
const calTimePreview = document.getElementById('calTimePreview');
const customTimeWrap = document.getElementById('customTimeWrap');
const customTimeInput = document.getElementById('customTimeInput');
const empCustomHourDisplay = document.getElementById('empCustomHourDisplay');
const empCustomMinuteDisplay = document.getElementById('empCustomMinuteDisplay');
const hourUpBtn = document.getElementById('hourUpBtn');
const hourDownBtn = document.getElementById('hourDownBtn');
const minuteUpBtn = document.getElementById('minuteUpBtn');
const minuteDownBtn = document.getElementById('minuteDownBtn');
const empBtnAM = document.getElementById('empBtnAM');
const empBtnPM = document.getElementById('empBtnPM');
let empCustomAmPm = 'AM';
const calSummaryText = document.getElementById('calSummaryText');
const calConfirmBtn = document.getElementById('calConfirmBtn');
const calCancelBtn = document.getElementById('calCancelBtn');

// Payment Modal refs
const paymentModal = document.getElementById('paymentModal');
const paymentCloseXBtn = document.getElementById('paymentCloseXBtn');
const payCancelBtn = document.getElementById('payCancelBtn');
const payNowBtn = document.getElementById('payNowBtn');
const payNowBtnText = document.getElementById('payNowBtnText');
const paySummaryRole = document.getElementById('paySummaryRole');
const paySummaryCat = document.getElementById('paySummaryCat');
const paySummaryTime = document.getElementById('paySummaryTime');
const paySummaryCount = document.getElementById('paySummaryCount');
const paySummaryLocation = document.getElementById('paySummaryLocation');
const payBreakdownMath = document.getElementById('payBreakdownMath');
const payBreakdownSubtotal = document.getElementById('payBreakdownSubtotal');
const payBreakdownFee = document.getElementById('payBreakdownFee');
const payBreakdownTotal = document.getElementById('payBreakdownTotal');
const matchEscrowPill = document.getElementById('matchEscrowPill');
const matchEscrowText = document.getElementById('matchEscrowText');
const resultsPaidBadge = document.getElementById('resultsPaidBadge');
const dPayment = document.getElementById('dPayment');

// Step 4 Matching refs
const dbSearchTimerBox = document.getElementById('dbSearchTimerBox');
const dbTryTimer = document.getElementById('dbTryTimer');
const dbSearchProgressFill = document.getElementById('dbSearchProgressFill');
const simulateTimeoutBtn = document.getElementById('simulateTimeoutBtn');
const matchFailed = document.getElementById('matchFailed');
const repayMatchFeeBtn = document.getElementById('repayMatchFeeBtn');
const editShiftFromFailedBtn = document.getElementById('editShiftFromFailedBtn');

/* ─── Scheduler Modal State & Logic ───────────────────────── */

let calViewYear = 2026;
let calViewMonth = 8; // September (0-indexed)
let tempSelectedDate = new Date(state.scheduledDate);
let tempSelectedTime = state.scheduledTime;

function format12Hour(timeStr) {
  if (!timeStr) return '8:00 AM';
  const [h, m] = timeStr.split(':').map(Number);
  const period = h >= 12 ? 'PM' : 'AM';
  const hour12 = h % 12 || 12;
  return `${hour12}:${String(m).padStart(2, '0')} ${period}`;
}

function formatDateShort(d) {
  const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
  return `${days[d.getDay()]}, ${months[d.getMonth()]} ${d.getDate()}, ${d.getFullYear()}`;
}

function formatDateFull(d) {
  const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
  return `${days[d.getDay()]}, ${MONTH_NAMES[d.getMonth()]} ${d.getDate()}, ${d.getFullYear()}`;
}

function updateModalSummary() {
  const dateStr = formatDateShort(tempSelectedDate);
  const timeStr = format12Hour(tempSelectedTime);
  calSummaryText.textContent = `Scheduled for ${dateStr} at ${timeStr}`;
  calTimePreview.textContent = timeStr;
}

function renderCalendar() {
  calMonthLabel.textContent = `${MONTH_NAMES[calViewMonth]} ${calViewYear}`;
  calDaysGrid.innerHTML = '';

  const firstDayIndex = new Date(calViewYear, calViewMonth, 1).getDay();
  const daysInMonth = new Date(calViewYear, calViewMonth + 1, 0).getDate();

  // Empty padding slots
  for (let i = 0; i < firstDayIndex; i++) {
    const emptyCell = document.createElement('div');
    emptyCell.className = 'db-cal-day empty';
    emptyCell.setAttribute('aria-hidden', 'true');
    calDaysGrid.appendChild(emptyCell);
  }

  // Days 1 through daysInMonth
  for (let d = 1; d <= daysInMonth; d++) {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'db-cal-day';
    btn.textContent = d;

    const cellDate = new Date(calViewYear, calViewMonth, d);
    cellDate.setHours(0, 0, 0, 0);

    const isPast = cellDate < TODAY;
    const isToday = cellDate.getTime() === TODAY.getTime();
    const isSelected = cellDate.getTime() === tempSelectedDate.getTime();

    if (isPast) {
      btn.classList.add('disabled');
      btn.setAttribute('aria-disabled', 'true');
      btn.disabled = true;
    } else {
      if (isToday) btn.classList.add('today');
      if (isSelected) btn.classList.add('selected');

      btn.addEventListener('click', () => {
        tempSelectedDate = new Date(cellDate);
        renderCalendar();
        updateModalSummary();
      });
    }

    calDaysGrid.appendChild(btn);
  }
}


function openScheduleModal() {
  tempSelectedDate = new Date(state.scheduledDate);
  tempSelectedTime = state.scheduledTime;
  calViewYear = tempSelectedDate.getFullYear();
  calViewMonth = tempSelectedDate.getMonth();

  renderCalendar();
  updateModalSummary();

  // Set active time chip
  let matchedChip = false;
  calTimeChips.forEach(chip => {
    if (chip.dataset.time === tempSelectedTime) {
      chip.classList.add('active');
      matchedChip = true;
    } else {
      chip.classList.remove('active');
    }
  });

  if (!matchedChip) {
    document.getElementById('customTimeChip').classList.add('active');
    customTimeWrap.style.display = 'block';
    syncEmpCustomSelects(tempSelectedTime);
  } else {
    customTimeWrap.style.display = 'none';
  }

  scheduleModal.classList.add('open');
  scheduleModal.setAttribute('aria-hidden', 'false');
  document.body.style.overflow = 'hidden';
}

function closeScheduleModal() {
  scheduleModal.classList.remove('open');
  scheduleModal.setAttribute('aria-hidden', 'true');
  document.body.style.overflow = '';
}

// Navigation buttons
calPrevBtn.addEventListener('click', () => {
  // Prevent navigating before September 2026
  if (calViewYear === 2026 && calViewMonth <= 8) return;
  calViewMonth--;
  if (calViewMonth < 0) {
    calViewMonth = 11;
    calViewYear--;
  }
  renderCalendar();
});

calNextBtn.addEventListener('click', () => {
  calViewMonth++;
  if (calViewMonth > 11) {
    calViewMonth = 0;
    calViewYear++;
  }
  renderCalendar();
});


function syncEmpCustomSelects(timeStr) {
  const [hRaw, mRaw] = (timeStr || '08:00').split(':').map(Number);
  empCustomAmPm = (hRaw >= 12 && !isNaN(hRaw)) ? 'PM' : 'AM';
  let h12 = (hRaw || 8) % 12;
  if (h12 === 0) h12 = 12;
  const mNum = isNaN(mRaw) ? 0 : mRaw;
  const mStr = String(mNum).padStart(2, '0');

  if (empCustomHourDisplay) empCustomHourDisplay.value = String(h12).padStart(2, '0');
  if (empCustomMinuteDisplay) empCustomMinuteDisplay.value = mStr;

  if (empBtnAM && empBtnPM) {
    empBtnAM.classList.toggle('active', empCustomAmPm === 'AM');
    empBtnAM.setAttribute('aria-checked', empCustomAmPm === 'AM');
    empBtnPM.classList.toggle('active', empCustomAmPm === 'PM');
    empBtnPM.setAttribute('aria-checked', empCustomAmPm === 'PM');
  }
}

function updateEmpCustomTime() {
  let h = parseInt(empCustomHourDisplay ? empCustomHourDisplay.value : '8', 10);
  if (isNaN(h) || h < 1) h = 12;
  if (h > 12) h = 12;

  let m = parseInt(empCustomMinuteDisplay ? empCustomMinuteDisplay.value : '0', 10);
  if (isNaN(m) || m < 0) m = 0;
  if (m > 59) m = 59;
  const mStr = String(m).padStart(2, '0');

  if (empCustomAmPm === 'PM' && h < 12) h += 12;
  if (empCustomAmPm === 'AM' && h === 12) h = 0;

  tempSelectedTime = `${String(h).padStart(2, '0')}:${mStr}`;
  if (customTimeInput) customTimeInput.value = tempSelectedTime;
  updateModalSummary();
}

// Hour steppers
if (hourUpBtn) {
  hourUpBtn.addEventListener('click', () => {
    let h = parseInt(empCustomHourDisplay ? empCustomHourDisplay.value : '8', 10) || 8;
    h = (h % 12) + 1;
    if (empCustomHourDisplay) empCustomHourDisplay.value = String(h).padStart(2, '0');
    updateEmpCustomTime();
  });
}

if (hourDownBtn) {
  hourDownBtn.addEventListener('click', () => {
    let h = parseInt(empCustomHourDisplay ? empCustomHourDisplay.value : '8', 10) || 8;
    h = (h === 1) ? 12 : h - 1;
    if (empCustomHourDisplay) empCustomHourDisplay.value = String(h).padStart(2, '0');
    updateEmpCustomTime();
  });
}

// Minute steppers (15-min intervals)
if (minuteUpBtn) {
  minuteUpBtn.addEventListener('click', () => {
    let m = parseInt(empCustomMinuteDisplay ? empCustomMinuteDisplay.value : '0', 10) || 0;
    m = (Math.floor(m / 15) * 15 + 15) % 60;
    if (empCustomMinuteDisplay) empCustomMinuteDisplay.value = String(m).padStart(2, '0');
    updateEmpCustomTime();
  });
}

if (minuteDownBtn) {
  minuteDownBtn.addEventListener('click', () => {
    let m = parseInt(empCustomMinuteDisplay ? empCustomMinuteDisplay.value : '0', 10) || 0;
    m = (Math.ceil(m / 15) * 15 - 15 + 60) % 60;
    if (empCustomMinuteDisplay) empCustomMinuteDisplay.value = String(m).padStart(2, '0');
    updateEmpCustomTime();
  });
}

// Direct typing & mousewheel on Hour
if (empCustomHourDisplay) {
  empCustomHourDisplay.addEventListener('focus', () => empCustomHourDisplay.select());
  empCustomHourDisplay.addEventListener('blur', () => {
    let h = parseInt(empCustomHourDisplay.value, 10);
    if (isNaN(h) || h < 1) h = 12;
    if (h > 12) h = 12;
    empCustomHourDisplay.value = String(h).padStart(2, '0');
    updateEmpCustomTime();
  });
  empCustomHourDisplay.addEventListener('keydown', (e) => {
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (hourUpBtn) hourUpBtn.click();
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (hourDownBtn) hourDownBtn.click();
    } else if (e.key === 'Enter') {
      empCustomHourDisplay.blur();
    }
  });
  empCustomHourDisplay.addEventListener('wheel', (e) => {
    e.preventDefault();
    if (e.deltaY < 0) {
      if (hourUpBtn) hourUpBtn.click();
    } else {
      if (hourDownBtn) hourDownBtn.click();
    }
  }, { passive: false });
}

// Direct typing & mousewheel on Minute
if (empCustomMinuteDisplay) {
  empCustomMinuteDisplay.addEventListener('focus', () => empCustomMinuteDisplay.select());
  empCustomMinuteDisplay.addEventListener('blur', () => {
    let m = parseInt(empCustomMinuteDisplay.value, 10);
    if (isNaN(m) || m < 0) m = 0;
    if (m > 59) m = 59;
    empCustomMinuteDisplay.value = String(m).padStart(2, '0');
    updateEmpCustomTime();
  });
  empCustomMinuteDisplay.addEventListener('keydown', (e) => {
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (minuteUpBtn) minuteUpBtn.click();
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (minuteDownBtn) minuteDownBtn.click();
    } else if (e.key === 'Enter') {
      empCustomMinuteDisplay.blur();
    }
  });
  empCustomMinuteDisplay.addEventListener('wheel', (e) => {
    e.preventDefault();
    if (e.deltaY < 0) {
      if (minuteUpBtn) minuteUpBtn.click();
    } else {
      if (minuteDownBtn) minuteDownBtn.click();
    }
  }, { passive: false });
}

// AM / PM Segmented Control
if (empBtnAM) {
  empBtnAM.addEventListener('click', () => {
    empCustomAmPm = 'AM';
    empBtnAM.classList.add('active');
    empBtnAM.setAttribute('aria-checked', 'true');
    if (empBtnPM) {
      empBtnPM.classList.remove('active');
      empBtnPM.setAttribute('aria-checked', 'false');
    }
    updateEmpCustomTime();
  });
}

if (empBtnPM) {
  empBtnPM.addEventListener('click', () => {
    empCustomAmPm = 'PM';
    empBtnPM.classList.add('active');
    empBtnPM.setAttribute('aria-checked', 'true');
    if (empBtnAM) {
      empBtnAM.classList.remove('active');
      empBtnAM.setAttribute('aria-checked', 'false');
    }
    updateEmpCustomTime();
  });
}

// Time chips
calTimeChips.forEach(chip => {
  chip.addEventListener('click', () => {
    calTimeChips.forEach(c => c.classList.remove('active'));
    chip.classList.add('active');

    const val = chip.dataset.time;
    if (val === 'custom') {
      customTimeWrap.style.display = 'block';
      syncEmpCustomSelects(tempSelectedTime);
      updateEmpCustomTime();
    } else {
      customTimeWrap.style.display = 'none';
      tempSelectedTime = val;
      if (customTimeInput) customTimeInput.value = val;
      updateModalSummary();
    }
  });
});

// Modal Actions
calConfirmBtn.addEventListener('click', () => {
  state.scheduledDate = new Date(tempSelectedDate);
  state.scheduledTime = tempSelectedTime;

  // Update Step 3 Schedule Card display
  scheduleDisplayDate.textContent = formatDateShort(state.scheduledDate);
  scheduleDisplayTime.textContent = `${format12Hour(state.scheduledTime)} · Scheduled Shift`;

  closeScheduleModal();
});

calCancelBtn.addEventListener('click', closeScheduleModal);

// Close on backdrop click
scheduleModal.addEventListener('click', (e) => {
  if (e.target === scheduleModal) {
    closeScheduleModal();
  }
});

// Close on Escape key
window.addEventListener('keydown', (e) => {
  if (e.key === 'Escape' && scheduleModal.classList.contains('open')) {
    closeScheduleModal();
  }
});

// Open modal from card click
if (openScheduleModalBtn) {
  openScheduleModalBtn.addEventListener('click', openScheduleModal);
  openScheduleModalBtn.addEventListener('keydown', (e) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      openScheduleModal();
    }
  });
}

/* ─── Validation Helpers ───────────────────────────────────── */

function validateGoogleMapLocation() {
  const mapVal = (googleMapLocationInput && googleMapLocationInput.value.trim()) || '';
  if (!mapVal) {
    if (googleMapLocationInput) {
      googleMapLocationInput.classList.remove('error');
      void googleMapLocationInput.offsetWidth; // re-trigger shake animation
      googleMapLocationInput.classList.add('error');
      googleMapLocationInput.focus();
      googleMapLocationInput.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
    const errorMsg = document.getElementById('googleMapErrorMsg');
    if (errorMsg) errorMsg.classList.add('visible');
    return false;
  }

  if (googleMapLocationInput) googleMapLocationInput.classList.remove('error');
  const errorMsg = document.getElementById('googleMapErrorMsg');
  if (errorMsg) errorMsg.classList.remove('visible');
  return true;
}

/* ─── Step navigation ─────────────────────────────────────── */

function goStep(n) {
  // If currently on Step 2 and attempting to proceed to Step 3 or higher, require Google Map location
  const currentActiveStep = document.querySelector('.db-step.active');
  const currentStepNum = currentActiveStep && currentActiveStep.id ? parseInt(currentActiveStep.id.replace('step', ''), 10) : 1;
  if (currentStepNum === 2 && n > 2) {
    if (!validateGoogleMapLocation()) {
      return;
    }
  }

  steps.forEach(s => s.classList.remove('active'));

  const target = document.getElementById('step' + n);
  if (!target) return;
  target.classList.add('active');

  // Progress bar (segments 1 to 4)
  segs.forEach(s => s.classList.toggle('active', Number(s.dataset.seg) <= n));

  // Hide progress bar on Step 5 (success screen)
  progress.style.display = (n >= 5) ? 'none' : 'flex';

  // Scroll card to top smoothly
  target.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
}

/* ─── Step 1: Category & Role selection ───────────────────── */

catCards.forEach(card => {
  card.addEventListener('click', () => {
    const cat = card.dataset.cat;

    // Deselect all
    catCards.forEach(c => c.setAttribute('aria-pressed', 'false'));

    // Select this one
    card.setAttribute('aria-pressed', 'true');
    state.category = cat;
    state.role = null;
    state.roleLabel = null;

    // Render role chips
    renderRoles(cat);

    // Show role panel
    rolesWrap.classList.add('open');
    rolesLabel.textContent = `Choose a ${CAT_LABELS[cat]} role`;

    // Disable Next until role is picked
    toStep2Btn.disabled = true;
    step1Hint.textContent = 'Now pick a specific role.';
  });
});

function renderRoles(cat) {
  const roles = ROLES[cat] || [];

  // 1. Sync hidden native select for form & script compatibility
  if (rolesSelect) {
    rolesSelect.innerHTML = '<option value="" disabled selected>Select a role…</option>';
    roles.forEach(role => {
      const opt = document.createElement('option');
      opt.value = role.id;
      opt.textContent = role.label;
      rolesSelect.appendChild(opt);
    });
  }

  // 2. Update category icon in trigger
  if (roleTriggerIcon && window.InstantCrewShared && window.InstantCrewShared.ICONS) {
    if (cat === 'kitchen') {
      roleTriggerIcon.innerHTML = window.InstantCrewShared.ICONS.kitchen(18);
    } else if (cat === 'delivery') {
      roleTriggerIcon.innerHTML = window.InstantCrewShared.ICONS.delivery(18);
    } else if (cat === 'helpers') {
      roleTriggerIcon.innerHTML = window.InstantCrewShared.ICONS.helpers(18);
    } else {
      roleTriggerIcon.innerHTML = window.InstantCrewShared.ICONS.briefcase(18);
    }
  }

  // 3. Update count pill
  if (rolesCountPill) {
    rolesCountPill.textContent = `${roles.length} roles available`;
    rolesCountPill.style.display = 'inline-block';
  }

  // 4. Reset trigger state
  if (roleTriggerText) {
    roleTriggerText.textContent = `Select a ${CAT_LABELS[cat] || ''} role…`;
    roleTriggerText.classList.add('placeholder');
  }
  if (roleSelectedBadge) {
    roleSelectedBadge.style.display = 'none';
  }

  // 5. Close dropdown if open & reset search
  closeRoleDropdown();
  if (roleSearchInput) {
    roleSearchInput.value = '';
  }
  if (roleSearchClear) {
    roleSearchClear.style.display = 'none';
  }

  // 6. Populate custom options list
  renderRoleOptionsList(roles);
}

function renderRoleOptionsList(roles, filterQuery = '') {
  if (!roleOptionsList) return;
  roleOptionsList.innerHTML = '';

  const query = filterQuery.trim().toLowerCase();
  const filtered = query
    ? roles.filter(r => r.label.toLowerCase().includes(query))
    : roles;

  if (filtered.length === 0) {
    if (roleEmptyMsg) roleEmptyMsg.style.display = 'block';
  } else {
    if (roleEmptyMsg) roleEmptyMsg.style.display = 'none';
  }

  filtered.forEach(role => {
    const isSelected = state.role === role.id;
    const item = document.createElement('div');
    item.className = `db-custom-option-item ${isSelected ? 'selected' : ''}`;
    item.setAttribute('role', 'option');
    item.setAttribute('aria-selected', String(isSelected));
    item.dataset.roleId = role.id;

    item.innerHTML = `
            <div class="db-custom-option-left">
                <span class="db-custom-option-dot"></span>
                <span class="db-custom-option-label">${role.label}</span>
            </div>
            <span class="db-custom-option-check">
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="20 6 9 17 4 12"></polyline>
                </svg>
            </span>
        `;

    item.addEventListener('click', (e) => {
      e.stopPropagation();
      selectRole(role);
    });

    roleOptionsList.appendChild(item);
  });
}

function selectRole(role) {
  state.role = role.id;
  state.roleLabel = role.label;

  if (rolesSelect) {
    rolesSelect.value = role.id;
  }

  if (roleTriggerText) {
    roleTriggerText.textContent = role.label;
    roleTriggerText.classList.remove('placeholder');
  }

  if (roleSelectedBadge) {
    roleSelectedBadge.style.display = 'inline-flex';
  }

  // Highlight selected item in options list
  if (roleOptionsList) {
    roleOptionsList.querySelectorAll('.db-custom-option-item').forEach(el => {
      const active = el.dataset.roleId === role.id;
      el.classList.toggle('selected', active);
      el.setAttribute('aria-selected', String(active));
    });
  }

  closeRoleDropdown();

  toStep2Btn.disabled = false;
  step1Hint.textContent = 'Ready — tap Next to continue.';
}

function toggleRoleDropdown() {
  if (!employerRoleDropdown) return;
  const isOpen = employerRoleDropdown.classList.contains('open');
  if (isOpen) {
    closeRoleDropdown();
  } else {
    openRoleDropdown();
  }
}

function openRoleDropdown() {
  if (!employerRoleDropdown || !roleDropdownMenu) return;
  employerRoleDropdown.classList.add('open');
  roleDropdownMenu.hidden = false;
  if (roleDropdownTrigger) roleDropdownTrigger.setAttribute('aria-expanded', 'true');
  if (roleSearchInput) {
    setTimeout(() => roleSearchInput.focus(), 60);
  }
}

function closeRoleDropdown() {
  if (!employerRoleDropdown || !roleDropdownMenu) return;
  employerRoleDropdown.classList.remove('open');
  roleDropdownMenu.hidden = true;
  if (roleDropdownTrigger) roleDropdownTrigger.setAttribute('aria-expanded', 'false');
}

// Trigger click
if (roleDropdownTrigger) {
  roleDropdownTrigger.addEventListener('click', (e) => {
    e.stopPropagation();
    toggleRoleDropdown();
  });
}

// Search input handling
if (roleSearchInput) {
  roleSearchInput.addEventListener('input', (e) => {
    const query = e.target.value;
    if (roleSearchClear) {
      roleSearchClear.style.display = query ? 'flex' : 'none';
    }
    const roles = ROLES[state.category] || [];
    renderRoleOptionsList(roles, query);
  });

  roleSearchInput.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      closeRoleDropdown();
      if (roleDropdownTrigger) roleDropdownTrigger.focus();
    }
  });
}

if (roleSearchClear) {
  roleSearchClear.addEventListener('click', (e) => {
    e.stopPropagation();
    roleSearchInput.value = '';
    roleSearchClear.style.display = 'none';
    roleSearchInput.focus();
    const roles = ROLES[state.category] || [];
    renderRoleOptionsList(roles, '');
  });
}

// Close when clicking outside
document.addEventListener('click', (e) => {
  if (employerRoleDropdown && !employerRoleDropdown.contains(e.target)) {
    closeRoleDropdown();
  }
  if (ratePeriodDropdown && !ratePeriodDropdown.contains(e.target)) {
    closeRatePeriodDropdown();
  }
  if (locationDropdown && !locationDropdown.contains(e.target)) {
    closeLocationDropdown();
  }
});

// Escape key to close
window.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') {
    if (employerRoleDropdown && employerRoleDropdown.classList.contains('open')) {
      closeRoleDropdown();
      if (roleDropdownTrigger) roleDropdownTrigger.focus();
    }
    if (ratePeriodDropdown && ratePeriodDropdown.classList.contains('open')) {
      closeRatePeriodDropdown();
      if (ratePeriodTrigger) ratePeriodTrigger.focus();
    }
    if (locationDropdown && locationDropdown.classList.contains('open')) {
      closeLocationDropdown();
      if (locationDropdownTrigger) locationDropdownTrigger.focus();
    }
  }
});

// Backward-compatibility: if native select changes, sync to custom UI
if (rolesSelect) {
  rolesSelect.addEventListener('change', () => {
    const roleObj = (ROLES[state.category] || []).find(r => r.id === rolesSelect.value);
    if (roleObj) {
      selectRole(roleObj);
    }
  });
}

toStep2Btn.addEventListener('click', () => goStep(2));

/* ─── Step 2: Employment Type, Rate & Location ────────────── */

empToggles.forEach(toggle => {
  toggle.addEventListener('click', () => {
    empToggles.forEach(t => t.classList.remove('active'));
    toggle.classList.add('active');
    state.employmentType = toggle.dataset.emp;
  });
});

offeredRateInput.addEventListener('input', (e) => {
  state.offeredRate = e.target.value;
});

/* ─── Custom Rate Period Dropdown Logic ─── */
function closeRatePeriodDropdown() {
  if (!ratePeriodDropdown || !ratePeriodMenu) return;
  ratePeriodDropdown.classList.remove('open');
  ratePeriodMenu.hidden = true;
  if (ratePeriodTrigger) ratePeriodTrigger.setAttribute('aria-expanded', 'false');
}

function openRatePeriodDropdown() {
  if (!ratePeriodDropdown || !ratePeriodMenu) return;
  closeLocationDropdown();
  closeRoleDropdown();
  ratePeriodDropdown.classList.add('open');
  ratePeriodMenu.hidden = false;
  if (ratePeriodTrigger) ratePeriodTrigger.setAttribute('aria-expanded', 'true');
}

function selectRatePeriod(val) {
  if (ratePeriodSelect) {
    ratePeriodSelect.value = val;
  }
  state.ratePeriod = val;
  if (ratePeriodTriggerText) {
    ratePeriodTriggerText.textContent = val;
  }
  if (ratePeriodMenu) {
    ratePeriodMenu.querySelectorAll('.db-select-custom-item').forEach(item => {
      const isMatch = item.dataset.value === val;
      item.classList.toggle('selected', isMatch);
      item.setAttribute('aria-selected', String(isMatch));
    });
  }
  closeRatePeriodDropdown();
}

if (ratePeriodTrigger) {
  ratePeriodTrigger.addEventListener('click', (e) => {
    e.stopPropagation();
    if (ratePeriodDropdown && ratePeriodDropdown.classList.contains('open')) {
      closeRatePeriodDropdown();
    } else {
      openRatePeriodDropdown();
    }
  });
}

if (ratePeriodMenu) {
  ratePeriodMenu.querySelectorAll('.db-select-custom-item').forEach(item => {
    item.addEventListener('click', (e) => {
      e.stopPropagation();
      const val = item.dataset.value;
      selectRatePeriod(val);
    });
  });
}

ratePeriodSelect.addEventListener('change', (e) => {
  selectRatePeriod(e.target.value);
});

/* ─── Custom Work Location Dropdown Logic ─── */
function closeLocationDropdown() {
  if (!locationDropdown || !locationDropdownMenu) return;
  locationDropdown.classList.remove('open');
  locationDropdownMenu.hidden = true;
  if (locationDropdownTrigger) locationDropdownTrigger.setAttribute('aria-expanded', 'false');
}

function openLocationDropdown() {
  if (!locationDropdown || !locationDropdownMenu) return;
  closeRatePeriodDropdown();
  closeRoleDropdown();
  locationDropdown.classList.add('open');
  locationDropdownMenu.hidden = false;
  if (locationDropdownTrigger) locationDropdownTrigger.setAttribute('aria-expanded', 'true');
  if (locationSearchInput) {
    locationSearchInput.value = '';
    renderLocationOptions('');
    setTimeout(() => locationSearchInput.focus(), 50);
  }
}

function selectLocation(val) {
  if (workLocationSelect) {
    workLocationSelect.value = val;
  }
  state.location = val;
  if (locationTriggerText) {
    locationTriggerText.textContent = val;
  }
  renderLocationOptions('');
  closeLocationDropdown();
}

function renderLocationOptions(query = '') {
  if (!locationOptionsList || !workLocationSelect) return;
  locationOptionsList.innerHTML = '';

  const q = query.trim().toLowerCase();
  const opts = Array.from(workLocationSelect.options);
  let matchCount = 0;

  opts.forEach(opt => {
    const text = opt.textContent;
    const val = opt.value;
    const matches = !q || text.toLowerCase().includes(q);

    if (matches) {
      matchCount++;
      const item = document.createElement('div');
      item.className = `db-select-custom-item${workLocationSelect.value === val ? ' selected' : ''}`;
      item.setAttribute('role', 'option');
      item.setAttribute('aria-selected', String(workLocationSelect.value === val));
      item.dataset.value = val;

      item.innerHTML = `
                <span class="db-select-item-dot"></span>
                <span class="db-select-item-label">${text}</span>
                <svg class="db-select-item-check" width="14" height="14" viewBox="0 0 24 24" fill="none"
                    stroke="currentColor" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="20 6 9 17 4 12"></polyline>
                </svg>
            `;

      item.addEventListener('click', (e) => {
        e.stopPropagation();
        selectLocation(val);
      });

      locationOptionsList.appendChild(item);
    }
  });

  if (locationEmptyMsg) {
    locationEmptyMsg.style.display = matchCount === 0 ? 'block' : 'none';
  }
}

if (locationDropdownTrigger) {
  locationDropdownTrigger.addEventListener('click', (e) => {
    e.stopPropagation();
    if (locationDropdown && locationDropdown.classList.contains('open')) {
      closeLocationDropdown();
    } else {
      openLocationDropdown();
    }
  });
}

if (locationSearchInput) {
  locationSearchInput.addEventListener('input', (e) => {
    renderLocationOptions(e.target.value);
  });
  locationSearchInput.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      closeLocationDropdown();
      if (locationDropdownTrigger) locationDropdownTrigger.focus();
    }
  });
}

// Initial render of locations
renderLocationOptions('');

workLocationSelect.addEventListener('change', (e) => {
  selectLocation(e.target.value);
});

/* ── Use Current Location (Geolocator) ── */

function notifyLocationError(err) {
  const msg = (err && err.message) || 'Could not get your location.';
  if (window.InstantCrewShared && InstantCrewShared.showPushToast) {
    InstantCrewShared.showPushToast('Location unavailable', msg, 'end');
  } else {
    alert(msg);
  }
}

/* ── Use Current Location (Geolocator) ── */

const USE_CURRENT_DEFAULT_HTML = useCurrentBtn.innerHTML;

const USE_CURRENT_SPINNER_HTML = `
  <svg width="13" height="13" viewBox="0 0 16 16" fill="none" class="db-spinner">
    <circle cx="8" cy="8" r="6" stroke="currentColor" stroke-width="2" stroke-dasharray="28" stroke-dashoffset="14"/>
  </svg> Locating…`;

const USE_CURRENT_OK_HTML = `
  <svg width="13" height="13" viewBox="0 0 16 16" fill="none">
    <path d="M3 8.5l3.5 3.5L13 5" stroke="#1C1C1E" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
  </svg> Located`;

function notifyLocationError(err) {
  const msg = (err && err.message) || 'Could not get your location.';
  if (window.InstantCrewShared && InstantCrewShared.showPushToast) {
    InstantCrewShared.showPushToast('Location unavailable', msg, 'end');
  } else {
    alert(msg);
  }
}

function buildMapsUrlFor(query) {
  return window.InstantCrewShared
    ? window.InstantCrewShared.formatGoogleMapsUrl(query)
    : `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(query)}`;
}

function applyDetectedLocation(geo) {
  // 1. City dropdown: match an existing option, otherwise add one
  if (geo.city) {
    const norm = s => s.toLowerCase().trim();
    let match = Array.from(workLocationSelect.options).find(o =>
      norm(o.value) === norm(geo.city) ||
      norm(geo.city).includes(norm(o.value)) ||
      norm(o.value).includes(norm(geo.city))
    );
    if (!match) {
      match = document.createElement('option');
      match.value = geo.city;
      match.textContent = geo.city;
      workLocationSelect.insertBefore(match, workLocationSelect.firstChild);
    }
    selectLocation(match.value);
  }

  // 2. Google Map field (readable address; exact pin kept in state.geo)
  if (googleMapLocationInput) {
    googleMapLocationInput.value = geo.label;
    googleMapLocationInput.classList.remove('error');
  }
  state.mapLocation = geo.label;
  state.googleMapsUrl = geo.mapsUrl;
  state.geo = { lat: geo.lat, lng: geo.lng, label: geo.label, mapsUrl: geo.mapsUrl };

  if (clearMapLocationBtn) clearMapLocationBtn.style.display = 'block';
  const errorMsg = document.getElementById('googleMapErrorMsg');
  if (errorMsg) errorMsg.classList.remove('visible');
}

useCurrentBtn.addEventListener('click', async () => {
  if (useCurrentBtn.classList.contains('locating')) return;

  if (!window.InstantCrewLocation) {
    notifyLocationError({ message: 'Location service failed to load. Please enter the address manually.' });
    return;
  }

  useCurrentBtn.classList.add('locating');
  useCurrentBtn.disabled = true;
  useCurrentBtn.innerHTML = USE_CURRENT_SPINNER_HTML;

  try {
    const geo = await window.InstantCrewLocation.locate();
    applyDetectedLocation(geo);
    useCurrentBtn.innerHTML = USE_CURRENT_OK_HTML;
  } catch (err) {
    notifyLocationError(err);
    useCurrentBtn.innerHTML = USE_CURRENT_DEFAULT_HTML;
  } finally {
    useCurrentBtn.classList.remove('locating');
    useCurrentBtn.disabled = false;
    setTimeout(() => { useCurrentBtn.innerHTML = USE_CURRENT_DEFAULT_HTML; }, 2500);
  }
});

/* ── Google Map Location Listeners ── */
if (googleMapLocationInput) {
  googleMapLocationInput.addEventListener('input', (e) => {
    const val = e.target.value.trim();
    state.mapLocation = val;
    state.geo = null; // user edited manually, so drop the GPS pin
    state.googleMapsUrl = buildMapsUrlFor(val || state.location || 'Cebu City');
    if (clearMapLocationBtn) {
      clearMapLocationBtn.style.display = val ? 'block' : 'none';
    }
    if (val) {
      googleMapLocationInput.classList.remove('error');
      const errorMsg = document.getElementById('googleMapErrorMsg');
      if (errorMsg) errorMsg.classList.remove('visible');
    }
  });

  googleMapLocationInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      if (toStep3Btn) toStep3Btn.click();
    }
  });
}

if (clearMapLocationBtn) {
  clearMapLocationBtn.addEventListener('click', () => {
    if (googleMapLocationInput) {
      googleMapLocationInput.value = '';
      googleMapLocationInput.focus();
    }
    state.mapLocation = '';
    state.googleMapsUrl = '';
    state.geo = null;
    clearMapLocationBtn.style.display = 'none';
  });
}

if (previewGoogleMapBtn) {
  previewGoogleMapBtn.addEventListener('click', () => {
    const query = (googleMapLocationInput && googleMapLocationInput.value.trim()) || state.location || 'Cebu City';
    const url = (state.geo && query === state.geo.label)
      ? state.geo.mapsUrl
      : buildMapsUrlFor(query);
    window.open(url, '_blank');
  });
}

if (toStep3Btn) {
  toStep3Btn.addEventListener('click', () => {
    if (!validateGoogleMapLocation()) return;
    goStep(3);
  });
}

/* ─── Step 3: Timing & Headcount ──────────────────────────── */

document.querySelectorAll('.db-toggle').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.db-toggle').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    state.timing = btn.dataset.timing;

    const isLater = state.timing === 'later';
    laterField.classList.toggle('open', isLater);

    if (isLater) {
      // Automatically prompt calendar if opening later for first time
      openScheduleModal();
    }
  });
});

function animateCount() {
  if (countVal) {
    countVal.style.transform = 'scale(1.3)';
    setTimeout(() => { countVal.style.transform = 'scale(1)'; }, 160);
  }
}

const btnMinus = document.getElementById('countMinus');
if (btnMinus) {
  btnMinus.addEventListener('click', () => {
    if (state.count > 1) {
      state.count--;
      if (countVal) countVal.textContent = state.count;
      animateCount();
    }
  });
}

const btnPlus = document.getElementById('countPlus');
if (btnPlus) {
  btnPlus.addEventListener('click', () => {
    if (state.count < 20) {
      state.count++;
      if (countVal) countVal.textContent = state.count;
      animateCount();
    }
  });
}

/* ─── Shift Payment & Escrow Modal Handlers ───────────────── */

function openPaymentModal() {
  const rate = RATES[state.category] || { amount: '₱75', note: 'Minimum 4 hours' };
  const rateVal = parseFloat(state.offeredRate) || (parseFloat((rate.amount || '').replace(/[^0-9.]/g, '')) || 85);
  const count = 1;
  state.count = 1;
  const hours = 4; // Minimum 4 hours
  const MATCH_FEE = 100; // Fixed ₱100 fee to find a matched crew

  let timingText;
  if (state.timing === 'now') {
    timingText = 'ASAP NOW';
  } else {
    timingText = `${formatDateShort(state.scheduledDate)} at ${format12Hour(state.scheduledTime)}`;
  }

  const finalMapLoc = (googleMapLocationInput && googleMapLocationInput.value.trim()) || state.mapLocation || (state.location ? `${state.location}` : 'Cebu City');
  const finalMapsUrl = (state.geo && finalMapLoc === state.geo.label)
    ? state.geo.mapsUrl
    : buildMapsUrlFor(finalMapLoc);

  state.mapLocation = finalMapLoc;
  state.googleMapsUrl = finalMapsUrl;

  if (paySummaryRole) paySummaryRole.textContent = state.roleLabel || 'Crew Member';
  if (paySummaryCat) paySummaryCat.textContent = CAT_LABELS[state.category] || 'Kitchen';
  if (paySummaryTime) paySummaryTime.textContent = timingText;
  if (paySummaryCount) paySummaryCount.textContent = `1 Crew Member (${hours}h min)`;
  if (paySummaryLocation) paySummaryLocation.textContent = finalMapLoc;

  if (payBreakdownMath) payBreakdownMath.textContent = `₱${rateVal}/hr`;
  if (payBreakdownSubtotal) payBreakdownSubtotal.textContent = `₱${MATCH_FEE.toFixed(2)}`;
  if (payBreakdownTotal) payBreakdownTotal.textContent = `₱${MATCH_FEE.toFixed(2)}`;

  state.pendingPayment = {
    rateVal,
    hours,
    count,
    subtotal: MATCH_FEE,
    total: MATCH_FEE,
    feeType: 'Fixed Crew Matching Fee',
    timingText,
    finalMapLoc,
    finalMapsUrl
  };

  if (payNowBtnText) {
    payNowBtnText.textContent = `Pay`;
  }
  if (payNowBtn) {
    payNowBtn.disabled = false;
    payNowBtn.style.background = '';
  }

  if (paymentModal) {
    paymentModal.classList.add('open');
    paymentModal.setAttribute('aria-hidden', 'false');
    const payBody = document.getElementById('paymentModalBody');
    if (payBody) payBody.scrollTop = 0;
  }
}

function closePaymentModal() {
  if (paymentModal) {
    paymentModal.classList.remove('open');
    paymentModal.setAttribute('aria-hidden', 'true');
  }
}

// Payment method tab switching
document.querySelectorAll('.db-pay-method-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.db-pay-method-btn').forEach(b => {
      b.classList.remove('active');
      b.setAttribute('aria-checked', 'false');
    });
    btn.classList.add('active');
    btn.setAttribute('aria-checked', 'true');

    const method = btn.getAttribute('data-method');
    const gcashWrap = document.getElementById('payMethodGcashWrap');
    const mayaWrap = document.getElementById('payMethodMayaWrap');
    const cardWrap = document.getElementById('payMethodCardWrap');

    if (gcashWrap) gcashWrap.style.display = method === 'gcash' ? 'block' : 'none';
    if (mayaWrap) mayaWrap.style.display = method === 'maya' ? 'block' : 'none';
    if (cardWrap) cardWrap.style.display = method === 'card' ? 'block' : 'none';
  });
});

if (paymentCloseXBtn) paymentCloseXBtn.addEventListener('click', closePaymentModal);
if (payCancelBtn) payCancelBtn.addEventListener('click', closePaymentModal);
if (paymentModal) {
  paymentModal.addEventListener('click', (e) => {
    if (e.target === paymentModal) closePaymentModal();
  });
}

// Quick chip clicks
const chipGcashDefault = document.getElementById('chipGcashDefault');
if (chipGcashDefault) {
  chipGcashDefault.addEventListener('click', () => {
    const inp = document.getElementById('payGcashNumber');
    if (inp) inp.value = '917 555 0199';
  });
}
const chipMayaDefault = document.getElementById('chipMayaDefault');
if (chipMayaDefault) {
  chipMayaDefault.addEventListener('click', () => {
    const inp = document.getElementById('payMayaNumber');
    if (inp) inp.value = '918 777 2233';
  });
}
// Toggle Card Number eye visibility
const toggleCardNumBtn = document.getElementById('toggleCardNumBtn');
const payCardNumber = document.getElementById('payCardNumber');
if (toggleCardNumBtn && payCardNumber) {
  toggleCardNumBtn.addEventListener('click', (e) => {
    e.preventDefault();
    e.stopPropagation();
    const eyeOpen = toggleCardNumBtn.querySelector('.db-card-eye-open');
    const eyeClosed = toggleCardNumBtn.querySelector('.db-card-eye-closed');
    const isPassword = payCardNumber.type === 'password';

    if (isPassword) {
      payCardNumber.type = 'text';
      toggleCardNumBtn.setAttribute('aria-label', 'Hide card number');
      toggleCardNumBtn.setAttribute('title', 'Hide card number');
      if (eyeOpen) eyeOpen.style.display = 'none';
      if (eyeClosed) eyeClosed.style.display = 'block';
    } else {
      payCardNumber.type = 'password';
      toggleCardNumBtn.setAttribute('aria-label', 'Show card number');
      toggleCardNumBtn.setAttribute('title', 'Show card number');
      if (eyeOpen) eyeOpen.style.display = 'block';
      if (eyeClosed) eyeClosed.style.display = 'none';
    }
  });

  // Format card number with spaces
  payCardNumber.addEventListener('input', (e) => {
    const val = e.target.value.replace(/\D/g, '').substring(0, 16);
    const formatted = val.match(/.{1,4}/g)?.join(' ') || val;
    e.target.value = formatted;
  });
}

// Toggle Card CVV eye visibility
const toggleCardCvcBtn = document.getElementById('toggleCardCvcBtn');
const payCardCvc = document.getElementById('payCardCvc');
if (toggleCardCvcBtn && payCardCvc) {
  toggleCardCvcBtn.addEventListener('click', (e) => {
    e.preventDefault();
    e.stopPropagation();
    const eyeOpen = toggleCardCvcBtn.querySelector('.db-cvv-eye-open');
    const eyeClosed = toggleCardCvcBtn.querySelector('.db-cvv-eye-closed');
    const isPassword = payCardCvc.type === 'password';

    if (isPassword) {
      payCardCvc.type = 'text';
      toggleCardCvcBtn.setAttribute('aria-label', 'Hide CVV');
      toggleCardCvcBtn.setAttribute('title', 'Hide CVV');
      if (eyeOpen) eyeOpen.style.display = 'none';
      if (eyeClosed) eyeClosed.style.display = 'block';
    } else {
      payCardCvc.type = 'password';
      toggleCardCvcBtn.setAttribute('aria-label', 'Show CVV');
      toggleCardCvcBtn.setAttribute('title', 'Show CVV');
      if (eyeOpen) eyeOpen.style.display = 'block';
      if (eyeClosed) eyeClosed.style.display = 'none';
    }
  });
}

// ── Step 3 "Find a Crew" -> Checks Auth First, then Triggers Payment! ──
toStep4Btn.addEventListener('click', () => {
  const currentSession = getSession();
  if (!currentSession) {
    authTriggerSource = 'step';
    openEmployerAuthModal();
    return;
  }
  // Logged in: proceed directly to payment and finding crew
  openPaymentModal();
});

// ── Pay Now -> Process Payment, Then Find Matching Crew ──
if (payNowBtn) {
  payNowBtn.addEventListener('click', () => {
    const payInfo = state.pendingPayment || { total: 100 };
    const methodTab = document.querySelector('.db-pay-method-btn.active');
    const method = methodTab ? methodTab.getAttribute('data-method') : 'gcash';
    const methodLabels = { gcash: 'GCash', maya: 'Maya', card: 'Card' };
    const methodLabel = methodLabels[method] || 'GCash';

    payNowBtn.disabled = true;
    if (payNowBtnText) {
      payNowBtnText.innerHTML = '<span class="db-btn-spinner"></span> Authorizing Escrow Payment…';
    }

    setTimeout(() => {
      if (payNowBtnText) {
        payNowBtnText.innerHTML = '✓ Payment Successful!';
      }
      payNowBtn.style.background = '#10B981';

      setTimeout(() => {
        closePaymentModal();
        executeFindCrewAfterPayment(payInfo, methodLabel);
      }, 400);
    }, 500);
  });
}

function executeFindCrewAfterPayment(payInfo, methodLabel) {
  state.paid = true;
  state.paymentMethod = methodLabel;
  state.paymentAmount = payInfo.total;

  const rate = RATES[state.category] || { amount: '₱75', note: 'Minimum 4 hours' };
  const timingText = payInfo.timingText || (state.timing === 'now' ? 'ASAP NOW' : `${formatDateShort(state.scheduledDate)} at ${format12Hour(state.scheduledTime)}`);
  const finalMapLoc = payInfo.finalMapLoc || (googleMapLocationInput && googleMapLocationInput.value.trim()) || state.mapLocation || (state.location ? `${state.location}` : 'Cebu City');
  const finalMapsUrl = payInfo.finalMapsUrl || (window.InstantCrewShared ? window.InstantCrewShared.formatGoogleMapsUrl(finalMapLoc) : `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(finalMapLoc)}`);

  const bookingId = Date.now();
  const bookingData = {
    id: bookingId,
    category: state.category,
    roleLabel: state.roleLabel || 'Crew Member',
    employmentType: state.employmentType,
    offeredRate: state.offeredRate,
    ratePeriod: state.ratePeriod,
    location: state.location,
    mapLocation: finalMapLoc,
    googleMapsUrl: finalMapsUrl,
    geo: state.geo ? { lat: state.geo.lat, lng: state.geo.lng } : null,
    timing: timingText,
    count: state.count,
    crewRate: rate.amount,
    status: 'matching', // Waiting for crew acceptance
    bookedAt: new Date().toISOString(),
    paid: true,
    paymentMethod: methodLabel,
    paymentAmount: payInfo.total || 100,
    escrowStatus: 'Fixed Match Fee Paid (₱100.00)'
  };

  state.currentBooking = bookingData;
  saveBookingToSession(bookingData);

  // Sync to shared jobs pool so applicants can see it and accept
  if (window.InstantCrewShared) {
    const s = getSession();
    const sharedJob = window.InstantCrewShared.addEmployerJob({
      ...bookingData,
      name: s ? s.name : 'Sample Employer',
      email: s ? s.email : 'sample@gmail.com',
      venue: s ? `${s.name}'s Shift` : 'Sample Employer Venue'
    });
    state.currentJobId = sharedJob.id;
  }

  // Show escrow status in Step 4 matching pane
  if (matchEscrowPill) {
    matchEscrowPill.style.display = 'inline-flex';
    if (matchEscrowText) {
      matchEscrowText.textContent = `₱${(payInfo.total || 100).toFixed(2)} via ${methodLabel}`;
    }
  }

  if (matchFailed) matchFailed.style.display = 'none';

  goStep(4);

  // Reset results pane — loading state active, results hidden until accepted!
  matchLoading.style.display = 'flex';
  matchResults.style.display = 'none';

  // Start 3-minute search timer
  startSearchTimer();

  // Start checking for crew acceptance (listening & polling)
  startCheckingForAcceptance();
}

/* ─── Step 4: Crew Results & 3-Minute Search Window ─────────── */

const SEARCH_MAX_SECONDS = 180; // Maximum of 3 mins

function formatSearchTimer(seconds) {
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}m ${secs < 10 ? '0' : ''}${secs}s remaining`;
}

function updateSearchTimerUI() {
  const count = Math.max(0, state.searchCountdown !== undefined ? state.searchCountdown : SEARCH_MAX_SECONDS);
  if (dbTryTimer) {
    dbTryTimer.textContent = formatSearchTimer(count);
    dbTryTimer.style.display = 'inline-block';
  }
  const fill = document.getElementById('dbSearchProgressFill');
  if (fill) {
    const pct = Math.max(0, Math.min(100, (count / SEARCH_MAX_SECONDS) * 100));
    fill.style.width = `${pct}%`;
  }
}

function startSearchTimer() {
  stopSearchTimer();
  state.searchCountdown = SEARCH_MAX_SECONDS;
  updateSearchTimerUI();
  state.searchTimerInterval = setInterval(() => {
    state.searchCountdown--;
    updateSearchTimerUI();
    if (state.searchCountdown <= 0) {
      stopSearchTimer();
      handleSearchTimeout();
    }
  }, 1000);
}

function stopSearchTimer() {
  if (state.searchTimerInterval) {
    clearInterval(state.searchTimerInterval);
    state.searchTimerInterval = null;
  }
}

function handleSearchTimeout() {
  stopSearchTimer();
  if (state.matchCheckInterval) {
    clearInterval(state.matchCheckInterval);
    state.matchCheckInterval = null;
  }

  if (matchLoading) matchLoading.style.display = 'none';
  if (matchFailed) matchFailed.style.display = 'block';

  state.paid = false; // Payment consumed

  // Update session
  const s = getSession();
  if (s && s.bookings) {
    const b = s.bookings.find(x => String(x.id) === String(state.currentBooking ? state.currentBooking.id : ''));
    if (b) {
      b.status = 'unfulfilled';
      b.paid = false;
      sessionStorage.setItem(SESSION_KEY, JSON.stringify(s));
    }
  }
}

// Demo simulate timeout button handler (triggers 3-min timeout)
if (simulateTimeoutBtn) {
  simulateTimeoutBtn.addEventListener('click', () => {
    handleSearchTimeout();
  });
}

// Repay match fee button handler (when 3-minute search expires)
if (repayMatchFeeBtn) {
  repayMatchFeeBtn.addEventListener('click', () => {
    openPaymentModal();
  });
}

// Edit shift from failed view
if (editShiftFromFailedBtn) {
  editShiftFromFailedBtn.addEventListener('click', () => {
    if (matchFailed) matchFailed.style.display = 'none';
    goStep(3);
  });
}

function checkCrewAcceptance() {
  if (!state.currentJobId || !window.InstantCrewShared) return;
  const jobs = window.InstantCrewShared.getJobs();
  const job = jobs.find(j => j.id === state.currentJobId);
  if (job && (job.acceptedCount > 0 || (job.acceptedCrew && job.acceptedCrew.length > 0))) {
    onCrewAccepted(job);
  }
}

function startCheckingForAcceptance() {
  if (state.matchCheckInterval) {
    clearInterval(state.matchCheckInterval);
    state.matchCheckInterval = null;
  }
  checkCrewAcceptance();
  state.matchCheckInterval = setInterval(checkCrewAcceptance, 600);
}

function onCrewAccepted(job) {
  stopSearchTimer();
  if (matchFailed) matchFailed.style.display = 'none';

  if (state.matchCheckInterval) {
    clearInterval(state.matchCheckInterval);
    state.matchCheckInterval = null;
  }

  if (matchResults.style.display === 'block') return; // already rendered

  matchLoading.style.display = 'none';
  matchResults.style.display = 'block';

  const acceptedCrew = job.acceptedCrew || [];
  renderCrewResults(acceptedCrew, job);

  const firstWorker = acceptedCrew[0] || 'A crew member';
  if (resultsAcceptedNote) {
    resultsAcceptedNote.textContent = `${firstWorker} accepted your ${job.role || state.roleLabel || 'crew'} shift offer and is ready to work!`;
  }

  // Show push toast
  if (window.InstantCrewShared) {
    window.InstantCrewShared.showPushToast('Offer Accepted!', `${firstWorker} accepted your ${job.role || state.roleLabel || 'crew'} shift offer!`, 'accept');
  }

  // Update session booking
  const s = getSession();
  if (s && s.bookings) {
    const b = s.bookings.find(x => String(x.id) === String(state.currentBooking ? state.currentBooking.id : ''));
    if (b) {
      b.status = 'accepted';
      b.acceptedCrew = acceptedCrew;
      sessionStorage.setItem(SESSION_KEY, JSON.stringify(s));
    }
  }
}

// Demo simulation button
if (simulateAcceptBtn) {
  simulateAcceptBtn.addEventListener('click', () => {
    if (!state.currentJobId || !window.InstantCrewShared) return;
    const candidateCrew = (CREW_DATA[state.category] && CREW_DATA[state.category][0]) || { name: 'Angelo Lopez' };
    window.InstantCrewShared.acceptJob(state.currentJobId, candidateCrew.name);
    checkCrewAcceptance();
  });
}

function renderCrewResults(acceptedCrew = [], job = null) {
  const defaultCrew = CREW_DATA[state.category] || [];
  const rate = RATES[state.category] || { amount: '₱75', note: 'Minimum 4 hours' };

  // Update rate
  rateAmount.innerHTML = `${rate.amount}<span>/hr</span>`;
  document.getElementById('rateAmount').nextElementSibling.textContent = rate.note;

  // Render crew cards
  crewList.innerHTML = '';
  const listToRender = (acceptedCrew && acceptedCrew.length > 0)
    ? acceptedCrew
    : (defaultCrew.slice(0, 1).map(c => c.name));

  listToRender.forEach((workerName, i) => {
    const matched = defaultCrew.find(c => c.name === workerName) || {
      initials: workerName.split(' ').map(w => w[0]).join('').slice(0, 2).toUpperCase() || 'CW',
      name: workerName,
      role: state.roleLabel || 'Crew Member',
      rating: 4.9,
      distance: '0.8 km',
      jobs: 142
    };

    const card = document.createElement('div');
    card.className = 'db-crew-card';
    card.setAttribute('role', 'listitem');
    card.style.animationDelay = `${i * 120}ms`;

    card.innerHTML = `
            <span class="db-avatar">${matched.initials}</span>
            <div class="db-crew-info">
                <p class="db-crew-name">${matched.name}</p>
                <p class="db-crew-meta">
                    <span class="db-crew-star" style="display:inline-flex; align-items:center; vertical-align:-2px; margin-right:2px;">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="#F59E0B" stroke="#F59E0B" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" class="icon-tabler-star"><path stroke="none" d="M0 0h24v24H0z" fill="none"/><path d="M12 17.75l-6.172 3.245l1.179 -6.873l-5 -4.867l6.9 -1l3.086 -6.253l3.086 6.253l6.9 1l-5 4.867l1.179 6.873z" /></svg>
                    </span> ${matched.rating}
                    &middot; ${matched.distance} away
                    &middot; ${matched.jobs} jobs
                </p>
            </div>
            <span class="db-crew-role-tag" style="background:#DCFCE7; color:#15803D; font-weight:700;">✓ Accepted</span>
        `;

    crewList.appendChild(card);
  });
}

confirmBooking.addEventListener('click', () => {
  const rate = RATES[state.category] || { amount: '₱75', note: 'Minimum 4 hours' };

  let timingText;
  if (state.timing === 'now') {
    timingText = 'As soon as possible';
  } else {
    const dateStr = formatDateShort(state.scheduledDate);
    const timeStr = format12Hour(state.scheduledTime);
    timingText = `${dateStr} at ${timeStr}`;
  }

  document.getElementById('dCat').textContent = CAT_LABELS[state.category] || '—';
  document.getElementById('dRole').textContent = state.roleLabel || '—';
  document.getElementById('dEmp').textContent = state.employmentType === 'full-time' ? 'Full-time' : 'Part-time';
  document.getElementById('dOfferedRate').textContent = `₱${state.offeredRate || '85'} ${state.ratePeriod}`;
  document.getElementById('dLocation').textContent = state.location || 'Cebu City';
  if (dMapLocation) {
    if (state.mapLocation) {
      dMapLocation.innerHTML = `<span style="font-weight:600;">${state.mapLocation}</span> <a href="${state.googleMapsUrl || '#'}" target="_blank" rel="noopener noreferrer" style="color:#EA4335; font-size:0.8rem; margin-left:6px; font-weight:700; text-decoration:none;">Open Map ↗</a>`;
    } else {
      dMapLocation.textContent = 'City Center';
    }
  }
  document.getElementById('dTime').textContent = timingText;
  document.getElementById('dCount').textContent = '1 Crew Member';
  document.getElementById('dRate').textContent = `${rate.amount}/hr (${rate.note.toLowerCase()})`;
  if (dPayment) {
    const paidTotal = state.paymentAmount ? `₱${state.paymentAmount.toFixed(2)}` : '₱100.00';
    dPayment.innerHTML = `<span style="color:#059669; font-weight:700;">✓ Match Fee Paid</span> (${paidTotal} via ${state.paymentMethod || 'GCash'})`;
  }

  // Finalize booking in session to active
  const s = getSession();
  if (s && s.bookings) {
    const b = s.bookings.find(x => String(x.id) === String(state.currentBooking ? state.currentBooking.id : ''));
    if (b) {
      b.status = 'active';
      b.paid = true;
      b.paymentAmount = state.paymentAmount;
      b.paymentMethod = state.paymentMethod;
      sessionStorage.setItem(SESSION_KEY, JSON.stringify(s));
    }
  }
  renderBookings();

  goStep(5);
});

/* ─── Step 5: Booking Details Accordion & Reset ──────────── */

detailsToggle.addEventListener('click', () => {
  const isOpen = detailsToggle.getAttribute('aria-expanded') === 'true';
  detailsToggle.setAttribute('aria-expanded', String(!isOpen));

  if (!isOpen) {
    detailsBox.removeAttribute('hidden');
    detailsToggle.style.borderRadius = '18px 18px 0 0';
  } else {
    detailsBox.setAttribute('hidden', '');
    detailsToggle.style.borderRadius = '';
  }
});

bookAnother.addEventListener('click', () => {
  // Reset state
  state.category = null;
  state.role = null;
  state.roleLabel = null;
  state.employmentType = 'full-time';
  state.offeredRate = '85';
  state.ratePeriod = 'per hour';
  state.location = 'Cebu City';
  state.mapLocation = '';
  state.googleMapsUrl = '';
  state.geo = null;
  if (googleMapLocationInput) googleMapLocationInput.value = '';
  if (clearMapLocationBtn) clearMapLocationBtn.style.display = 'none';
  state.timing = 'now';
  state.scheduledDate = new Date(TODAY);
  state.scheduledTime = '08:00';
  state.count = 1;
  state.currentJobId = null;
  state.currentBooking = null;
  state.paid = false;
  state.pendingPayment = null;
  state.paymentMethod = 'GCash';
  stopSearchTimer();
  if (matchFailed) matchFailed.style.display = 'none';
  if (state.matchCheckInterval) {
    clearInterval(state.matchCheckInterval);
    state.matchCheckInterval = null;
  }
  if (matchResults) matchResults.style.display = 'none';
  if (matchLoading) matchLoading.style.display = 'flex';

  // Reset Step 1 UI
  catCards.forEach(c => c.setAttribute('aria-pressed', 'false'));
  if (rolesSelect) rolesSelect.value = '';
  if (roleTriggerText) {
    roleTriggerText.textContent = 'Select a role…';
    roleTriggerText.classList.add('placeholder');
  }
  if (roleSelectedBadge) roleSelectedBadge.style.display = 'none';
  if (rolesCountPill) rolesCountPill.style.display = 'none';
  closeRoleDropdown();
  rolesWrap.classList.remove('open');
  toStep2Btn.disabled = true;
  step1Hint.textContent = 'Select a category to continue.';

  // Reset Step 2 UI
  empToggles.forEach(t => t.classList.toggle('active', t.dataset.emp === 'full-time'));
  offeredRateInput.value = '85';
  selectRatePeriod('per hour');
  selectLocation('Cebu City');

  // Reset Step 3 UI
  document.querySelectorAll('.db-toggle').forEach(b => b.classList.remove('active'));
  document.querySelector('.db-toggle[data-timing="now"]').classList.add('active');
  laterField.classList.remove('open');
  if (countVal) countVal.textContent = '1';
  scheduleDisplayDate.textContent = formatDateShort(state.scheduledDate);
  scheduleDisplayTime.textContent = `${format12Hour(state.scheduledTime)} · Scheduled Shift`;

  // Reset Step 5 UI
  detailsToggle.setAttribute('aria-expanded', 'false');
  detailsBox.setAttribute('hidden', '');
  detailsToggle.style.borderRadius = '';

  // Show progress bar
  progress.style.display = 'flex';

  goStep(1);
});

/* ─── Back buttons ────────────────────────────────────────── */

document.querySelectorAll('.db-back-btn').forEach(btn => {
  btn.addEventListener('click', () => {
    const target = Number(btn.dataset.back);

    if (target < 4 && state.matchCheckInterval) {
      clearInterval(state.matchCheckInterval);
      state.matchCheckInterval = null;
    }

    if (target === 4) {
      matchLoading.style.display = 'flex';
      matchResults.style.display = 'none';
    }

    goStep(target);
  });
});

/* ─── Init ────────────────────────────────────────────────── */

// Initial setup of date/time preview in Step 3
if (scheduleDisplayDate) {
  scheduleDisplayDate.textContent = formatDateShort(state.scheduledDate);
}
if (scheduleDisplayTime) {
  scheduleDisplayTime.textContent = `${format12Hour(state.scheduledTime)} · Scheduled Shift`;
}

/* ═══════════════════════════════════════════════════════════
   AUTH GUARD & SESSION INTEGRATION
   ═══════════════════════════════════════════════════════════ */

const SESSION_KEY = 'ic_employer_session';

function getSession() {
  try {
    return JSON.parse(sessionStorage.getItem(SESSION_KEY));
  } catch {
    return null;
  }
}

// Initialize profile display (works for logged-in or guest users)
populateProfile();

/* ═══════════════════════════════════════════════════════════
   EMPLOYER PROFILE PANEL
   ═══════════════════════════════════════════════════════════ */

const profileOverlay = document.getElementById('profileOverlay');
const openProfilePanelBtn = document.getElementById('openProfilePanel');
const closeProfilePanelBtn = document.getElementById('closeProfilePanel');
const logoutBtn = document.getElementById('logoutBtn');
const tabActiveBtn = document.getElementById('tabActive');
const tabPreviousBtn = document.getElementById('tabPrevious');
const panelActiveEl = document.getElementById('panelActive');
const panelPreviousEl = document.getElementById('panelPrevious');
const activeBookingsList = document.getElementById('activeBookingsList');
const previousBookingsList = document.getElementById('previousBookingsList');
const activeEmpty = document.getElementById('activeEmpty');
const previousEmpty = document.getElementById('previousEmpty');

/* Populate profile UI from session */
function populateProfile() {
  const s = getSession();
  const profileInitialsEl = document.getElementById('profileInitials');
  const panelAvatarEl = document.getElementById('panelAvatarInitials');
  const panelNameEl = document.getElementById('panelUserName');
  const panelEmailEl = document.getElementById('panelUserEmail');
  const statYearEl = document.getElementById('statYear');
  const headerLogoEl = document.getElementById('headerLogoLink');
  const headerBackArrowEl = document.getElementById('headerBackArrow');
  const headerLoginBtn = document.getElementById('headerLoginBtn');
  const headerNotifWrap = document.getElementById('headerNotifWrap');
  const openProfilePanelBtn = document.getElementById('openProfilePanel');

  if (!s) {
    if (headerLogoEl) {
      headerLogoEl.href = 'index.html';
      headerLogoEl.setAttribute('aria-label', 'Back to Landing Page');
      headerLogoEl.setAttribute('title', 'Back to Home');
    }
    if (headerBackArrowEl) {
      headerBackArrowEl.style.display = 'inline-flex';
    }
    if (headerLoginBtn) {
      headerLoginBtn.style.display = 'inline-flex';
    }
    if (headerNotifWrap) {
      headerNotifWrap.style.display = 'none';
    }
    if (openProfilePanelBtn) {
      openProfilePanelBtn.style.display = 'none';
    }
    if (profileInitialsEl) profileInitialsEl.textContent = 'EP';
    if (panelAvatarEl) panelAvatarEl.textContent = 'EP';
    if (panelNameEl) panelNameEl.textContent = 'Employer';
    if (panelEmailEl) panelEmailEl.textContent = 'Sign in to access saved records';
    if (statYearEl) statYearEl.textContent = 2026;
    return;
  }

  if (headerLogoEl) {
    headerLogoEl.href = 'employer.html';
    headerLogoEl.setAttribute('aria-label', 'Instant Crew Employer Dashboard');
    headerLogoEl.removeAttribute('title');
  }
  if (headerBackArrowEl) {
    headerBackArrowEl.style.display = 'none';
  }
  if (headerLoginBtn) {
    headerLoginBtn.style.display = 'none';
  }
  if (headerNotifWrap) {
    headerNotifWrap.style.display = '';
  }
  if (openProfilePanelBtn) {
    openProfilePanelBtn.style.display = 'inline-flex';
  }

  const nameParts = (s.name || 'SE').split(' ');
  const initials = nameParts.map(n => n[0]).join('').slice(0, 2).toUpperCase();

  if (profileInitialsEl) profileInitialsEl.textContent = initials;
  if (panelAvatarEl) panelAvatarEl.textContent = initials;
  if (panelNameEl) panelNameEl.textContent = s.name || 'Employer';
  if (panelEmailEl) panelEmailEl.textContent = s.email || '';
  if (statYearEl) statYearEl.textContent = s.joinedYear || 2026;
}

/* Open / Close Panel */
function openProfilePanel() {
  if (!getSession()) {
    openEmployerAuthModal();
    return;
  }
  profileOverlay.classList.add('open');
  profileOverlay.setAttribute('aria-hidden', 'false');
  document.body.style.overflow = 'hidden';
  populateProfile();
  switchBookingTab('active');
  renderBookings();
}

function closeProfilePanel() {
  profileOverlay.classList.remove('open');
  profileOverlay.setAttribute('aria-hidden', 'true');
  document.body.style.overflow = '';
}

if (openProfilePanelBtn) {
  openProfilePanelBtn.addEventListener('click', openProfilePanel);
}
if (closeProfilePanelBtn) {
  closeProfilePanelBtn.addEventListener('click', closeProfilePanel);
}
if (profileOverlay) {
  profileOverlay.addEventListener('click', e => {
    if (e.target === profileOverlay) closeProfilePanel();
  });
}
document.addEventListener('keydown', e => {
  if (e.key === 'Escape' && profileOverlay && profileOverlay.classList.contains('open')) {
    closeProfilePanel();
  }
});

/* ═══════════════════════════════════════════════════════════
   STEP 3 IN-PAGE EMPLOYER AUTH MODAL CONTROLLER
   Prompted when employer clicks "Find a Crew" without logging in
   ═══════════════════════════════════════════════════════════ */
let authTriggerSource = 'step'; // 'step' or 'header'
const employerAuthModal = document.getElementById('employerAuthModal');
const employerAuthCloseBtn = document.getElementById('employerAuthCloseBtn');
const tabEmployerLogin = document.getElementById('tabEmployerLogin');
const tabEmployerSignup = document.getElementById('tabEmployerSignup');
const employerModalLoginForm = document.getElementById('employerModalLoginForm');
const employerModalSignupForm = document.getElementById('employerModalSignupForm');
const employerAuthError = document.getElementById('employerAuthError');
const headerLoginBtn = document.getElementById('headerLoginBtn');

if (headerLoginBtn) {
  headerLoginBtn.addEventListener('click', () => {
    authTriggerSource = 'header';
    openEmployerAuthModal();
  });
}

function openEmployerAuthModal() {
  if (!employerAuthModal) return;
  employerAuthModal.classList.add('open');
  employerAuthModal.setAttribute('aria-hidden', 'false');
  document.body.style.overflow = 'hidden';
  if (employerAuthError) employerAuthError.style.display = 'none';
}

function closeEmployerAuthModal() {
  if (!employerAuthModal) return;
  employerAuthModal.classList.remove('open');
  employerAuthModal.setAttribute('aria-hidden', 'true');
  document.body.style.overflow = '';
}

if (employerAuthCloseBtn) {
  employerAuthCloseBtn.addEventListener('click', closeEmployerAuthModal);
}
if (employerAuthModal) {
  employerAuthModal.addEventListener('click', e => {
    if (e.target === employerAuthModal) closeEmployerAuthModal();
  });
}
document.addEventListener('keydown', e => {
  if (e.key === 'Escape' && employerAuthModal && employerAuthModal.classList.contains('open')) {
    closeEmployerAuthModal();
  }
});

// Auth Tab switching
if (tabEmployerLogin && tabEmployerSignup) {
  tabEmployerLogin.addEventListener('click', () => {
    tabEmployerLogin.classList.add('active');
    tabEmployerSignup.classList.remove('active');
    if (employerModalLoginForm) employerModalLoginForm.style.display = 'block';
    if (employerModalSignupForm) employerModalSignupForm.style.display = 'none';
    if (employerAuthError) employerAuthError.style.display = 'none';
  });

  tabEmployerSignup.addEventListener('click', () => {
    tabEmployerSignup.classList.add('active');
    tabEmployerLogin.classList.remove('active');
    if (employerModalLoginForm) employerModalLoginForm.style.display = 'none';
    if (employerModalSignupForm) employerModalSignupForm.style.display = 'block';
    if (employerAuthError) employerAuthError.style.display = 'none';
  });
}

function showEmployerAuthError(msg) {
  if (employerAuthError) {
    employerAuthError.textContent = msg;
    employerAuthError.style.display = 'block';
  }
}

// Auth modal password visibility toggle
document.querySelectorAll('.db-auth-toggle-pass').forEach(btn => {
  btn.addEventListener('click', (e) => {
    e.preventDefault();
    e.stopPropagation();
    const wrap = btn.closest('.db-auth-input-wrap');
    const input = wrap ? wrap.querySelector('input') : null;
    const eyeOpen = btn.querySelector('.eye-open');
    const eyeClosed = btn.querySelector('.eye-closed');
    if (!input) return;

    const isPassword = input.type === 'password';
    input.type = isPassword ? 'text' : 'password';
    if (isPassword) {
      input.focus();
      if (eyeOpen) eyeOpen.style.display = 'none';
      if (eyeClosed) eyeClosed.style.display = 'block';
      btn.setAttribute('aria-label', 'Hide password');
      btn.setAttribute('title', 'Hide password');
    } else {
      input.focus();
      if (eyeOpen) eyeOpen.style.display = 'block';
      if (eyeClosed) eyeClosed.style.display = 'none';
      btn.setAttribute('aria-label', 'Show password');
      btn.setAttribute('title', 'Show password');
    }
  });
});

// Log In form handler
if (employerModalLoginForm) {
  employerModalLoginForm.addEventListener('submit', () => {
    const email = document.getElementById('empModalLoginEmail').value.trim().toLowerCase();
    const pass = document.getElementById('empModalLoginPass').value;
    const btn = document.getElementById('btnSubmitModalLogin');

    if (!email || pass.length < 4) {
      showEmployerAuthError('Please enter a valid email and password (minimum 4 characters).');
      return;
    }

    if (btn) {
      btn.disabled = true;
      btn.innerHTML = '<span class="db-btn-spinner"></span> Logging in…';
    }

    setTimeout(() => {
      let existingBookings = [];
      try {
        const prev = JSON.parse(sessionStorage.getItem(SESSION_KEY));
        if (prev && Array.isArray(prev.bookings)) existingBookings = prev.bookings;
      } catch (e) { }

      const userSession = {
        name: email === 'sample@gmail.com' ? 'Sample Employer' : email.split('@')[0],
        email: email,
        company: 'Sample Co.',
        joinedYear: 2026,
        bookings: existingBookings
      };

      sessionStorage.setItem(SESSION_KEY, JSON.stringify(userSession));
      populateProfile();
      closeEmployerAuthModal();

      if (btn) {
        btn.disabled = false;
        btn.innerHTML = '<span>Log In &amp; Continue Finding Crew</span>';
      }

      // Only advance to payment modal if triggered from Step 3 "Find a Crew"
      if (authTriggerSource === 'step') {
        openPaymentModal();
      }
      authTriggerSource = 'step';
    }, 400);
  });
}

// Sign Up form handler
if (employerModalSignupForm) {
  employerModalSignupForm.addEventListener('submit', () => {
    const company = document.getElementById('empModalSignupCompany').value.trim();
    const contact = document.getElementById('empModalSignupContact').value.trim();
    const email = document.getElementById('empModalSignupEmail').value.trim().toLowerCase();
    const pass = document.getElementById('empModalSignupPass').value;
    const btn = document.getElementById('btnSubmitModalSignup');

    if (!email || !contact || pass.length < 4) {
      showEmployerAuthError('Please fill in all required fields (password minimum 4 characters).');
      return;
    }

    if (btn) {
      btn.disabled = true;
      btn.innerHTML = '<span class="db-btn-spinner"></span> Creating account…';
    }

    setTimeout(() => {
      const userSession = {
        name: contact,
        email: email,
        company: company || 'My Company',
        joinedYear: 2026,
        bookings: []
      };

      sessionStorage.setItem(SESSION_KEY, JSON.stringify(userSession));
      populateProfile();
      closeEmployerAuthModal();

      if (btn) {
        btn.disabled = false;
        btn.innerHTML = '<span>Create Account &amp; Continue Finding Crew</span>';
      }

      // Only advance to payment modal if triggered from Step 3 "Find a Crew"
      if (authTriggerSource === 'step') {
        openPaymentModal();
      }
      authTriggerSource = 'step';
    }, 450);
  });
}

/* ── Bookings Tabs ─────────────────────────────────────── */

function switchBookingTab(tab) {
  const isActive = tab === 'active';
  if (tabActiveBtn) {
    tabActiveBtn.classList.toggle('active', isActive);
    tabActiveBtn.setAttribute('aria-selected', String(isActive));
  }
  if (tabPreviousBtn) {
    tabPreviousBtn.classList.toggle('active', !isActive);
    tabPreviousBtn.setAttribute('aria-selected', String(!isActive));
  }

  if (panelActiveEl) {
    panelActiveEl.hidden = !isActive;
    panelActiveEl.classList.toggle('active', isActive);
  }
  if (panelPreviousEl) {
    panelPreviousEl.hidden = isActive;
    panelPreviousEl.classList.toggle('active', !isActive);
  }
}

if (tabActiveBtn) tabActiveBtn.addEventListener('click', () => switchBookingTab('active'));
if (tabPreviousBtn) tabPreviousBtn.addEventListener('click', () => switchBookingTab('previous'));

/* ── Default Seed Bookings for Employer ─────────────────── */
const DEFAULT_EMPLOYER_ACTIVE_BOOKINGS = [
  {
    id: 'contract-active-chef',
    contractId: 'contract-active-chef',
    category: 'kitchen',
    role: 'CHEF',
    roleLabel: 'Head Line Chef',
    employmentType: 'full-time',
    location: "kianberong2005's Shift",
    mapLocation: "kianberong2005's Shift",
    timing: 'Contract In Progress',
    count: 1,
    crewRate: '₱85',
    status: 'active',
    transitStatus: 'on_the_way',
    departedAt: '12:17 AM',
    workerName: 'kianberong2005',
    acceptedCrew: ['kianberong2005'],
    bookedAt: 'Today'
  }
];

const DEFAULT_EMPLOYER_PREVIOUS_BOOKINGS = [
  {
    id: 'contract-demo-2',
    contractId: 'contract-demo-2',
    category: 'kitchen',
    role: 'Line Cook',
    roleLabel: 'Line Cook / Prep Assistant',
    employmentType: 'part-time',
    location: 'Bistro Moderne — Downtown',
    mapLocation: 'Bistro Moderne — Downtown',
    timing: 'Today, 1:00 PM – 5:00 PM (4 hrs)',
    count: 1,
    crewRate: '₱95',
    status: 'completed',
    endedBy: 'employer',
    workerName: 'Ronald Mendoza',
    acceptedCrew: ['Ronald Mendoza'],
    bookedAt: 'Yesterday'
  },
  {
    id: 'contract-demo-1',
    contractId: 'contract-demo-1',
    category: 'helpers',
    role: 'Event Helper',
    roleLabel: 'Event Setup & Banquet Helper',
    employmentType: 'part-time',
    location: 'Grand Ballroom & Pavilion',
    mapLocation: 'Grand Ballroom & Pavilion',
    timing: 'Yesterday, 9:00 AM – 2:00 PM (5 hrs)',
    count: 2,
    crewRate: '₱75',
    status: 'completed',
    endedBy: 'employer',
    workerName: 'Marco Reyes',
    acceptedCrew: ['Marco Reyes', 'Jayson V.'],
    bookedAt: '2 days ago'
  },
  {
    id: 'contract-demo-3',
    contractId: 'contract-demo-3',
    category: 'delivery',
    role: 'Motorcycle Rider',
    roleLabel: 'Express Delivery Rider',
    employmentType: 'full-time',
    location: 'Metro Logistics Express Hub',
    mapLocation: 'Metro Logistics Express Hub',
    timing: 'Full-time Day Shift (8 hrs / day)',
    count: 1,
    crewRate: '₱85',
    status: 'completed',
    endedBy: 'worker',
    workerName: 'David Cruz',
    acceptedCrew: ['David Cruz'],
    bookedAt: '3 days ago'
  }
];

/* ── Sync Bookings With Shared Contracts / Jobs State ──── */
function syncBookingsWithSharedState() {
  const s = getSession();
  if (!s) return;
  if (!s.bookings) s.bookings = [];

  // 1. If s.bookings is empty, seed default active and previous bookings so neither tab is empty
  if (s.bookings.length === 0) {
    s.bookings = [
      ...DEFAULT_EMPLOYER_ACTIVE_BOOKINGS.map(b => ({ ...b })),
      ...DEFAULT_EMPLOYER_PREVIOUS_BOOKINGS.map(b => ({ ...b }))
    ];
  }

  if (!window.InstantCrewShared) {
    sessionStorage.setItem(SESSION_KEY, JSON.stringify(s));
    return;
  }

  const contracts = window.InstantCrewShared.getActiveContracts();
  const jobs = window.InstantCrewShared.getJobs();
  let changed = false;

  // 2. Import contracts from window.InstantCrewShared into s.bookings so active and ended crew appear in employer profile
  contracts.forEach(c => {
    // Find matching booking in s.bookings
    const existing = s.bookings.find(b =>
      String(b.id) === String(c.id) ||
      String(b.id) === String(c.bookingRefId) ||
      String(b.id) === String(c.jobId) ||
      (c.jobId && String(b.id) === 'emp-shift-' + c.id) ||
      (b.contractId && String(b.contractId) === String(c.id))
    );

    if (!existing) {
      // Add contract as a booking in employer session
      s.bookings.unshift({
        id: c.bookingRefId || c.id,
        jobId: c.jobId || null,
        contractId: c.id,
        category: c.category || 'kitchen',
        role: c.role || c.title || 'Crew Member',
        roleLabel: (c.title || c.role || 'Crew Member').replace(/\s*\((Full-Time|Part-Time)\)/gi, '').trim(),
        employmentType: c.employmentType || 'full-time',
        location: c.venue || 'On-site Location',
        mapLocation: c.mapLocation || c.location || c.venue || 'Cebu City',
        googleMapsUrl: c.googleMapsUrl || null,
        timing: c.timing || 'Contract In Progress',
        count: 1,
        crewRate: c.rate || '₱85',
        status: c.status === 'active' ? 'active' : 'completed',
        transitStatus: c.transitStatus || null,
        departedAt: c.departedAt || null,
        arrivedAt: c.arrivedAt || null,
        workerName: c.workerName || 'kianberong2005',
        acceptedCrew: [c.workerName || 'kianberong2005'],
        bookedAt: c.startedAt || 'Today'
      });
      changed = true;
    } else {
      // Update existing booking from contract state
      if (c.status === 'ended' && existing.status !== 'completed') {
        existing.status = 'completed';
        existing.endedBy = c.endedBy || 'worker';
        changed = true;
      } else if (c.status === 'active' && existing.status !== 'active') {
        existing.status = 'active';
        changed = true;
      }
      if (c.workerName && (!existing.acceptedCrew || !existing.acceptedCrew.includes(c.workerName))) {
        existing.acceptedCrew = [c.workerName];
        existing.workerName = c.workerName;
        changed = true;
      }
      if (existing.transitStatus !== c.transitStatus || existing.departedAt !== c.departedAt || existing.arrivedAt !== c.arrivedAt) {
        existing.transitStatus = c.transitStatus;
        existing.departedAt = c.departedAt;
        existing.arrivedAt = c.arrivedAt;
        changed = true;
      }
    }
  });

  // 3. Ensure state.currentBooking from active dashboard is in s.bookings
  if (typeof state !== 'undefined' && state && state.currentBooking) {
    const hasCurrent = s.bookings.some(b => String(b.id) === String(state.currentBooking.id));
    if (!hasCurrent) {
      s.bookings.unshift(state.currentBooking);
      changed = true;
    }
  }

  // 4. Sync jobs pool updates
  s.bookings.forEach(b => {
    const job = jobs.find(j =>
      String(j.id) === String(b.id) ||
      String(j.bookingRefId) === String(b.id) ||
      (j.id === 'emp-shift-' + b.id)
    );
    if (job) {
      if (job.contractStatus === 'ended' && b.status !== 'completed') {
        b.status = 'completed';
        changed = true;
      }
      if (job.acceptedCrew && job.acceptedCrew.length > 0 && (!b.acceptedCrew || b.acceptedCrew.length === 0)) {
        b.acceptedCrew = job.acceptedCrew;
        b.workerName = job.acceptedCrew[0];
        if (b.status === 'matching') b.status = 'active';
        changed = true;
      }
      if (job.transitStatus && (b.transitStatus !== job.transitStatus || b.departedAt !== job.departedAt || b.arrivedAt !== job.arrivedAt)) {
        b.transitStatus = job.transitStatus;
        b.departedAt = job.departedAt;
        b.arrivedAt = job.arrivedAt;
        changed = true;
      }
    }
  });

  sessionStorage.setItem(SESSION_KEY, JSON.stringify(s));
}

/* ── Render Bookings List ──────────────────────────────── */

let lastEmployerBookingsSig = null;

function renderBookings(force = false) {
  syncBookingsWithSharedState();
  const s = getSession();
  const bookings = (s && s.bookings) ? s.bookings : [];

  const active = bookings.filter(b => b.status === 'active' || b.status === 'matching' || b.status === 'accepted');
  const previous = bookings.filter(b => b.status === 'completed' || b.status === 'ended');

  // Stats
  const statActiveEl = document.getElementById('statActive');
  const statPreviousEl = document.getElementById('statPrevious');
  if (statActiveEl) statActiveEl.textContent = active.length;
  if (statPreviousEl) statPreviousEl.textContent = previous.length;

  const currentSig = JSON.stringify(bookings.map(b => `${b.id}_${b.status}_${b.transitStatus || ''}_${b.departedAt || ''}_${b.arrivedAt || ''}_${(b.acceptedCrew || []).join(',')}_${b.neededCrew || b.count}`));
  if (!force && currentSig === lastEmployerBookingsSig) {
    return;
  }
  lastEmployerBookingsSig = currentSig;

  // Active list
  if (activeBookingsList) {
    activeBookingsList.innerHTML = '';
    active.forEach(b => activeBookingsList.appendChild(buildBookingItem(b)));
  }
  if (activeEmpty) {
    activeEmpty.style.display = active.length === 0 ? 'flex' : 'none';
  }

  // Previous list
  if (previousBookingsList) {
    previousBookingsList.innerHTML = '';
    previous.forEach(b => previousBookingsList.appendChild(buildBookingItem(b)));
  }
  if (previousEmpty) {
    previousEmpty.style.display = previous.length === 0 ? 'flex' : 'none';
  }
}

function buildBookingItem(b) {
  const item = document.createElement('div');
  const isEnded = b.status === 'completed' || b.status === 'ended';
  const isBookingActive = b.status === 'active' || b.status === 'matching' || b.status === 'accepted';
  item.className = `db-booking-item app-job-card ${isEnded ? 'ended' : ''}`;

  const iconSvg = window.InstantCrewShared ? window.InstantCrewShared.getRoleIcon(b, 22) : '';
  const empType = (b.employmentType || 'full-time').toLowerCase();
  const isPart = empType.includes('part');
  const empClass = isPart ? 'part-time' : 'full-time';
  const empLabel = isPart ? 'PART-TIME' : 'FULL-TIME';

  const crewName = (b.acceptedCrew && b.acceptedCrew.length > 0) ? b.acceptedCrew[0] : (b.workerName || '');

  let statusBadgeHtml = '';
  if (isEnded) {
    statusBadgeHtml = `<span class="app-job-status-badge ended">Ended</span>`;
  } else if (b.status === 'matching') {
    statusBadgeHtml = `<span class="app-job-status-badge matching"><span class="dot"></span>Matching</span>`;
  } else {
    statusBadgeHtml = `<span class="app-job-status-badge active"><span class="dot"></span>ACTIVE</span>`;
  }

  const roleTitle = (b.roleLabel || b.role || 'Crew Member')
    .replace(/\s*\((Full-Time|Part-Time)\)/gi, '')
    .trim()
    .toUpperCase();

  let displayLocation = '';
  if (b.location && !b.location.toLowerCase().includes('current venue')) {
    displayLocation = b.location.trim();
  } else if (b.venue && !b.venue.toLowerCase().includes('current venue')) {
    displayLocation = b.venue.trim();
  } else if (b.location) {
    displayLocation = b.location.replace(/\s*\(Current Venue\)/gi, '').trim();
  } else if (b.venue) {
    displayLocation = b.venue.replace(/\s*\(Current Venue\)/gi, '').trim();
  } else if (b.mapLocation) {
    displayLocation = b.mapLocation.replace(/\s*\(Current Venue\)/gi, '').trim();
  }
  if (!displayLocation) displayLocation = 'Cebu City (Current)';

  // Timing display
  const isAsap = !b.timing || /asap|instant|immediately|now/i.test(b.timing);
  let displayTiming = b.timing;
  if (isAsap) {
    displayTiming = isEnded ? 'Shift Concluded' : 'Contract In Progress';
  }

  const rawRate = b.crewRate || (b.offeredRate ? `₱${b.offeredRate}` : '₱85');
  const displayRate = String(rawRate).replace(/\/hr.*$/i, '').trim();

  item.innerHTML = `
        <div class="app-job-card-header">
            <div class="app-job-card-left">
                <div class="app-job-card-icon">${iconSvg}</div>
                <div class="app-job-card-title-col">
                    <div class="app-job-card-title-row" style="display:inline-flex; align-items:center; flex-wrap:nowrap; gap:8px; white-space:nowrap;">
                        <h4 class="app-job-card-title" style="white-space:nowrap; margin:0;">${roleTitle}</h4>
                        <span class="app-job-emp-badge ${empClass}" style="flex-shrink:0;">${empLabel}</span>
                    </div>
                </div>
            </div>
            ${statusBadgeHtml}
        </div>

        <div class="app-job-card-meta">
            <div class="app-job-meta-line" style="display:flex; align-items:center; gap:6px;">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="flex-shrink:0;">
                    <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"></path>
                    <circle cx="12" cy="10" r="3"></circle>
                </svg>
                <span class="app-meta-venue-text" style="white-space:nowrap; overflow:hidden; text-overflow:ellipsis;">${displayLocation}</span>
            </div>
            <div class="app-job-meta-line" style="display:flex; align-items:center; gap:6px;">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="flex-shrink:0;">
                    <circle cx="12" cy="12" r="10"></circle>
                    <polyline points="12 6 12 12 16 14"></polyline>
                </svg>
                <span>${displayTiming}</span>
            </div>
            <div class="app-job-meta-line" style="color:#0F172A; font-weight:700; display:flex; align-items:center; gap:6px;">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="flex-shrink:0;">
                    <line x1="12" y1="1" x2="12" y2="23"></line>
                    <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"></path>
                </svg>
                <span>Rate: ${displayRate}/hr</span>
            </div>
            ${crewName ? `
            <div class="app-job-meta-line" style="color:#059669; font-weight:600; display:flex; align-items:center; gap:6px;">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="flex-shrink:0;">
                    <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"></path>
                    <circle cx="12" cy="7" r="4"></circle>
                </svg>
                <span>Crew: ${crewName}</span>
            </div>` : ''}
        </div>

        ${!isEnded ? `
        <div class="app-job-card-footer active-contract-footer">
            ${b.transitStatus === 'on_the_way' ? `
            <div class="app-card-transit-box">
                <span class="app-card-transit-status on-the-way">
                    <span class="app-transit-beacon"><span class="app-transit-ping"></span><span class="app-transit-dot"></span></span>
                    <span>En Route (${b.departedAt || 'Just now'})</span>
                </span>
            </div>` : (b.transitStatus === 'arrived' ? `
            <div class="app-card-transit-box">
                <span class="app-card-transit-status arrived">
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="#10B981" stroke-width="2.6" stroke-linecap="round" stroke-linejoin="round">
                        <polyline points="20 6 9 17 4 12"></polyline>
                    </svg>
                    <span>Arrived on Site (${b.arrivedAt || 'Active'})</span>
                </span>
            </div>` : (b.status === 'matching' ? `
            <div class="app-card-transit-box">
                <span class="app-card-transit-status pending">
                    <span>Looking for Nearby Crew</span>
                </span>
            </div>` : `
            <div class="app-card-transit-box">
                <span class="app-card-transit-status pending">
                    <span>Crew Preparing &bull; Awaiting Transit</span>
                </span>
            </div>`))}
            <button type="button" class="app-btn-end-contract-clean db-btn-end-contract" data-booking-id="${b.id}">
                End Job
            </button>
        </div>` : `
        <div class="app-job-card-footer ended-footer">
            <span class="app-ended-note">
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="12" cy="12" r="10"></circle>
                    <polyline points="12 6 12 12 16 14"></polyline>
                </svg>
                <span>${b.endedAt ? `Concluded ${b.endedAt}` : (b.bookedAt ? `Concluded ${b.bookedAt}` : 'Concluded contract')}</span>
            </span>
            <span class="app-job-archived-tag">Archived</span>
        </div>`}
    `;

  const viewDetailBtn = item.querySelector('.app-job-view-detail-btn');
  if (viewDetailBtn) {
    viewDetailBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      openLocationDetailModal({
        ...b,
        title: roleTitle,
        venue: venueName,
        mapLocation: locationAddress,
        employmentType: empLabel,
        isAccepted: !isEnded
      });
    });
  }

  const endBtn = item.querySelector('.db-btn-end-contract');
  if (endBtn) {
    endBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      if (confirm(`Conclude and end the contract for ${roleTitle}?`)) {
        endEmployerBooking(b.id);
      }
    });
  }

  return item;
}

/* ════ Location Detail Modal Handlers for Employer ════ */
function openLocationDetailModal(shift) {
  const appLocationDetailModal = document.getElementById('appLocationDetailModal');
  if (!appLocationDetailModal) {
    const address = shift.mapLocation || shift.location || 'Cebu City';
    const mapsUrl = shift.googleMapsUrl || (window.InstantCrewShared ? window.InstantCrewShared.formatGoogleMapsUrl(address) : `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(address)}`);
    window.open(mapsUrl, '_blank', 'noopener,noreferrer');
    return;
  }

  const venue = shift.venue || shift.employerName || shift.location || 'Employer Shift';
  const cleanRole = (shift.title || shift.role || 'Crew').replace(/\s*\((Full-Time|Part-Time)\)/gi, '').trim();
  const roleUpper = cleanRole.toUpperCase();

  const isPart = (shift.employmentType && shift.employmentType.toLowerCase().includes('part')) ||
    (shift.title && shift.title.toLowerCase().includes('part')) ||
    (shift.role && shift.role.toLowerCase().includes('part'));
  const empUpper = isPart ? 'PART-TIME' : 'FULL-TIME';

  const cardBadgesHtml = `
        <span class="app-loc-badge-role">${roleUpper}</span>
        <span class="app-loc-badge-emp ${isPart ? 'part-time' : 'full-time'}">${empUpper}</span>
    `;

  const address = shift.mapLocation || shift.location || 'Cebu City';
  const mapsUrl = shift.googleMapsUrl || (window.InstantCrewShared ? window.InstantCrewShared.formatGoogleMapsUrl(address) : `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(address)}`);

  const locModalVenueTitle = document.getElementById('locModalVenueTitle');
  const locModalVenueName = document.getElementById('locModalVenueName');
  const locModalRoleSub = document.getElementById('locModalRoleSub');
  const locModalCardBadges = document.getElementById('locModalCardBadges');
  const locModalAddress = document.getElementById('locModalAddress');
  const openGoogleMapsActionBtn = document.getElementById('openGoogleMapsActionBtn');
  const copyLocAddressBtn = document.getElementById('copyLocAddressBtn');

  if (locModalVenueTitle) locModalVenueTitle.textContent = `${venue} Location`;
  if (locModalVenueName) locModalVenueName.textContent = venue;
  if (locModalRoleSub) locModalRoleSub.textContent = 'Verified Work Location';
  if (locModalCardBadges) locModalCardBadges.innerHTML = cardBadgesHtml;
  if (locModalAddress) locModalAddress.textContent = address;
  if (openGoogleMapsActionBtn) openGoogleMapsActionBtn.href = mapsUrl;

  if (copyLocAddressBtn) {
    copyLocAddressBtn.onclick = (e) => {
      e.stopPropagation();
      navigator.clipboard.writeText(address).then(() => {
        if (window.InstantCrewShared) window.InstantCrewShared.showPushToast('Copied', 'Location copied to clipboard', 'info');
        else alert('Location copied to clipboard!');
      }).catch(() => {
        prompt('Copy address:', address);
      });
    };
  }

  appLocationDetailModal.style.display = 'flex';
  appLocationDetailModal.classList.add('open');
  appLocationDetailModal.setAttribute('aria-hidden', 'false');
  document.body.style.overflow = 'hidden';
}

function closeLocationDetailModal() {
  const appLocationDetailModal = document.getElementById('appLocationDetailModal');
  if (!appLocationDetailModal) return;
  appLocationDetailModal.classList.remove('open');
  appLocationDetailModal.style.display = 'none';
  appLocationDetailModal.setAttribute('aria-hidden', 'true');
  const profileModal = document.getElementById('profileModal');
  if (profileModal && !profileModal.hidden) {
    document.body.style.overflow = 'hidden';
  } else {
    document.body.style.overflow = '';
  }
}

const closeLocModalBtn = document.getElementById('closeLocModalBtn');
if (closeLocModalBtn) {
  closeLocModalBtn.addEventListener('click', (e) => {
    e.stopPropagation();
    closeLocationDetailModal();
  });
}
const appLocationDetailModalEl = document.getElementById('appLocationDetailModal');
if (appLocationDetailModalEl) {
  appLocationDetailModalEl.addEventListener('click', (e) => {
    if (e.target === appLocationDetailModalEl) {
      closeLocationDetailModal();
    }
  });
}
document.addEventListener('keydown', (e) => {
  if (e.key === 'Escape') {
    const modal = document.getElementById('appLocationDetailModal');
    if (modal && modal.classList.contains('open')) {
      closeLocationDetailModal();
    }
  }
});

/* End booking / contract from employer side */
function endEmployerBooking(bookingId) {
  const s = getSession();
  if (!s || !s.bookings) return;
  const booking = s.bookings.find(b => String(b.id) === String(bookingId) || (b.contractId && String(b.contractId) === String(bookingId)));
  if (booking) {
    booking.status = 'completed';
    booking.endedBy = 'employer';
    sessionStorage.setItem(SESSION_KEY, JSON.stringify(s));
  }
  if (window.InstantCrewShared) {
    window.InstantCrewShared.endContract(bookingId, 'employer', s ? s.name : 'Employer');
    window.InstantCrewShared.showPushToast('Contract Ended', `Contract for ${booking ? (booking.roleLabel || booking.role) : 'crew'} concluded.`, 'end');
  }
  renderBookings(true);
}

/* ── Save a booking to session when confirmed ─────────── */
function saveBookingToSession(bookingData) {
  const s = getSession();
  if (!s) return;
  if (!s.bookings) s.bookings = [];

  // Retain all existing active bookings until explicitly ended!
  // Check if booking already exists in array (update it)
  const existingIndex = s.bookings.findIndex(b => String(b.id) === String(bookingData.id));
  if (existingIndex >= 0) {
    s.bookings[existingIndex] = { ...s.bookings[existingIndex], ...bookingData };
  } else {
    s.bookings.unshift(bookingData);
  }
  sessionStorage.setItem(SESSION_KEY, JSON.stringify(s));
}



/* ── Empty State "Find Crew" redirects to Step 1 ─────── */

const searchFromActiveEmptyBtn = document.getElementById('searchFromActiveEmpty');
const searchFromPreviousEmptyBtn = document.getElementById('searchFromPreviousEmpty');

function goFindCrew() {
  closeProfilePanel();
  goStep(1);
}

if (searchFromActiveEmptyBtn) searchFromActiveEmptyBtn.addEventListener('click', goFindCrew);
if (searchFromPreviousEmptyBtn) searchFromPreviousEmptyBtn.addEventListener('click', goFindCrew);

/* ── Logout ────────────────────────────────────────────── */

const headerLogoLink = document.getElementById('headerLogoLink');
if (headerLogoLink) {
  headerLogoLink.addEventListener('click', (e) => {
    if (!getSession()) {
      // Not logged in: go back to landing page!
      window.location.href = 'index.html';
      return;
    }
    e.preventDefault();
    if (typeof closeProfilePanel === 'function') closeProfilePanel();
    if (typeof goStep === 'function') goStep(1);
    window.scrollTo({ top: 0, behavior: 'smooth' });
  });
}

const headerBackArrow = document.getElementById('headerBackArrow');
if (headerBackArrow) {
  headerBackArrow.addEventListener('click', (e) => {
    window.location.href = 'index.html';
  });
}

if (logoutBtn) {
  logoutBtn.addEventListener('click', () => {
    fetch('/api/auth/logout', { method: 'POST', credentials: 'same-origin', keepalive: true });
    sessionStorage.removeItem(SESSION_KEY);
    window.location.href = 'index.html';
  });
}

/* ═══════════════════════════════════════════════════════════
   EMPLOYER NOTIFICATIONS SYSTEM
   ═══════════════════════════════════════════════════════════ */

const notifBtn = document.getElementById('notifBtn');
const notifBadge = document.getElementById('notifBadge');
const notifDropdown = document.getElementById('notifDropdown');
const notifList = document.getElementById('notifList');
const clearNotifsBtn = document.getElementById('clearNotifsBtn');

let lastEmployerNotifCount = 0;

function renderEmployerNotifications() {
  if (!window.InstantCrewShared) return;
  const notifs = window.InstantCrewShared.getNotifications('employer');
  const unread = notifs.filter(n => !n.read).length;

  if (notifBadge) {
    if (unread > 0) {
      notifBadge.textContent = unread > 9 ? '9+' : unread;
      notifBadge.style.display = 'flex';
    } else {
      notifBadge.style.display = 'none';
    }
  }

  if (notifList) {
    if (notifs.length === 0) {
      notifList.innerHTML = '<div class="db-notif-empty">No notifications yet.</div>';
    } else {
      notifList.innerHTML = '';
      notifs.forEach(n => {
        const el = document.createElement('div');
        el.className = `db-notif-item ${n.read ? '' : 'unread'}`;
        const iconSvg = window.InstantCrewShared ? window.InstantCrewShared.getNotificationIcon(n.type, 20) : '';
        el.innerHTML = `
                    <span class="db-notif-icon">${iconSvg}</span>
                    <div>
                        <p class="db-notif-title">${n.title}</p>
                        <p class="db-notif-desc">${n.message}</p>
                        <span class="db-notif-time">${n.time}</span>
                    </div>
                `;
        notifList.appendChild(el);
      });
    }
  }

  // Trigger push toast on new notification
  if (notifs.length > lastEmployerNotifCount && lastEmployerNotifCount > 0) {
    const latest = notifs[0];
    window.InstantCrewShared.showPushToast(latest.title, latest.message, latest.type);
  }
  lastEmployerNotifCount = notifs.length;
}

if (notifBtn) {
  notifBtn.addEventListener('click', (e) => {
    e.stopPropagation();
    const isHidden = notifDropdown.hidden;
    notifDropdown.hidden = !isHidden;
    if (isHidden && window.InstantCrewShared) {
      window.InstantCrewShared.markAllNotificationsRead('employer');
      if (notifBadge) notifBadge.style.display = 'none';
    }
  });
}

if (clearNotifsBtn) {
  clearNotifsBtn.addEventListener('click', () => {
    if (window.InstantCrewShared) {
      window.InstantCrewShared.clearNotifications('employer');
      renderEmployerNotifications();
    }
  });
}

document.addEventListener('click', (e) => {
  if (notifDropdown && !notifDropdown.hidden && !notifDropdown.contains(e.target) && e.target !== notifBtn) {
    notifDropdown.hidden = true;
  }
});

function updateStep5TransitLiveStatus() {
  const box = document.getElementById('step5TransitBox');
  if (!box) return;
  const badgeEl = document.getElementById('step5TransitBadge');
  const timeEl = document.getElementById('step5TransitTime');
  const bodyEl = document.getElementById('step5TransitBody');
  if (!badgeEl || !timeEl || !bodyEl) return;

  const s = getSession();
  let currentB = null;
  if (s && s.bookings) {
    currentB = (state.currentBooking && s.bookings.find(b => String(b.id) === String(state.currentBooking.id))) ||
      s.bookings.find(b => b.status === 'active');
  }
  const contracts = (window.InstantCrewShared ? window.InstantCrewShared.getActiveContracts() : []);
  const activeContract = contracts.find(c =>
    (currentB && (String(c.bookingRefId) === String(currentB.id) || String(c.id) === String(currentB.id) || String(c.jobId) === String(currentB.id) || c.jobId === 'emp-shift-' + currentB.id)) ||
    c.status === 'active'
  );

  const transitStatus = (currentB && currentB.transitStatus) || (activeContract && activeContract.transitStatus) || null;
  const departedAt = (currentB && currentB.departedAt) || (activeContract && activeContract.departedAt) || null;
  const arrivedAt = (currentB && currentB.arrivedAt) || (activeContract && activeContract.arrivedAt) || null;
  const crewName = (currentB && currentB.workerName) || (currentB && currentB.acceptedCrew && currentB.acceptedCrew[0]) || (activeContract && activeContract.workerName) || 'Angelo Lopez';
  const venueName = (currentB && currentB.location) || (activeContract && activeContract.venue) || state.location || 'your location';

  box.style.display = 'block';

  const dTimeEl = document.getElementById('dTime');
  if (dTimeEl && (/asap|now/i.test(dTimeEl.textContent) || dTimeEl.textContent === 'As soon as possible')) {
    dTimeEl.textContent = 'Immediate Shift (Accepted)';
  }

  if (transitStatus === 'on_the_way') {
    box.className = 'db-live-transit-box on-the-way';
    badgeEl.className = 'db-live-transit-badge on-the-way';
    badgeEl.innerHTML = `<span class="db-transit-beacon"><span class="db-transit-ping"></span><span class="db-transit-dot"></span></span><span>Crew is On the Way!</span>`;
    timeEl.textContent = departedAt ? ` ${departedAt}` : 'Departed: Just now';
    bodyEl.innerHTML = `<strong>${crewName}</strong> has notified you that they have departed and are en route to <strong>${venueName}</strong>.`;
  } else if (transitStatus === 'arrived') {
    box.className = 'db-live-transit-box arrived';
    badgeEl.className = 'db-live-transit-badge arrived';
    badgeEl.innerHTML = `<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg><span>Crew Has Arrived</span>`;
    timeEl.textContent = arrivedAt ? `Arrived: ${arrivedAt}` : 'Arrived: Just now';
    bodyEl.innerHTML = `<strong>${crewName}</strong> has safely arrived at <strong>${venueName}</strong> and is ready to work!`;
  } else {
    box.className = 'db-live-transit-box pending';
    badgeEl.className = 'db-live-transit-badge pending';
    badgeEl.innerHTML = `<span>Preparing for Shift</span>`;
    timeEl.textContent = 'Awaiting Departure';
    bodyEl.innerHTML = `<strong>${crewName}</strong> has accepted your booking and will notify you when heading to <strong>${venueName}</strong>.`;
  }
}

function syncEmployerState() {
  renderEmployerNotifications();
  renderBookings();
  checkCrewAcceptance();
  updateStep5TransitLiveStatus();
}

window.addEventListener('ic_state_change', syncEmployerState);
window.addEventListener('storage', syncEmployerState);
window.addEventListener('focus', syncEmployerState);
document.addEventListener('visibilitychange', () => {
  if (document.visibilityState === 'visible') syncEmployerState();
});
setInterval(syncEmployerState, 800);

/* ── Run on load ────────────────────────────────────────── */
populateProfile();
renderEmployerNotifications();
renderBookings();
updateStep5TransitLiveStatus();
