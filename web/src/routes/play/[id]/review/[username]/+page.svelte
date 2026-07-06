<script lang="ts">
	import ReviewCard from '$lib/components/reviews/review-card.svelte';
	import { formatDate } from '$lib/utils/formatting';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const play = $derived(data.play);
	const reviewee = $derived(data.reviewee);
	const windowState = $derived(data.window.state);
	const playTitle = $derived(play.name || play.venue_name);
</script>

<svelte:head>
	<title>Review {reviewee.display_name} · {playTitle}</title>
</svelte:head>

<div class="mx-auto max-w-xl w-full">
	<p class="text-sm text-muted text-center">
		{playTitle} · {formatDate(play.starts_at, play.timezone, { year: 'numeric' })}
	</p>
	<h1 class="font-medium mb-4 mt-1 text-center">
		How was your game with {reviewee.display_name}?
	</h1>

	{#if windowState === 'not_open'}
		<p class="text-sm text-muted p-4 border border-border rounded-md border-dashed">
			Reviews open once the game has ended.
		</p>
	{:else}
		{#if windowState === 'closed'}
			<p class="text-sm text-muted mb-4 px-3 py-2 border border-border rounded-md bg-card/50">
				The review window has closed. Reviews can no longer be changed.
			</p>
		{/if}

		<ReviewCard
			{reviewee}
			peerProps={data.peerProps}
			hostProps={data.hostProps}
			readonly={windowState === 'closed'}
		/>
	{/if}
</div>
