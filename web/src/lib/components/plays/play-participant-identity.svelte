<script lang="ts">
	import { resolve } from '$app/paths';
	import { RatingBadge } from '$lib/components/ui/badge/index';
	import UserAvatar from '$lib/components/ui/avatar/user-avatar.svelte';
	import type { components } from '$lib/api/types.gen';

	type Participant = components['schemas']['PlayParticipantPreviewPublic'];

	let {
		participant,
		secondary
	}: {
		participant: Participant;
		secondary?: string;
	} = $props();

	const name = $derived(
		participant.display_name ?? (participant.is_guest ? 'Guest player' : 'Player')
	);
	const secondaryLabel = $derived(
		secondary ?? (participant.is_host ? 'Host' : participant.is_guest ? 'Guest' : null)
	);
</script>

<div class="flex gap-3 min-w-0 items-center">
	{#if participant.username}
		<a
			href={resolve(`/${participant.username}`)}
			aria-label={`View ${name}'s profile`}
			class="rounded-full shrink-0 focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
		>
			<div class="relative">
				<UserAvatar
					src={participant.photo_url}
					nameForFallback={name}
					className="h-9 w-9 text-xs"
				/>
				{#if participant.rating_code}
					<RatingBadge value={participant.rating_code} class="absolute -bottom-1 -right-1" />
				{/if}
			</div>
		</a>
	{:else}
		<div class="shrink-0 relative">
			<UserAvatar src={participant.photo_url} nameForFallback={name} className="h-9 w-9 text-xs" />
			{#if participant.rating_code}
				<RatingBadge value={participant.rating_code} class="absolute -bottom-1 -right-1" />
			{/if}
		</div>
	{/if}
	<div class="min-w-0">
		<p class="text-sm text-foreground font-medium break-words">
			{name}
		</p>
		{#if secondaryLabel}
			<p class="text-xs text-muted">{secondaryLabel}</p>
		{/if}
	</div>
</div>
