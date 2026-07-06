<script lang="ts">
	import type { Snippet } from 'svelte';
	import * as Dialog from '$lib/components/ui/dialog/index';
	import { BADMINTON_LEVELS } from '$lib/consts/index';
	import { reviewPropLabel } from '$lib/consts/review-props';
	import type { components } from '$lib/api/types.gen';

	let {
		sport,
		sportLabel,
		props,
		trigger
	}: {
		sport: components['schemas']['PublicUserProfileSport'];
		sportLabel: string;
		props: { prop: string; count: number }[];
		trigger: Snippet<[{ props: Record<string, unknown> }]>;
	} = $props();

	const totalProps = $derived(props.reduce((sum, row) => sum + row.count, 0));

	// Prefer the spelled-out level name ("Low Intermediate") when one exists
	const selfRating = $derived.by(() => {
		if (!sport.rating_code) return null;
		const label = BADMINTON_LEVELS.find((level) => level.value === sport.rating_code)?.label;
		return label?.split(' - ')[1] ?? sport.rating_code;
	});

	function gamesLabel(count: number) {
		return count === 1 ? 'Game' : 'Games';
	}
</script>

<Dialog.Root>
	<Dialog.Trigger>
		{#snippet child({ props: triggerProps })}
			{@render trigger({ props: triggerProps })}
		{/snippet}
	</Dialog.Trigger>
	<!-- max-w-[calc(100%-2rem)] keeps the mobile edge gutter; sm:max-w-sm caps it on larger screens -->
	<Dialog.Content
		class="p-6 border border-border gap-5 max-w-[calc(100%-2rem)] w-full shadow-card/30 shadow-lg sm:max-w-sm"
	>
		<Dialog.Title class="text-base font-semibold text-center">{sportLabel}</Dialog.Title>

		<div class="text-center">
			<p class="text-foreground font-medium">
				{selfRating ?? 'Not set'}
			</p>
			<p class="text-xs text-muted">Self rating</p>
		</div>

		<div class="flex gap-10 justify-center">
			<div class="text-center">
				<p class="text-lg text-foreground font-semibold">{sport.rostered_play_count}</p>
				<p class="text-xs text-muted">{gamesLabel(sport.rostered_play_count)}</p>
			</div>
			<div class="text-center">
				<p class="text-lg text-foreground font-semibold">{totalProps}</p>
				<p class="text-xs text-muted">{totalProps === 1 ? 'Prop' : 'Props'}</p>
			</div>
		</div>

		{#if props.length > 0}
			<div class="flex flex-wrap gap-2 justify-center">
				{#each props as { prop, count } (prop)}
					<span
						class="text-sm px-3 py-1.5 border border-border rounded-full bg-card/50 inline-flex gap-1.5 items-center"
					>
						{reviewPropLabel(prop)}
						<span class="text-xs text-amber-500 font-semibold">×{count}</span>
					</span>
				{/each}
			</div>
		{:else}
			<p class="text-sm text-muted text-center">No props received yet</p>
		{/if}
	</Dialog.Content>
</Dialog.Root>
