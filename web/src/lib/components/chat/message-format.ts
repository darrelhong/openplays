// Generic message formatting pipeline: body -> spans (per parser) -> segments.
//
// To add another "-ify" (e.g. mentions):
// 1. Add a segment kind to MessageSegment, e.g. { type: 'mention'; value: string; username: string }
// 2. Write an EntityParser that returns spans with a segment() builder
// 3. Register it in PARSERS (order is priority when spans overlap)
// 4. Render the new segment kind in message-bubble.svelte

export type MessageSegment =
	| { type: 'text'; value: string }
	| { type: 'link'; value: string; href: string };

// A span is a half-open range over the body plus a builder for its segment
export type EntitySpan = {
	offset: number;
	length: number;
	segment: (value: string) => MessageSegment;
};

export type EntityParser = (body: string) => EntitySpan[];

// --- linkify ---

// http(s) only, so a crafted message can never produce javascript: or data: hrefs
const URL_PATTERN = /\bhttps?:\/\/[^\s<>]+/gi;

// Sentence punctuation that commonly trails a pasted URL
const TRAILING_PUNCTUATION = /[.,!?;:)\]}'"]+$/;

const parseLinks: EntityParser = (body) => {
	const spans: EntitySpan[] = [];
	for (const match of body.matchAll(URL_PATTERN)) {
		const url = match[0].replace(TRAILING_PUNCTUATION, '') || match[0];
		spans.push({
			offset: match.index,
			length: url.length,
			segment: (value) => ({ type: 'link', value, href: value })
		});
	}
	return spans;
};

// --- pipeline ---

const PARSERS: EntityParser[] = [parseLinks];

export function formatMessage(body: string): MessageSegment[] {
	const spans = PARSERS.flatMap((parse) => parse(body)).sort((a, b) => a.offset - b.offset);

	const segments: MessageSegment[] = [];
	let lastIndex = 0;
	for (const span of spans) {
		const end = span.offset + span.length;
		// Overlapping or out-of-bounds spans lose to the earlier span and
		// render as plain text
		if (span.offset < lastIndex || span.length <= 0 || end > body.length) {
			continue;
		}
		if (span.offset > lastIndex) {
			segments.push({ type: 'text', value: body.slice(lastIndex, span.offset) });
		}
		segments.push(span.segment(body.slice(span.offset, end)));
		lastIndex = end;
	}
	if (lastIndex < body.length) {
		segments.push({ type: 'text', value: body.slice(lastIndex) });
	}
	return segments;
}
