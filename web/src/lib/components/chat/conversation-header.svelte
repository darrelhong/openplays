<script lang="ts">
	import { resolve } from '$app/paths';
	import { ArrowLeft } from '@lucide/svelte';
	import ConversationAvatar from './conversation-avatar.svelte';
	import type { Conversation } from './types';

	let { conversation }: { conversation: Conversation } = $props();
</script>

<header
	class="mx-auto px-3 py-2 border border-border rounded-xl bg-card flex gap-3 max-w-3xl w-full items-center"
>
	<a
		href={resolve('/chat')}
		class="text-muted p-1 rounded-lg transition-colors hover:text-foreground hover:bg-accent md:hidden"
		aria-label="Back to conversations"
	>
		<ArrowLeft class="size-4" />
	</a>
	<ConversationAvatar {conversation} />
	<div class="min-w-0">
		<h2 class="text-sm text-foreground font-semibold truncate">
			{conversation.title}
		</h2>
		<p class="text-xs text-muted">
			{conversation.kind === 'play' ? 'Game chat' : 'Direct message'}
		</p>
	</div>
</header>
