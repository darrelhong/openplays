<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
	import Button from '$lib/components/ui/button.svelte';
	import PlaysDesktopTable from '$lib/components/plays/plays-desktop-table.svelte';
	import PlaysMobileGrid from '$lib/components/plays/plays-mobile-grid.svelte';
	import type { components } from '$lib/api/types.gen';

	let {
		plays,
		variant
	}: {
		plays: components['schemas']['PagePlayPublic'];
		variant: 'upcoming' | 'past';
	} = $props();

	function getNextPageUrl(nextCursor: string): string {
		const params = new SvelteURLSearchParams(page.url.searchParams);
		params.set('cursor', nextCursor);
		return `?${params.toString()}`;
	}
</script>

{#if plays.items && plays.items.length > 0}
	<p class="text-muted mb-3">
		Showing {plays.total}
		{variant}
		{plays.total === 1 ? 'game' : 'games'}
	</p>
	<PlaysMobileGrid plays={plays.items} />
	<PlaysDesktopTable plays={plays.items} />
{:else if variant === 'past'}
	<section class="my-4 p-6 border border-border rounded-lg border-dashed max-w-xl">
		<p class="text-lg font-semibold">No past games yet</p>
		<p class="text-sm text-muted mt-1">
			Games you hosted or played in show up here after they end.
		</p>
	</section>
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
	{#if plays.has_more && plays.next_cursor != null}
		<Button
			class="ms-auto"
			variant="outline"
			href={getNextPageUrl(plays.next_cursor)}
			data-sveltekit-noscroll>Next</Button
		>
	{/if}
</div>
