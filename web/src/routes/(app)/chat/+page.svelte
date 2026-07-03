<script lang="ts">
	import ChatPage from '$lib/components/chat/chat-page.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<ChatPage conversations={data.conversations} user={data.user} />
