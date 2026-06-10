<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
	import Button from '$lib/components/ui/button.svelte';
	import PlaysDesktopTable from '$lib/components/plays/plays-desktop-table.svelte';
	import PlaysMobileGrid from '$lib/components/plays/plays-mobile-grid.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	function getNextPageUrl(nextCursor: string): string {
		const params = new SvelteURLSearchParams(page.url.searchParams);
		params.set('cursor', nextCursor);
		return `?${params.toString()}`;
	}
</script>

<h1 class="text-xl font-semibold mb-2">My Games</h1>

{#if data.plays.items && data.plays.items.length > 0}
	<p class="text-muted mb-3">
		Showing {data.plays.total} upcoming {data.plays.total === 1 ? 'game' : 'games'}
	</p>
	<PlaysMobileGrid plays={data.plays.items} showViewerState />
	<PlaysDesktopTable plays={data.plays.items} showViewerState />
{:else}
	<section class="my-4 p-6 border border-border rounded-lg border-dashed max-w-xl">
		<p class="text-lg font-semibold">No upcoming games!</p>
		<p class="text-sm text-muted mt-1">Create a game or browse open plays to get started.</p>
		<div class="mt-4 flex flex-wrap gap-2">
			<Button href={resolve('/create')}>Create Game</Button>
			<Button href={resolve('/')} variant="outline">Browse Plays</Button>
		</div>
	</section>
{/if}

<div class="my-6 flex gap-4 w-full">
	{#if page.url.searchParams.has('cursor')}
		<Button variant="outline" onclick={() => history.back()}>Previous</Button>
	{/if}
	{#if data.plays.has_more && data.plays.next_cursor != null}
		<Button
			class="ms-auto"
			variant="outline"
			href={getNextPageUrl(data.plays.next_cursor)}
			data-sveltekit-noscroll>Next</Button
		>
	{/if}
</div>
