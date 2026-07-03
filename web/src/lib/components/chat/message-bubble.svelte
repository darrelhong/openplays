<script lang="ts">
	import { enhance } from '$app/forms';
	import { Trash2 } from '@lucide/svelte';
	import { refreshSubmit } from './refresh-submit';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index';
	import { formatNotificationTime } from '$lib/components/notifications/notification-time';
	import { cn } from '$lib/utils/cn';
	import type { Message } from './types';

	let { message, mine }: { message: Message; mine: boolean } = $props();

	let menuOpen = $state(false);
	let deleteFormEl = $state<HTMLFormElement | null>(null);

	const deletable = $derived(Boolean(message.can_delete && message.body));

	const bubbleClass = $derived(
		cn(
			'px-3 py-2 rounded-2xl max-w-[82%] shadow-sm',
			mine
				? 'text-primary-foreground rounded-br-md bg-primary min-w-28'
				: 'text-foreground border border-border rounded-bl-md bg-card'
		)
	);

	const senderName = $derived(
		message.sender.display_name || message.sender.username || 'Player'
	);
</script>

{#snippet body()}
	<span class="flex gap-2 items-baseline justify-between">
		<span class="text-xs font-medium">
			{mine ? 'You' : senderName}
		</span>
		<span class={cn('text-[11px]', mine ? 'opacity-70' : 'text-muted')}>
			{formatNotificationTime(message.created_at)}
		</span>
	</span>
	<span class="text-sm block whitespace-pre-wrap break-words">
		{message.body ?? 'Message deleted'}
	</span>
{/snippet}

<div data-message-id={message.id} class={cn('flex', mine ? 'justify-end' : 'justify-start')}>
	{#if deletable}
		<DropdownMenu.Root bind:open={menuOpen}>
			<DropdownMenu.Trigger>
				{#snippet child({ props })}
					<button
						{...props}
						type="button"
						class={cn(bubbleClass, 'text-left')}
						oncontextmenu={(event) => {
							event.preventDefault();
							menuOpen = true;
						}}
					>
						{@render body()}
					</button>
				{/snippet}
			</DropdownMenu.Trigger>
			<DropdownMenu.Content align={mine ? 'end' : 'start'} sideOffset={4}>
				<DropdownMenu.Item
					class="text-destructive flex gap-2 items-center"
					onSelect={() => deleteFormEl?.requestSubmit()}
				>
					<Trash2 class="size-3.5" />
					Delete
				</DropdownMenu.Item>
			</DropdownMenu.Content>
		</DropdownMenu.Root>

		<form
			bind:this={deleteFormEl}
			method="POST"
			action="?/delete"
			use:enhance={refreshSubmit}
			class="hidden"
		>
			<input type="hidden" name="message_id" value={message.id} />
		</form>
	{:else}
		<div class={bubbleClass}>
			{@render body()}
		</div>
	{/if}
</div>
