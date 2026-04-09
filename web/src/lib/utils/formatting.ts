export function capitalize(s: string): string {
	return s.charAt(0).toUpperCase() + s.slice(1);
}

/**
 * Format an ISO datetime string as a short absolute datetime.
 * e.g. "8 Apr 2026, 5:30 pm"
 */
export function formatDateTime(iso: string): string {
	const d = new Date(iso);
	return d.toLocaleString('en-SG', {
		day: 'numeric',
		month: 'short',
		year: 'numeric',
		hour: 'numeric',
		minute: '2-digit'
	});
}

/**
 * Format an ISO date string for display.
 * e.g. "Tue, 8 Apr"
 */
export function formatDate(iso: string, tz: string): string {
	const d = new Date(iso);
	return d.toLocaleDateString('en-SG', {
		weekday: 'short',
		month: 'short',
		day: 'numeric',
		timeZone: tz
	});
}

/**
 * Format an ISO time string for display.
 * Omits minutes when they're :00, e.g. "7 pm" vs "7:30 pm".
 */
export function formatTime(iso: string, tz: string): string {
	const d = new Date(iso);
	const minute = d.toLocaleString('en-SG', { minute: 'numeric', timeZone: tz });
	return d.toLocaleTimeString('en-SG', {
		hour: 'numeric',
		...(minute !== '0' && { minute: '2-digit' }),
		timeZone: tz
	});
}

/**
 * Format a fee in cents to a currency string.
 * Whole dollar amounts omit decimals ($10), otherwise uses standard 2 decimals ($10.50).
 */
export function formatFee(cents: number, currency: string): string {
	const dollars = cents / 100;
	const isWholeDollar = cents % 100 === 0;

	return new Intl.NumberFormat('en-SG', {
		style: 'currency',
		currency,
		minimumFractionDigits: isWholeDollar ? 0 : 2,
		maximumFractionDigits: 2
	}).format(dollars);
}

/**
 * Safely extract a finite number from an unknown value.
 */
export function getNumericFee(value: unknown): number | null {
	return typeof value === 'number' && Number.isFinite(value) ? value : null;
}

/**
 * Extract a gendered fee from a play's meta object.
 */
export function getMetaFee(
	meta: Record<string, unknown> | null | undefined,
	key: 'fee_male' | 'fee_female'
): number | null {
	if (meta == null) return null;
	return getNumericFee(meta[key]);
}

/**
 * Format a play's fee for display, falling back to gendered pricing from meta.
 */
export function formatPlayFee(play: {
	fee?: number;
	currency: string;
	meta: Record<string, unknown>;
}): string {
	const fee = getNumericFee(play.fee);
	if (fee != null) return formatFee(fee, play.currency);

	const feeMale = getMetaFee(play.meta, 'fee_male');
	const feeFemale = getMetaFee(play.meta, 'fee_female');
	const fees = [
		feeMale != null ? `${formatFee(feeMale, play.currency)} (M)` : null,
		feeFemale != null ? `${formatFee(feeFemale, play.currency)} (F)` : null
	].filter((value): value is string => value != null);

	return fees.length > 0 ? fees.join(', ') : '-';
}

/**
 * Format a skill level range for display.
 */
export function formatLevel(min?: string, max?: string): string {
	if (min && max) return `${min} - ${max}`;
	if (min) return `${min}+`;
	if (max) return `- ${max}`;
	return '-';
}
