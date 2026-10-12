// Register Service Worker
if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js')
      .then((reg) => console.log('[PWA] Service Worker registered:', reg.scope))
      .catch((err) => console.warn('[PWA] Service Worker registration failed:', err));
  });
}

let currentTab = 'pending'; // 'pending' | 'done'
let remindersCache = [];

// Inisialisasi Aplikasi
document.addEventListener('DOMContentLoaded', () => {
  setupTabs();
  setupModal();
  setupQuickAdd();
  loadData();
});

function setupTabs() {
  const tabPending = document.getElementById('tab-pending');
  const tabDone = document.getElementById('tab-done');

  if (tabPending && tabDone) {
    tabPending.addEventListener('click', () => {
      currentTab = 'pending';
      tabPending.classList.add('active');
      tabDone.classList.remove('active');
      loadData();
    });

    tabDone.addEventListener('click', () => {
      currentTab = 'done';
      tabDone.classList.add('active');
      tabPending.classList.remove('active');
      loadData();
    });
  }
}

function setupModal() {
  const modal = document.getElementById('task-modal');
  const btnOpen = document.getElementById('btn-open-modal');
  const btnClose = document.getElementById('btn-close-modal');
  const btnCancel = document.getElementById('btn-cancel-modal');
  const form = document.getElementById('task-detail-form');
  const repeatSelect = document.getElementById('task-repeat');
  const customDaysGroup = document.getElementById('custom-days-group');
  const intervalGroup = document.getElementById('interval-group');

  if (!modal) return;

  // Toggle dropdown pengulangan
  if (repeatSelect) {
    repeatSelect.addEventListener('change', () => {
      const val = repeatSelect.value;
      customDaysGroup.style.display = val === 'custom_days' ? 'block' : 'none';
      intervalGroup.style.display = val === 'interval' ? 'block' : 'none';
    });
  }

  // Buka modal tambah baru
  if (btnOpen) {
    btnOpen.addEventListener('click', () => {
      openModalForCreate();
    });
  }

  // Tutup modal
  const closeModal = () => {
    modal.style.display = 'none';
  };
  if (btnClose) btnClose.addEventListener('click', closeModal);
  if (btnCancel) btnCancel.addEventListener('click', closeModal);

  // Submit form modal
  if (form) {
    form.addEventListener('submit', async (e) => {
      e.preventDefault();
      await handleSaveTask();
      closeModal();
    });
  }
}

function openModalForCreate() {
  const modal = document.getElementById('task-modal');
  const title = document.getElementById('modal-title');
  const editId = document.getElementById('task-edit-id');
  const textInput = document.getElementById('task-text');
  const dueInput = document.getElementById('task-due');
  const repeatSelect = document.getElementById('task-repeat');
  const customDaysGroup = document.getElementById('custom-days-group');
  const intervalGroup = document.getElementById('interval-group');
  const quickInput = document.getElementById('quick-add-input');

  title.innerText = 'Tambah Tugas / Pengingat';
  editId.value = '';
  textInput.value = quickInput ? quickInput.value.trim() : '';

  // Default waktu: 1 jam dari sekarang
  const now = new Date();
  now.setHours(now.getHours() + 1);
  now.setMinutes(0);
  dueInput.value = formatDateTimeLocal(now);

  repeatSelect.value = 'none';
  customDaysGroup.style.display = 'none';
  intervalGroup.style.display = 'none';

  // Reset checkboxes
  document.querySelectorAll('input[name="repeat-day"]').forEach(cb => cb.checked = false);

  modal.style.display = 'flex';
}

function openModalForEdit(id) {
  const item = remindersCache.find(r => r.ID === id);
  if (!item) return;

  const modal = document.getElementById('task-modal');
  const title = document.getElementById('modal-title');
  const editId = document.getElementById('task-edit-id');
  const textInput = document.getElementById('task-text');
  const dueInput = document.getElementById('task-due');
  const repeatSelect = document.getElementById('task-repeat');
  const customDaysGroup = document.getElementById('custom-days-group');
  const intervalGroup = document.getElementById('interval-group');
  const intervalInput = document.getElementById('task-interval-days');

  title.innerText = 'Edit Tugas';
  editId.value = item.ID;
  textInput.value = item.Text;

  const dueDate = new Date(item.DueAtUTC);
  dueInput.value = formatDateTimeLocal(dueDate);

  // Set repeat rule
  const rule = item.RepeatRule || 'none';
  document.querySelectorAll('input[name="repeat-day"]').forEach(cb => cb.checked = false);
  customDaysGroup.style.display = 'none';
  intervalGroup.style.display = 'none';

  if (rule === 'daily' || rule === 'weekdays' || rule === 'weekly') {
    repeatSelect.value = rule;
  } else if (rule.startsWith('days:')) {
    repeatSelect.value = 'custom_days';
    customDaysGroup.style.display = 'block';
    const days = rule.replace('days:', '').split(',');
    days.forEach(d => {
      const cb = document.querySelector(`input[name="repeat-day"][value="${d.trim()}"]`);
      if (cb) cb.checked = true;
    });
  } else if (rule.startsWith('interval:')) {
    repeatSelect.value = 'interval';
    intervalGroup.style.display = 'block';
    intervalInput.value = rule.replace('interval:', '').trim();
  } else {
    repeatSelect.value = 'none';
  }

  modal.style.display = 'flex';
}

async function handleSaveTask() {
  const editId = document.getElementById('task-edit-id').value;
  const text = document.getElementById('task-text').value.trim();
  const dueVal = document.getElementById('task-due').value;
  const repeatVal = document.getElementById('task-repeat').value;
  let finalRule = '';

  if (repeatVal === 'daily' || repeatVal === 'weekdays' || repeatVal === 'weekly') {
    finalRule = repeatVal;
  } else if (repeatVal === 'custom_days') {
    const selectedDays = [];
    document.querySelectorAll('input[name="repeat-day"]:checked').forEach(cb => {
      selectedDays.push(cb.value);
    });
    if (selectedDays.length > 0) {
      finalRule = `days:${selectedDays.join(',')}`;
    }
  } else if (repeatVal === 'interval') {
    const n = document.getElementById('task-interval-days').value || 2;
    finalRule = `interval:${n}`;
  }

  let dueIso = '';
  if (dueVal) {
    dueIso = new Date(dueVal).toISOString();
  }

  if (editId) {
    // Edit existing
    await fetch(`/api/reminders/${editId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        text: text,
        due_at: dueIso,
        repeat_rule: finalRule
      })
    });
  } else {
    // Create new
    await fetch('/api/reminders', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        text: text,
        due_at: dueIso,
        repeat_rule: finalRule
      })
    });
  }

  const quickInput = document.getElementById('quick-add-input');
  if (quickInput) quickInput.value = '';
  loadData();
}

function setupQuickAdd() {
  const addForm = document.getElementById('quick-add-form');
  if (addForm) {
    addForm.addEventListener('submit', async (e) => {
      e.preventDefault();
      const input = document.getElementById('quick-add-input');
      const text = input.value.trim();
      if (!text) return;

      await createSimpleReminder(text);
      input.value = '';
    });
  }
}

async function loadData() {
  try {
    const res = await fetch(`/api/reminders?status=${currentTab}`);
    if (res.status === 401) {
      window.location.href = '/login';
      return;
    }
    const reminders = await res.json();
    remindersCache = reminders || [];
    renderReminders(remindersCache);
  } catch (err) {
    console.error('Gagal memuat data:', err);
  }
}

function renderReminders(reminders) {
  const container = document.getElementById('reminders-list');
  if (!container) return;

  if (!reminders || reminders.length === 0) {
    container.innerHTML = `<div class="empty-state">${
      currentTab === 'pending' ? 'Belum ada tugas aktif 🎉' : 'Belum ada tugas yang selesai.'
    }</div>`;
    return;
  }

  container.innerHTML = reminders.map((r) => {
    const dueDate = new Date(r.DueAtUTC);
    const dateFormatted = formatDueBadge(dueDate);
    const repeatLabel = formatRepeatBadge(r.RepeatRule);
    const isDone = r.Status === 'done';

    return `
      <div class="card" id="reminder-${r.ID}">
        <div class="card-content" onclick="openModalForEdit(${r.ID})" style="cursor: pointer; flex: 1;">
          <div class="card-title ${isDone ? 'done-text' : ''}">${escapeHtml(r.Text)}</div>
          <div style="display: flex; gap: 6px; align-items: center; margin-top: 2px;">
            <div class="card-time">⏰ ${dateFormatted}</div>
            ${repeatLabel ? `<span class="repeat-badge">🔁 ${repeatLabel}</span>` : ''}
          </div>
        </div>
        <div class="card-actions-wrapper">
          <button class="card-action ${isDone ? 'done' : ''}" onclick="toggleStatus(${r.ID}, '${r.Status}')" title="${isDone ? 'Batal Selesai' : 'Selesai'}">
            ✓
          </button>
          <button class="card-action-del" onclick="deleteTask(${r.ID})" title="Hapus">
            ✕
          </button>
        </div>
      </div>
    `;
  }).join('');
}

async function toggleStatus(id, currentStatus) {
  try {
    const endpoint = currentStatus === 'done' ? `/api/reminders/${id}/undone` : `/api/reminders/${id}/done`;
    const res = await fetch(endpoint, { method: 'POST' });
    if (res.ok) {
      loadData();
    }
  } catch (err) {
    console.error('Gagal mengubah status tugas:', err);
  }
}

async function deleteTask(id) {
  if (!confirm('Hapus tugas ini?')) return;
  try {
    const res = await fetch(`/api/reminders/${id}`, { method: 'DELETE' });
    if (res.ok) {
      loadData();
    }
  } catch (err) {
    console.error('Gagal menghapus tugas:', err);
  }
}

async function createSimpleReminder(text) {
  try {
    const res = await fetch('/api/reminders', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text: text })
    });
    if (res.ok) {
      loadData();
    }
  } catch (err) {
    console.error('Gagal menambah reminder:', err);
  }
}

// Helpers Waktu & Format
function formatDateTimeLocal(d) {
  const pad = (n) => String(n).padStart(2, '0');
  const year = d.getFullYear();
  const month = pad(d.getMonth() + 1);
  const day = pad(d.getDate());
  const hours = pad(d.getHours());
  const minutes = pad(d.getMinutes());
  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

function formatDueBadge(date) {
  const now = new Date();
  const isToday = date.toDateString() === now.toDateString();

  const tomorrow = new Date();
  tomorrow.setDate(tomorrow.getDate() + 1);
  const isTomorrow = date.toDateString() === tomorrow.toDateString();

  const timeStr = date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });

  if (isToday) return `Hari ini ${timeStr}`;
  if (isTomorrow) return `Besok ${timeStr}`;

  const dayStr = date.toLocaleDateString('id-ID', { weekday: 'short', day: 'numeric', month: 'short' });
  return `${dayStr} ${timeStr}`;
}

function formatRepeatBadge(rule) {
  if (!rule || rule === 'none') return '';
  if (rule === 'daily') return 'Tiap Hari';
  if (rule === 'weekdays') return 'Sen-Jum';
  if (rule === 'weekly') return '1x / Minggu';
  if (rule.startsWith('days:')) {
    const dayMap = { '1': 'Sen', '2': 'Sel', '3': 'Rab', '4': 'Kam', '5': 'Jum', '6': 'Sab', '7': 'Min' };
    const days = rule.replace('days:', '').split(',').map(d => dayMap[d.trim()] || d);
    return days.join(', ');
  }
  if (rule.startsWith('interval:')) {
    const n = rule.replace('interval:', '').trim();
    return `Tiap ${n} hari`;
  }
  return rule;
}

function escapeHtml(str) {
  if (!str) return '';
  return str.replace(/[&<>'"]/g, 
    tag => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[tag] || tag)
  );
}
