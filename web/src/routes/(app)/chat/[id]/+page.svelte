<script lang="ts">
	import ChatPage from '$lib/components/chat/chat-page.svelte';
	import type { ActionData, PageData } from './$types';

	let { data, form }: { data: PageData; form?: ActionData } = $props();
</script>

<ChatPage
	conversations={data.conversations}
	nextCursor={data.nextCursor}
	messages={data.messages}
	selectedConversationId={data.selectedConversationId}
	user={data.user}
	error={form?.error}
/>
