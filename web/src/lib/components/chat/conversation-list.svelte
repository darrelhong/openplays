<script lang="ts">
	import { tick } from 'svelte';
	import { resolve } from '$app/paths';
	import { formatNotificationTime } from '$lib/components/notifications/notification-time';
	import { cn } from '$lib/utils/cn';
	import ConversationAvatar from './conversation-avatar.svelte';
	import type { Conversation } from './types';

	let {
		conversations,
		selectedConversationId,
		nextCursor
	}: {
		conversations: Conversation[];
		selectedConversationId?: string;
		nextCursor?: string;
	} = $props();

	let olderConversations = $state<Conversation[]>([]);
	let loadingOlder = $state(false);
	// undefined = continue from the first page's cursor; null = no more pages
	let cursor = $state<string | null | undefined>(undefined);

	const effectiveCursor = $derived(cursor === undefined ? (nextCursor ?? null) : cursor);

	// A conversation with new activity moves onto the refreshed first page, so
	// dedupe keeps that fresher copy over the paginated one
	const allConversations = $derived.by(() => {
		const firstPageIds = new Set(conversations.map((conversation) => conversation.id));
		return [
			...conversations,
			...olderConversations.filter((conversation) => !firstPageIds.has(conversation.id))
		];
	});

	let sentinelVisible = false;

	async function loadOlder() {
		if (loadingOlder) return;
		loadingOlder = true;
		try {
			// The observer only fires on enter/leave, and a filtered page may be too
			// short to push the sentinel out — keep loading until it leaves view or
			// pages run out
			while (sentinelVisible && effectiveCursor) {
				const response = await fetch(
					`/chat/conversations?cursor=${encodeURIComponent(effectiveCursor)}`
				);
				if (!response.ok) return;
				const { items, next_cursor }: { items: Conversation[]; next_cursor: string | null } =
					await response.json();
				// Conversations with no messages stay hidden, same as the first page
				olderConversations = [
					...olderConversations,
					...items.filter((conversation) => conversation.last_message)
				];
				cursor = next_cursor;
				await tick();
			}
		} finally {
			loadingOlder = false;
		}
	}

	let navEl = $state<HTMLElement | null>(null);
	let sentinelEl = $state<HTMLElement | null>(null);

	$effect(() => {
		if (!navEl || !sentinelEl) return;
		const observer = new IntersectionObserver(
			(entries) => {
				sentinelVisible = entries.some((entry) => entry.isIntersecting);
				if (sentinelVisible) {
					loadOlder();
				}
			},
			{ root: navEl, rootMargin: '200px' }
		);
		observer.observe(sentinelEl);
		return () => observer.disconnect();
	});

	function conversationSubtitle(conversation: Conversation) {
		if (conversation.last_message?.body) {
			return conversation.last_message.body;
		}
		if (conversation.last_message?.deleted_at) {
			return 'Message deleted';
		}
		return conversation.kind === 'play' ? 'Game chat' : 'Direct message';
	}
</script>

<aside
	class={cn(
		'border border-border rounded-xl bg-card flex flex-col w-full overflow-hidden md:shrink-0 md:w-80',
		selectedConversationId ? 'hidden md:flex' : 'flex'
	)}
>
	<div class="px-4 py-3 border-b border-border">
		<h1 class="text-base text-foreground font-semibold">Chat</h1>
	</div>

	{#if allConversations.length === 0}
		<p class="text-sm text-muted px-4 py-5">No conversations yet</p>
	{:else}
		<nav bind:this={navEl} class="p-2 flex flex-1 flex-col gap-0.5 overflow-y-auto">
			{#each allConversations as conversation (conversation.id)}
				<a
					href={resolve(`/chat/${conversation.id}`)}
					class={cn(
						'px-2 py-2.5 rounded-lg flex gap-3 transition-colors hover:bg-accent/70',
						conversation.id === selectedConversationId && 'bg-accent'
					)}
				>
					<ConversationAvatar {conversation} />
					<span class="flex-1 min-w-0">
						<span class="flex gap-2 items-center justify-between">
							<span class="text-sm text-foreground font-medium truncate">{conversation.title}</span>
							<span class="text-xs text-muted shrink-0"
								>{formatNotificationTime(conversation.updated_at)}</span
							>
						</span>
						<span class="mt-0.5 flex gap-2 items-center">
							<span class="text-xs text-muted truncate">{conversationSubtitle(conversation)}</span>
							{#if conversation.unread_count > 0}
								<span class="rounded-full bg-red-500 shrink-0 size-2" aria-hidden="true"></span>
							{/if}
						</span>
					</span>
				</a>
			{/each}
			{#if loadingOlder}
				<p class="text-xs text-muted py-2 text-center">Loading more…</p>
			{/if}
			{#if effectiveCursor}
				<div bind:this={sentinelEl} aria-hidden="true"></div>
			{/if}
		</nav>
	{/if}
</aside>
