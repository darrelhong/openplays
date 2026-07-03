<script lang="ts">
	import { resolve } from '$app/paths';
	import { formatNotificationTime } from '$lib/components/notifications/notification-time';
	import { cn } from '$lib/utils/cn';
	import ConversationAvatar from './conversation-avatar.svelte';
	import type { Conversation } from './types';

	let {
		conversations,
		selectedConversationId
	}: {
		conversations: Conversation[];
		selectedConversationId?: string;
	} = $props();

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

	{#if conversations.length === 0}
		<p class="text-sm text-muted px-4 py-5">No conversations yet</p>
	{:else}
		<nav class="p-2 flex flex-1 flex-col gap-0.5 overflow-y-auto">
			{#each conversations as conversation (conversation.id)}
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
		</nav>
	{/if}
</aside>
