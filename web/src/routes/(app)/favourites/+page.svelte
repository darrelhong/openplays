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

<h1 class="text-xl font-semibold mb-2">Favourites</h1>

{#if data.plays.items && data.plays.items.length > 0}
	<p class="text-muted mb-3">
		Showing {data.plays.total} favourite {data.plays.total === 1 ? 'listing' : 'listings'}
	</p>
	<PlaysMobileGrid plays={data.plays.items} />
	<PlaysDesktopTable plays={data.plays.items} />
{:else}
	<section class="my-4 p-6 border border-border rounded-lg border-dashed max-w-xl">
		<p class="text-lg font-semibold">No favourites yet!</p>
		<p class="text-sm text-muted mt-1">Save listings from the browse page to find them here.</p>
		<div class="mt-4 flex flex-wrap gap-2">
			<Button href={resolve('/')}>Browse Listings</Button>
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
