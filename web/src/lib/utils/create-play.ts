/**
 * Converts fee from dollars (user input) to cents (API format).
 * Returns undefined if input is empty/null.
 */
export function feeToCents(dollars: string | null): number | undefined {
	if (!dollars) return undefined;
	const num = Number(dollars);
	if (isNaN(num)) return undefined;
	return Math.round(num * 100);
}

/**
 * Builds an RFC3339 timestamp from date string and time string.
 * @param date - YYYY-MM-DD format
 * @param time - HH:MM format
 * @param tzOffset - timezone offset like "+08:00"
 * @returns RFC3339 string like "2026-05-02T10:00:00+08:00", or empty if inputs missing
 */
export function buildRFC3339(date: string, time: string, tzOffset: string): string {
	if (!date || !time) return '';
	return `${date}T${time}:00${tzOffset}`;
}

/**
 * Converts a number string to number, or undefined if empty.
 */
export function optionalNumber(value: string | null): number | undefined {
	if (!value) return undefined;
	const num = Number(value);
	if (isNaN(num)) return undefined;
	return num;
}
