<script lang="ts">
	import { enhance } from '$app/forms';
	import Star from '@lucide/svelte/icons/star';
	import type { SubmitFunction } from '@sveltejs/kit';
	import Button from '$lib/components/ui/button.svelte';
	import type { Play } from './types';

	let { play }: { play: Play } = $props();

	let optimisticFavourite = $state<boolean | null>(null);
	let hasOptimisticFavourite = $state(false);
	let pending = $state(false);
	let errorMessage = $state<string | null>(null);

	$effect(() => {
		const backendFavourite = play.is_favourited ?? null;
		hasOptimisticFavourite = false;
		optimisticFavourite = backendFavourite;
		errorMessage = null;
	});

	const currentFavourite = $derived(
		hasOptimisticFavourite ? optimisticFavourite : (play.is_favourited ?? null)
	);
	const isKnown = $derived(currentFavourite != null);
	const isFavourited = $derived(currentFavourite === true);
	const action = $derived(isFavourited ? '?/unfavourite' : '?/favourite');
	const label = $derived(isFavourited ? 'Remove from favourites' : 'Add to favourites');

	const enhanceFavourite: SubmitFunction = ({ cancel }) => {
		if (pending || currentFavourite == null) {
			cancel();
			return;
		}

		const previous = currentFavourite;
		optimisticFavourite = !previous;
		hasOptimisticFavourite = true;
		pending = true;
		errorMessage = null;

		return async ({ result }) => {
			pending = false;

			if (result.type === 'failure' || result.type === 'error') {
				optimisticFavourite = previous;
				hasOptimisticFavourite = true;
				errorMessage =
					result.type === 'failure' && typeof result.data?.error === 'string'
						? result.data.error
						: 'Failed to update favourite';
			}
		};
	};
</script>

{#if isKnown}
	<form method="POST" {action} use:enhance={enhanceFavourite}>
		<input type="hidden" name="play_id" value={play.id} />
		<Button
			type="submit"
			size="xs"
			variant="ghost"
			title={label}
			aria-label={label}
			disabled={pending}
			class="p-0 shrink-0 h-6 w-6"
		>
			<Star class={isFavourited ? 'h-3.5 w-3.5 fill-current' : 'h-3.5 w-3.5'} />
		</Button>
		{#if errorMessage}
			<span class="sr-only" aria-live="polite">{errorMessage}</span>
		{/if}
	</form>
{/if}
