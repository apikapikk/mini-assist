// Register Service Worker
if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/sw.js')
      .then((reg) => console.log('[PWA] Service Worker registered:', reg.scope))
      .catch((err) => console.warn('[PWA] Service Worker registration failed:', err));
  });
}

// Inisialisasi Aplikasi
document.addEventListener('DOMContentLoaded', () => {
  loadTodayData();

  const addForm = document.getElementById('quick-add-form');
  if (addForm) {
    addForm.addEventListener('submit', (e) => {
      e.preventDefault();
      const input = document.getElementById('quick-add-input');
      const text = input.value.trim();
      if (!text) return;

      createReminder(text);
      input.value = '';
    });
  }
});

async function loadTodayData() {
  try {
    const res = await fetch('/api/reminders?status=pending');
    if (res.status === 401) {
      window.location.href = '/login';
      return;
    }
    const reminders = await res.json();
    renderReminders(reminders);
  } catch (err) {
    console.error('Gagal memuat data:', err);
  }
}

function renderReminders(reminders) {
  const container = document.getElementById('reminders-list');
  if (!container) return;

  if (!reminders || reminders.length === 0) {
    container.innerHTML = '<div class="empty-state">Belum ada reminder untuk sekarang 🎉</div>';
    return;
  }

  container.innerHTML = reminders.map((r) => {
    const due = new Date(r.DueAtUTC).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    return `
      <div class="card" id="reminder-${r.ID}">
        <div class="card-content">
          <div class="card-title">${escapeHtml(r.Text)}</div>
          <div class="card-time">⏰ ${due}</div>
        </div>
        <button class="card-action" onclick="markDone(${r.ID})" title="Selesai">
          ✓
        </button>
      </div>
    `;
  }).join('');
}

async function markDone(id) {
  try {
    const res = await fetch(`/api/reminders/${id}/done`, { method: 'POST' });
    if (res.ok) {
      const el = document.getElementById(`reminder-${id}`);
      if (el) el.remove();
      loadTodayData();
    }
  } catch (err) {
    console.error('Gagal menyelesaikan reminder:', err);
  }
}

async function createReminder(text) {
  try {
    const res = await fetch('/api/reminders', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ text: text })
    });
    if (res.ok) {
      loadTodayData();
    }
  } catch (err) {
    console.error('Gagal menambah reminder:', err);
  }
}

function escapeHtml(str) {
  return str.replace(/[&<>'"]/g, 
    tag => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[tag] || tag)
  );
}
