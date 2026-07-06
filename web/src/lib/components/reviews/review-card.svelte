<script lang="ts">
	import { applyAction, enhance } from '$app/forms';
	import UserAvatar from '$lib/components/ui/avatar/user-avatar.svelte';
	import { Badge } from '$lib/components/ui/badge/index';
	import Button from '$lib/components/ui/button.svelte';
	import StarRatingInput from './star-rating-input.svelte';
	import PropChips from './prop-chips.svelte';
	import { MAX_PROPS_PER_REVIEW, reviewPropLabel } from '$lib/consts/review-props';
	import type { components } from '$lib/api/types.gen';

	type Reviewee = components['schemas']['PlayRevieweePublic'];

	let {
		reviewee,
		peerProps,
		hostProps,
		readonly = false
	}: {
		reviewee: Reviewee;
		peerProps: string[];
		hostProps: string[];
		readonly?: boolean;
	} = $props();

	// Deliberately seeded once from the server-loaded review: the card owns
	// the draft state after mount
	// svelte-ignore state_referenced_locally
	let rating = $state<number | null>(reviewee.my_review?.rating ?? null);
	// svelte-ignore state_referenced_locally
	let selectedProps = $state<string[]>(reviewee.my_review?.props ?? []);
	// svelte-ignore state_referenced_locally
	let shoutout = $state(reviewee.my_review?.shoutout ?? '');

	let status = $state<'idle' | 'saving'>('idle');
	let errorMessage = $state<string | null>(null);

	const hasReview = $derived(reviewee.my_review != null);
	// Reviews lock after submit for now. The edit plumbing stays intact (the
	// PUT is an upsert and the window allows edits server-side) — revisit to
	// let reviewers fix shoutout typos and the like.
	const locked = $derived(readonly || hasReview);
	const isEmpty = $derived(rating == null && selectedProps.length === 0 && !shoutout.trim());
</script>

<section class="p-4 border border-border rounded-md bg-card/50">
	<div class="mb-5 flex flex-col gap-2 items-center">
		<UserAvatar
			src={reviewee.photo_url}
			nameForFallback={reviewee.display_name}
			className="h-9 w-9"
		/>
		<div class="flex gap-2 items-center">
			<p class="font-medium">{reviewee.display_name}</p>
			{#if reviewee.is_host}
				<Badge variant="info" size="xs">Host</Badge>
			{/if}
		</div>
	</div>

	{#if locked}
		<div class="text-center space-y-2">
			{#if hasReview}
				{#if reviewee.my_review?.rating != null}
					<p class="text-sm">You rated {reviewee.my_review.rating}/5</p>
				{/if}
				{#if reviewee.my_review?.props?.length}
					<div class="flex flex-wrap gap-1.5 justify-center">
						{#each reviewee.my_review.props as slug (slug)}
							<Badge variant="outline" size="xs">{reviewPropLabel(slug)}</Badge>
						{/each}
					</div>
				{/if}
				{#if reviewee.my_review?.shoutout}
					<p class="text-sm text-muted">“{reviewee.my_review.shoutout}”</p>
				{/if}
			{:else}
				<p class="text-sm text-muted">You didn't review this player.</p>
			{/if}
		</div>
	{:else}
		<form
			method="POST"
			action="?/review"
			class="space-y-5"
			use:enhance={() => {
				status = 'saving';
				errorMessage = null;
				return async ({ result }) => {
					if (result.type === 'redirect') {
						// Saved: the action sends the reviewer back to the game
						await applyAction(result);
						return;
					}
					status = 'idle';
					const failure = result.type === 'failure' ? result.data : null;
					errorMessage =
						typeof failure?.error === 'string' ? failure.error : 'Failed to save review';
				};
			}}
		>
			<input type="hidden" name="reviewee_user_id" value={reviewee.user_id} />

			<div>
				<p class="text-sm mb-2 text-center">Rate your game</p>
				<StarRatingInput
					bind:value={rating}
					label={`Rate ${reviewee.display_name}`}
					class="justify-center"
				/>
			</div>

			{#if rating != null}
				<!-- Props and the shoutout reveal once a rating is picked -->
				<div class="space-y-5">
					<div class="space-y-3">
						<div>
							<p class="text-sm mb-2 text-center">
								Give props
								<span class="text-xs text-muted">· pick up to {MAX_PROPS_PER_REVIEW}</span>
							</p>
							<PropChips options={peerProps} bind:selected={selectedProps} class="justify-center" />
						</div>
						{#if reviewee.is_host}
							<div>
								<p class="text-xs text-muted mb-1.5 text-center">For hosting</p>
								<PropChips
									options={hostProps}
									bind:selected={selectedProps}
									class="justify-center"
								/>
							</div>
						{/if}
					</div>

					<textarea
						name="shoutout"
						bind:value={shoutout}
						rows="2"
						maxlength="500"
						placeholder={`Give ${reviewee.display_name} a public shoutout`}
						class="text-sm p-2 border border-border rounded-md bg-background w-full placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-primary"
					></textarea>

					{#if errorMessage}
						<p class="text-sm text-destructive text-center">{errorMessage}</p>
					{/if}

					<Button type="submit" size="sm" class="w-full" disabled={isEmpty || status === 'saving'}>
						{status === 'saving' ? 'Submitting…' : 'Submit'}
					</Button>
				</div>
			{/if}
		</form>
	{/if}
</section>
