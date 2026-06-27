export type PermissionState = NotificationPermission | 'unsupported';

export type UserNotification = {
	id: string;
	title: string;
	body?: string;
	url?: string;
	kind?: string;
	play_id?: string;
	read_at?: string;
	created_at: string;
};

export type PushNotificationMessage = {
	title?: unknown;
	body?: unknown;
	url?: unknown;
	kind?: unknown;
	play_id?: unknown;
};
