<script lang="ts">
	import ConversationHeader from './conversation-header.svelte';
	import ConversationList from './conversation-list.svelte';
	import MessageComposer from './message-composer.svelte';
	import MessageList from './message-list.svelte';
	import type { Conversation, Message, Viewer } from './types';

	let {
		conversations,
		nextCursor,
		messages = [],
		selectedConversationId,
		user,
		error
	}: {
		conversations: Conversation[];
		nextCursor?: string;
		messages?: Message[];
		selectedConversationId?: string;
		user: Viewer;
		error?: string;
	} = $props();

	// Fall back to a generic conversation so the view still renders when the
	// conversation is missing from the list (e.g. beyond the page limit)
	const selectedConversation = $derived.by(() => {
		if (!selectedConversationId) return undefined;
		return (
			conversations.find((conversation) => conversation.id === selectedConversationId) ??
			({
				id: selectedConversationId,
				kind: 'dm',
				title: 'Conversation',
				unread_count: 0,
				updated_at: ''
			} satisfies Conversation)
		);
	});
</script>

<div class="py-4 flex flex-1 gap-4 min-h-0 w-full">
	<ConversationList {conversations} {selectedConversationId} {nextCursor} />

	{#if selectedConversation}
		<!-- Header and composer float over the message list, which scrolls behind them.
		     The column escapes the wrapper's vertical padding so messages run to the
		     edges and fade out under the header and composer, telegram-style -->
		<section class="flex flex-1 flex-col min-w-0 relative -my-4">
			<div
				class="h-20 pointer-events-none inset-x-0 top-0 absolute from-background to-transparent bg-gradient-to-b"
			></div>
			<div class="inset-x-0 top-4 absolute z-10">
				<ConversationHeader conversation={selectedConversation} />
			</div>
			{#key selectedConversation.id}
				<MessageList
					{messages}
					viewerId={user.id}
					conversationId={selectedConversation.id}
					conversationKind={selectedConversation.kind}
				/>
			{/key}
			<div
				class="h-20 pointer-events-none inset-x-0 bottom-0 absolute from-background to-transparent bg-gradient-to-t"
			></div>
			<div class="inset-x-0 bottom-4 absolute z-10">
				<MessageComposer {error} />
			</div>
		</section>
	{/if}
</div>
