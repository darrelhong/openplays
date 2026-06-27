<script lang="ts">
	import { resolve } from '$app/paths';
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

{#if item.play_id}
	<a
		href={resolve(`/play/${item.play_id}`)}
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
