// Helper untuk konversi VAPID public key base64 URL-safe ke Uint8Array
function urlBase64ToUint8Array(base64String) {
  const padding = '='.repeat((4 - base64String.length % 4) % 4);
  const base64 = (base64String + padding)
    .replace(/\-/g, '+')
    .replace(/_/g, '/');

  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);

  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

document.addEventListener('DOMContentLoaded', async () => {
  const btnEnable = document.getElementById('btn-enable-push');
  const btnTest = document.getElementById('btn-test-push');
  const statusEl = document.getElementById('push-status');

  if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
    statusEl.innerText = 'Browser/perangkat ini belum mendukung Web Push.';
    btnEnable.disabled = true;
    return;
  }

  // Cek apakah sudah berlangganan
  const reg = await navigator.serviceWorker.ready;
  const existingSub = await reg.pushManager.getSubscription();

  if (existingSub) {
    statusEl.innerText = '✅ Notifikasi aktif di perangkat ini!';
    btnEnable.style.display = 'none';
    btnTest.style.display = 'block';
  }

  btnEnable.addEventListener('click', async () => {
    btnEnable.disabled = true;
    statusEl.innerText = 'Meminta izin notifikasi...';

    try {
      const permission = await Notification.requestPermission();
      if (permission !== 'granted') {
        statusEl.innerText = '❌ Izin notifikasi ditolak oleh pengguna.';
        btnEnable.disabled = false;
        return;
      }

      // Ambil VAPID Public Key dari server
      const keyRes = await fetch('/api/push/vapid-public-key');
      const { publicKey } = await keyRes.json();

      if (!publicKey) {
        statusEl.innerText = '⚠️ Server belum memiliki VAPID_PUBLIC_KEY.';
        btnEnable.disabled = false;
        return;
      }

      // Daftarkan push subscription ke browser
      const sub = await reg.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(publicKey)
      });

      // Kirim data subscription ke server Go
      const saveRes = await fetch('/api/push/subscribe', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(sub)
      });

      if (saveRes.ok) {
        statusEl.innerText = '🎉 Berhasil mengaktifkan notifikasi!';
        btnEnable.style.display = 'none';
        btnTest.style.display = 'block';
      } else {
        statusEl.innerText = 'Gagal menyimpan subscription ke server.';
        btnEnable.disabled = false;
      }
    } catch (err) {
      statusEl.innerText = 'Error: ' + err.message;
      btnEnable.disabled = false;
    }
  });

  btnTest.addEventListener('click', async () => {
    btnTest.disabled = true;
    statusEl.innerText = 'Mengirim notifikasi tes...';

    try {
      const res = await fetch('/api/push/test', { method: 'POST' });
      const data = await res.json();
      if (res.ok) {
        statusEl.innerText = `✅ Notifikasi terkirim ke ${data.sentTo} perangkat!`;
      } else {
        statusEl.innerText = 'Gagal kirim tes: ' + (data.error || 'Unknown error');
      }
    } catch (err) {
      statusEl.innerText = 'Gagal kirim tes: ' + err.message;
    } finally {
      btnTest.disabled = false;
    }
  });
});
