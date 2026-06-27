self.addEventListener('push', (event) => {
	let payload = {};

	if (event.data) {
		const text = event.data.text();
		try {
			payload = JSON.parse(text);
		} catch {
			payload = { body: text };
		}
	}
	const extraData = payload.data && typeof payload.data === 'object' ? payload.data : {};

	const title = payload.title || 'OpenPlays';
	const options = {
		body: payload.body || '',
		tag: payload.tag,
		icon: '/icons/icon-192.png',
		badge: '/icons/icon-192.png',
		data: {
			url: payload.url || '/',
			kind: payload.kind,
			play_id: payload.play_id,
			...extraData
		}
	};

	event.waitUntil(
		Promise.all([
			self.registration.showNotification(title, options),
			self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clients) => {
				for (const client of clients) {
					client.postMessage({ type: 'openplays:notification-received', notification: payload });
				}
			})
		])
	);
});

self.addEventListener('notificationclick', (event) => {
	event.notification.close();

	const targetURL = new URL(event.notification.data?.url || '/', self.location.origin).href;

	event.waitUntil(
		self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clients) => {
			for (const client of clients) {
				if (client.url === targetURL && 'focus' in client) {
					return client.focus();
				}
			}

			if (self.clients.openWindow) {
				return self.clients.openWindow(targetURL);
			}
		})
	);
});
