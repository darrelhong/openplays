<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { ArrowLeft } from '@lucide/svelte';
	import ConversationAvatar from './conversation-avatar.svelte';
	import type { Conversation } from './types';

	let { conversation }: { conversation: Conversation } = $props();

	// Arriving from a game page, the back-to-conversations arrow is out of context
	const cameFromGame = $derived(page.url.searchParams.get('from') === 'play');
</script>

<header
	class="mx-auto px-3 py-2 border border-border rounded-xl bg-card flex gap-3 max-w-3xl w-full items-center"
>
	{#if !cameFromGame}
		<a
			href={resolve('/chat')}
			class="text-muted p-1 rounded-lg transition-colors hover:text-foreground hover:bg-accent md:hidden"
			aria-label="Back to conversations"
		>
			<ArrowLeft class="size-4" />
		</a>
	{/if}
	{#if conversation.kind === 'play' && conversation.play_id}
		<a
			href={resolve(`/play/${conversation.play_id}`)}
			class="group flex flex-1 gap-3 min-w-0 items-center"
		>
			<ConversationAvatar {conversation} />
			<span class="min-w-0">
				<h2 class="text-sm text-foreground font-semibold block truncate group-hover:underline">
					{conversation.title}
				</h2>
				<span class="text-xs text-muted block">Game chat</span>
			</span>
		</a>
	{:else}
		<ConversationAvatar {conversation} />
		<div class="min-w-0">
			<h2 class="text-sm text-foreground font-semibold truncate">
				{conversation.title}
			</h2>
			<p class="text-xs text-muted">
				{conversation.kind === 'play' ? 'Game chat' : 'Direct message'}
			</p>
		</div>
	{/if}
</header>
