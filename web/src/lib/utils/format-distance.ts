export function formatDistance(value: string, options: { suffix?: boolean } = {}) {
	const timestamp = new Date(value).getTime();
	if (!Number.isFinite(timestamp)) {
		return '';
	}
	const withSuffix = (label: string) => (options.suffix ? `${label} ago` : label);
	const seconds = Math.max(0, Math.round((Date.now() - timestamp) / 1000));
	if (seconds < 60) return 'now';
	const minutes = Math.round(seconds / 60);
	if (minutes < 60) return withSuffix(`${minutes}m`);
	const hours = Math.round(minutes / 60);
	if (hours < 24) return withSuffix(`${hours}h`);
	const days = Math.round(hours / 24);
	if (days < 30) return withSuffix(`${days}d`);
	return new Date(value).toLocaleDateString('en-SG', { day: 'numeric', month: 'short' });
}
