import { initializeApp } from "https://www.gstatic.com/firebasejs/12.19.0/firebase-app.js";
import { getMessaging, getToken, onMessage } from "https://www.gstatic.com/firebasejs/12.19.0/firebase-messaging.js";

const { firebaseConfig, vapidKey } = self.GUNFISH_FCM_TEST;
const app = initializeApp(firebaseConfig);
const messaging = getMessaging(app);

const $ = (id) => document.getElementById(id);

function setStatus(id, text, cls) {
  const el = $(id);
  el.textContent = text;
  el.className = "status" + (cls ? " " + cls : "");
}

function defaultPayload(token) {
  return JSON.stringify({
    message: {
      token,
      notification: {
        title: "Gunfish FCM test",
        body: "Sent at " + new Date().toLocaleTimeString(),
      },
      data: { source: "gunfish-fcm-test" },
      webpush: {
        fcm_options: { link: location.origin + "/" },
      },
    },
  }, null, 2);
}

function addLog(kind, payload) {
  const log = $("log");
  log.querySelector(".empty")?.remove();
  const entry = document.createElement("div");
  entry.className = "entry";
  const meta = document.createElement("div");
  meta.className = "meta";
  meta.textContent = `${new Date().toLocaleTimeString()} · ${kind}`;
  const pre = document.createElement("pre");
  pre.textContent = JSON.stringify(payload, null, 2);
  entry.append(meta, pre);
  log.prepend(entry);
}

$("enable").addEventListener("click", async () => {
  try {
    setStatus("enable-status", "Requesting permission...");
    const permission = await Notification.requestPermission();
    if (permission !== "granted") {
      setStatus("enable-status", `Notification permission is ${permission}.`, "err");
      return;
    }
    const registration = await navigator.serviceWorker.register("/firebase-messaging-sw.js");
    const token = await getToken(messaging, { vapidKey, serviceWorkerRegistration: registration });
    $("token").textContent = token;
    $("token-row").hidden = false;
    $("payload").value = defaultPayload(token);
    $("send").disabled = false;
    setStatus("enable-status", "Ready.", "ok");
    console.log("FCM registration token:", token);
  } catch (err) {
    console.error(err);
    setStatus("enable-status", String(err), "err");
  }
});

$("copy-token").addEventListener("click", async () => {
  await navigator.clipboard.writeText($("token").textContent);
  $("copy-token").textContent = "Copied";
  setTimeout(() => ($("copy-token").textContent = "Copy"), 1500);
});

$("send").addEventListener("click", async () => {
  try {
    JSON.parse($("payload").value);
  } catch (err) {
    setStatus("send-status", "Invalid JSON: " + err.message, "err");
    return;
  }
  setStatus("send-status", "Sending...");
  try {
    const res = await fetch("/send", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: $("payload").value,
    });
    const text = await res.text();
    setStatus("send-status", `Gunfish responded ${res.status}: ${text.trim()}`, res.ok ? "ok" : "err");
  } catch (err) {
    setStatus("send-status", String(err), "err");
  }
});

// Foreground messages. The browser does not show notifications for them automatically.
onMessage(messaging, async (payload) => {
  addLog("foreground", payload);
  if (payload.notification) {
    const registration = await navigator.serviceWorker.getRegistration("/");
    registration?.showNotification(payload.notification.title ?? "", {
      body: payload.notification.body,
      icon: payload.notification.image,
    });
  }
});

// Background messages forwarded from the service worker.
navigator.serviceWorker.addEventListener("message", (event) => {
  if (event.data?.type === "fcm-background") {
    addLog("background", event.data.payload);
  }
});
