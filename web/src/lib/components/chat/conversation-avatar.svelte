<script lang="ts">
	import { Gamepad2 } from '@lucide/svelte';
	import UserAvatar from '$lib/components/ui/avatar/user-avatar.svelte';
	import type { Conversation } from './types';

	let { conversation }: { conversation: Conversation } = $props();
</script>

{#if conversation.kind === 'dm'}
	<UserAvatar
		src={conversation.other_user?.photo_url}
		nameForFallback={conversation.title}
		className="h-9 w-9 text-xs"
	/>
{:else}
	<span
		class="text-primary rounded-full bg-primary/15 flex shrink-0 h-9 w-9 items-center justify-center"
	>
		<Gamepad2 class="size-4" />
	</span>
{/if}
