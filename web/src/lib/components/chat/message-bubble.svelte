<script lang="ts">
	import { enhance } from '$app/forms';
	import { Trash2 } from '@lucide/svelte';
	import { requestLinkOpen } from './link-confirm.svelte';
	import { formatMessage } from './message-format';
	import { refreshSubmit } from './refresh-submit';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index';
	import { formatNotificationTime } from '$lib/components/notifications/notification-time';
	import { cn } from '$lib/utils/cn';
	import type { Message } from './types';

	let {
		message,
		mine,
		conversationKind
	}: {
		message: Message;
		mine: boolean;
		conversationKind: 'dm' | 'play';
	} = $props();

	let menuOpen = $state(false);
	let deleteFormEl = $state<HTMLFormElement | null>(null);

	const deletable = $derived(Boolean(message.can_delete && message.body));

	// Alignment already identifies the sender in a DM (and for own messages),
	// so only group chats label incoming bubbles
	const showSender = $derived(conversationKind === 'play' && !mine);

	const bubbleClass = $derived(
		cn(
			'px-3 pb-1 pt-2 rounded-2xl max-w-[82%] shadow-sm',
			mine
				? 'text-primary-foreground rounded-br-md bg-primary'
				: 'text-foreground border border-border rounded-bl-md bg-card'
		)
	);

	const senderName = $derived(
		message.sender.display_name || message.sender.username || 'Player'
	);

	const segments = $derived(message.body ? formatMessage(message.body) : []);

	function openLink(event: Event, href: string) {
		// Keep the click from reaching the delete-menu trigger around the bubble
		event.preventDefault();
		event.stopPropagation();
		requestLinkOpen(href);
	}
</script>

{#snippet body()}
	{#if showSender}
		<span class="text-xs font-medium block">{senderName}</span>
	{/if}
	<!-- The floated time shares the last line of text when it fits, telegram-style.
	     Markup stays whitespace-free inside the pre-wrap span -->
	<span class="text-sm block whitespace-pre-wrap break-words"
		>{#if message.body}{#each segments as segment, index (index)}{#if segment.type === 'link'}<span
					role="link"
					tabindex="0"
					class="underline underline-offset-2 cursor-pointer break-all"
					onpointerdown={(event) => event.stopPropagation()}
					onclick={(event) => openLink(event, segment.href)}
					onkeydown={(event) => {
						if (event.key === 'Enter') openLink(event, segment.href);
					}}>{segment.value}</span
				>{:else}{segment.value}{/if}{/each}{:else}Message deleted{/if}<span
			class={cn('text-[11px] ml-2 mt-1.5 float-right', mine ? 'opacity-70' : 'text-muted')}
			>{formatNotificationTime(message.created_at)}</span
		></span
	>
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
