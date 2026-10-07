importScripts("https://www.gstatic.com/firebasejs/12.19.0/firebase-app-compat.js");
importScripts("https://www.gstatic.com/firebasejs/12.19.0/firebase-messaging-compat.js");
importScripts("/config.js");

firebase.initializeApp(self.GUNFISH_FCM_TEST.firebaseConfig);
const messaging = firebase.messaging();

// Called when the page is not in the foreground.
// Messages with a notification payload are displayed by the SDK automatically.
messaging.onBackgroundMessage(async (payload) => {
  const clients = await self.clients.matchAll({ type: "window", includeUncontrolled: true });
  for (const client of clients) {
    client.postMessage({ type: "fcm-background", payload });
  }
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  event.waitUntil(
    self.clients.matchAll({ type: "window", includeUncontrolled: true }).then((clients) => {
      if (clients.length > 0) {
        return clients[0].focus();
      }
      return self.clients.openWindow("/");
    }),
  );
});
