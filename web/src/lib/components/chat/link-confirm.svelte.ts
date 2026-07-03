// Shared state for the chat link confirmation dialog: any message bubble can
// request a link open, and the single dialog instance in the chat page shows it
export const linkConfirm = $state<{ url: string | null }>({ url: null });

export function requestLinkOpen(url: string) {
	linkConfirm.url = url;
}
