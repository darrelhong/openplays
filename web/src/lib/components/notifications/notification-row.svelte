<script lang="ts">
	import { invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { formatNotificationTime } from './notification-time';
	import type { UserNotification } from './types';

	let {
		item,
		unread,
		onSelect
	}: {
		item: UserNotification;
		unread: boolean;
		onSelect?: () => void;
	} = $props();

	const chatID = $derived(item.url?.match(/^\/chat\/([^/]+)$/)?.[1]);
	const urlPlayID = $derived(item.url?.match(/^\/play\/([^/]+)$/)?.[1]);
	const linkedPlayID = $derived(urlPlayID ?? item.play_id);

	// Navigating to the current URL doesn't re-run loads, so refresh explicitly
	// to pick up new messages when the conversation is already open
	function handleChatSelect() {
		onSelect?.();
		if (page.url.pathname === `/chat/${chatID}`) {
			invalidateAll();
		}
	}
</script>

{#snippet content()}
	<span class="row-span-2 min-w-0">
		<span class="text-foreground font-medium block truncate">{item.title}</span>
		{#if item.body}
			<span class="text-muted leading-snug block">{item.body}</span>
		{/if}
	</span>
	<span class="text-xs text-muted col-start-2 row-start-1 whitespace-nowrap"
		>{formatNotificationTime(item.created_at)}</span
	>
	{#if unread}
		<span
			class="mb-1.5 rounded-full bg-red-500 col-start-2 row-start-2 size-2 self-end justify-self-end"
			aria-hidden="true"
		></span>
	{/if}
{/snippet}

{#if chatID}
	<a
		href={resolve(`/chat/${chatID}`)}
		onclick={handleChatSelect}
		class="text-sm px-4 py-3 gap-3 grid grid-cols-[minmax(0,1fr)_auto] grid-rows-[auto_1fr] transition-colors items-start hover:bg-accent"
	>
		{@render content()}
	</a>
{:else if linkedPlayID}
	<a
		href={resolve(`/play/${linkedPlayID}`)}
		onclick={onSelect}
		class="text-sm px-4 py-3 gap-3 grid grid-cols-[minmax(0,1fr)_auto] grid-rows-[auto_1fr] transition-colors items-start hover:bg-accent"
	>
		{@render content()}
	</a>
{:else}
	<div
		class="text-sm px-4 py-3 gap-3 grid grid-cols-[minmax(0,1fr)_auto] grid-rows-[auto_1fr] transition-colors items-start hover:bg-accent"
	>
		{@render content()}
	</div>
{/if}
