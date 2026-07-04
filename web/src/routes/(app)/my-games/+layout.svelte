<script lang="ts">
	import { page } from '$app/state';
	import Button from '$lib/components/ui/button.svelte';
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();

	const isPast = $derived(page.url.pathname.endsWith('/past'));
</script>

<h1 class="text-xl font-semibold mb-2">My Games</h1>

<div class="mb-3 flex gap-2">
	<Button href="/my-games" size="sm" variant={isPast ? 'outline' : 'default'}>Upcoming</Button>
	<Button href="/my-games/past" size="sm" variant={isPast ? 'default' : 'outline'}>Past</Button>
</div>

{@render children()}
