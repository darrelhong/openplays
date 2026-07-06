<script lang="ts">
	import Star from '@lucide/svelte/icons/star';
	import * as Dialog from '$lib/components/ui/dialog/index';
	import type { components } from '$lib/api/types.gen';

	let {
		rating,
		displayName
	}: {
		rating: components['schemas']['PublicUserRating'];
		displayName: string;
	} = $props();

	// distribution holds 1..5 star counts at indexes 0..4
	const columns = $derived(
		[1, 2, 3, 4, 5].map((star) => ({ star, count: rating.distribution?.[star - 1] ?? 0 }))
	);
	const maxCount = $derived(Math.max(...columns.map((column) => column.count), 1));
</script>

<Dialog.Root>
	<Dialog.Trigger>
		{#snippet child({ props })}
			<button
				type="button"
				class="text-sm inline-flex gap-1 cursor-pointer items-center hover:underline"
				aria-label={`${displayName}'s rating: ${rating.average.toFixed(1)} out of 5`}
				{...props}
			>
				<Star class="text-amber-400 h-3.5 w-3.5 fill-amber-400" aria-hidden="true" />
				{rating.average.toFixed(1)}
			</button>
		{/snippet}
	</Dialog.Trigger>
	<!-- w-fit: the modal hugs the chart instead of spanning the screen, mobile included -->
	<Dialog.Content
		class="p-5 pt-8 border border-border gap-4 min-w-56 w-fit shadow-card/30 shadow-lg"
	>
		<Dialog.Title class="sr-only">{displayName}'s rating</Dialog.Title>

		<div class="flex flex-col gap-0.5 items-center">
			<p class="text-xl font-semibold flex gap-1.5 items-center">
				<Star class="text-amber-400 h-4.5 w-4.5 fill-amber-400" aria-hidden="true" />
				{rating.average.toFixed(1)}
			</p>
			<p class="text-sm text-muted">
				{rating.count}
				{rating.count === 1 ? 'rating' : 'ratings'}
			</p>
		</div>

		<!-- w-fit keeps the baseline only as wide as the bars -->
		<div class="mx-auto w-fit">
			<div class="border-b border-border flex gap-4 items-end">
				{#each columns as { star, count } (star)}
					<div class="flex h-16 w-6 items-end justify-center">
						<div
							class="rounded-t-sm bg-amber-400 w-4"
							style={`height: ${(count / maxCount) * 64}px`}
						></div>
					</div>
				{/each}
			</div>
			<div class="mt-1 flex gap-4">
				{#each columns as { star } (star)}
					<span class="text-xs text-muted text-center w-6">{star}</span>
				{/each}
			</div>
		</div>
	</Dialog.Content>
</Dialog.Root>
