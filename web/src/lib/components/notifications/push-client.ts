import type { PermissionState } from './types';

type PushStatus = {
	supported: boolean;
	permission: PermissionState;
	pushEnabled: boolean;
};

export function hasPushSupport() {
	return 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window;
}

export async function syncExistingPushSubscription(): Promise<PushStatus> {
	const supported = hasPushSupport();
	if (!supported) {
		return { supported, permission: 'unsupported', pushEnabled: false };
	}

	try {
		const registration = await navigator.serviceWorker.register('/service-worker.js');
		const permission = Notification.permission;
		if (permission !== 'granted') {
			return { supported, permission, pushEnabled: false };
		}

		const existing = await registration.pushManager.getSubscription();
		if (!existing) {
			return { supported, permission, pushEnabled: false };
		}

		await subscribeForPush(registration);
		return { supported, permission, pushEnabled: true };
	} catch {
		return { supported, permission: Notification.permission, pushEnabled: false };
	}
}

export async function requestPushNotifications(): Promise<PushStatus> {
	const supported = hasPushSupport();
	if (!supported) {
		return { supported, permission: 'unsupported', pushEnabled: false };
	}

	try {
		const registration = await navigator.serviceWorker.register('/service-worker.js');
		let permission = Notification.permission;
		if (permission === 'default') {
			permission = await Notification.requestPermission();
		}
		if (permission !== 'granted') {
			return { supported, permission, pushEnabled: false };
		}

		await subscribeForPush(registration);
		return { supported, permission, pushEnabled: true };
	} catch {
		return { supported, permission: Notification.permission, pushEnabled: false };
	}
}

async function subscribeForPush(registration: ServiceWorkerRegistration) {
	const keyResponse = await fetch('/notifications/push/vapid-public-key');
	if (!keyResponse.ok) {
		throw new Error('Failed to fetch push public key');
	}
	const { public_key: publicKey } = (await keyResponse.json()) as { public_key: string };

	const existing = await registration.pushManager.getSubscription();
	await existing?.unsubscribe();

	const subscription = await registration.pushManager.subscribe({
		userVisibleOnly: true,
		applicationServerKey: urlBase64ToUint8Array(publicKey)
	});

	await savePushSubscription(subscription);
}

async function savePushSubscription(subscription: PushSubscription) {
	const saveResponse = await fetch('/notifications/push/subscriptions', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: JSON.stringify(subscription)
	});
	if (!saveResponse.ok) {
		throw new Error('Failed to save push subscription');
	}
}

function urlBase64ToUint8Array(value: string) {
	const padding = '='.repeat((4 - (value.length % 4)) % 4);
	const base64 = (value + padding).replace(/-/g, '+').replace(/_/g, '/');
	const raw = atob(base64);
	const output = new Uint8Array(raw.length);

	for (let i = 0; i < raw.length; i += 1) {
		output[i] = raw.charCodeAt(i);
	}
	return output;
}
