<script lang="ts">
	import { tick } from 'svelte';
	import MessageBubble from './message-bubble.svelte';
	import type { Message } from './types';

	let {
		messages,
		viewerId,
		conversationId,
		conversationKind
	}: {
		messages: Message[];
		viewerId: string;
		conversationId: string;
		conversationKind: 'dm' | 'play';
	} = $props();

	const PAGE_SIZE = 50;

	let olderMessages = $state<Message[]>([]);
	let loadingOlder = $state(false);
	let reachedEnd = $state(false);

	// The latest page from the server can overlap the cached history after a
	// send refreshes it, so dedupe by id
	const allMessages = $derived.by(() => {
		const latestIds = new Set(messages.map((message) => message.id));
		return [...olderMessages.filter((message) => !latestIds.has(message.id)), ...messages];
	});

	// A short first page means there's no older history to fetch
	const exhausted = $derived(reachedEnd || allMessages.length < PAGE_SIZE);

	let sentinelVisible = false;

	async function loadOlder() {
		if (loadingOlder) return;
		loadingOlder = true;
		try {
			// The observer only fires when the sentinel enters or leaves view, and a
			// fetched page may be too short to push it out — keep loading until the
			// sentinel is off screen or history runs out
			while (sentinelVisible && !exhausted) {
				const oldest = allMessages[0];
				if (!oldest) return;
				const response = await fetch(`/chat/${conversationId}/messages?before_id=${oldest.id}`);
				if (!response.ok) return;
				const { items }: { items: Message[] } = await response.json();
				const anchor = scrollEl?.querySelector(`[data-message-id="${oldest.id}"]`);
				const anchorTop = anchor?.getBoundingClientRect().top ?? 0;
				olderMessages = [...items, ...olderMessages];
				if (items.length < PAGE_SIZE) {
					reachedEnd = true;
				}
				// Flush the DOM so the observer can report the sentinel leaving view
				await tick();
				// Keep the viewport anchored on the previously-oldest message so the
				// prepended history extends upward instead of pushing messages down
				if (anchor && scrollEl) {
					scrollEl.scrollTop += anchor.getBoundingClientRect().top - anchorTop;
				}
			}
		} finally {
			loadingOlder = false;
		}
	}

	let scrollEl = $state<HTMLElement | null>(null);
	let sentinelEl = $state<HTMLElement | null>(null);

	$effect(() => {
		if (!scrollEl || !sentinelEl) return;
		const observer = new IntersectionObserver(
			(entries) => {
				sentinelVisible = entries.some((entry) => entry.isIntersecting);
				if (sentinelVisible) {
					loadOlder();
				}
			},
			{ root: scrollEl, rootMargin: '200px' }
		);
		observer.observe(sentinelEl);
		return () => observer.disconnect();
	});
</script>

<!-- column-reverse anchors the scroll position to the newest messages without scroll scripting -->
<div bind:this={scrollEl} class="flex flex-1 flex-col-reverse overflow-y-auto">
	<!-- Top/bottom padding keeps messages clear of the floating header and composer -->
	<div class="mx-auto pb-24 pt-24 flex flex-col gap-2 max-w-3xl w-full">
		{#if allMessages.length === 0}
			<p class="text-sm text-muted py-10 text-center">No messages yet</p>
		{:else}
			{#if !exhausted}
				<div bind:this={sentinelEl} aria-hidden="true"></div>
			{/if}
			{#if loadingOlder}
				<p class="text-xs text-muted py-2 text-center">Loading older messages…</p>
			{/if}
			{#each allMessages as message (message.id)}
				<MessageBubble {message} mine={message.sender.id === viewerId} {conversationKind} />
			{/each}
		{/if}
	</div>
</div>
