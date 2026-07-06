<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import UserAvatar from '$lib/components/ui/avatar/user-avatar.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import PropsSummaryDialog from '$lib/components/reviews/props-summary-dialog.svelte';
	import RatingSummaryDialog from '$lib/components/reviews/rating-summary-dialog.svelte';
	import { SPORTS } from '$lib/consts/index';
	import { formatDate } from '$lib/utils/formatting';
	import type { ActionData, PageData } from './$types';

	let { data, form }: { data: PageData; form?: ActionData } = $props();

	const profile = $derived(data.profile);
	const sports = $derived(profile.sports ?? []);
	const isOwnProfile = $derived(data.user.id === profile.id);
	const shoutouts = $derived(profile.shoutouts ?? []);
	// Props are sport-linked: they render inside the sport's card
	function propsFor(sport: string) {
		return (profile.props ?? []).filter((row) => row.sport === sport);
	}

	function sportLabel(value: string) {
		return SPORTS.find((sport) => sport.value === value)?.label ?? value;
	}

	function gamesLabel(count: number) {
		return count === 1 ? '1 game' : `${count} games`;
	}

	type SportSummary = (typeof sports)[number];
	type SportProps = { prop: string; count: number }[];
</script>

{#snippet sportCardContent(sport: SportSummary, sportProps: SportProps)}
	<div class="flex gap-3 items-center justify-between">
		<p class="text-sm text-foreground font-medium">{sportLabel(sport.sport)}</p>
		<p class="text-xs text-muted">{gamesLabel(sport.rostered_play_count)}</p>
	</div>
	<div class="mt-1 flex gap-3 items-center">
		<p class="text-xs text-muted">
			Rating <span class="text-foreground">{sport.rating_code ?? 'Not set'}</span>
		</p>
		{#if sportProps.length > 0}
			<span class="text-xs text-muted">
				{sportProps.reduce((sum, row) => sum + row.count, 0)} props
			</span>
		{/if}
	</div>
{/snippet}

<div class="mx-auto mt-8 max-w-xl w-full">
	<section class="py-4">
		<div class="flex gap-4 items-start justify-between">
			<div class="flex gap-4 min-w-0 items-center">
				<UserAvatar
					src={profile.photo_url}
					nameForFallback={profile.display_name}
					className="h-12 w-12 text-lg"
				/>
				<div class="min-w-0">
					<div class="flex gap-3 min-w-0 items-center">
						<h1 class="text-lg text-foreground font-semibold break-words">
							{profile.display_name}
						</h1>
						{#if profile.rating}
							<RatingSummaryDialog rating={profile.rating} displayName={profile.display_name} />
						{/if}
					</div>
					<p class="text-sm text-muted">@{profile.username}</p>
				</div>
			</div>
			{#if !isOwnProfile}
				<form method="POST" action="?/message" use:enhance>
					<Button type="submit" size="sm">Message</Button>
				</form>
			{/if}
		</div>

		{#if form?.error}
			<p class="text-sm text-destructive mt-3">{form.error}</p>
		{/if}

		<div class="mt-6 py-4 border-y border-border">
			<p class="text-sm text-muted">Games</p>
			<p class="text-lg text-foreground font-semibold">{profile.rostered_play_count}</p>
		</div>

		<section class="mt-5">
			<h2 class="text-sm text-foreground font-semibold mb-3">Sports</h2>
			{#if sports.length > 0}
				<div class="gap-2 grid">
					{#each sports as sport (sport.sport)}
						{@const sportProps = propsFor(sport.sport)}
						<!-- The whole card opens the sport's summary -->
						<PropsSummaryDialog {sport} sportLabel={sportLabel(sport.sport)} props={sportProps}>
							{#snippet trigger({ props: triggerProps })}
								<button
									type="button"
									class="px-3 py-2 text-start border border-border rounded-md bg-card/50 w-full cursor-pointer transition-colors hover:border-primary/50"
									{...triggerProps}
								>
									{@render sportCardContent(sport, sportProps)}
								</button>
							{/snippet}
						</PropsSummaryDialog>
					{/each}
				</div>
			{:else}
				<p class="text-sm text-muted">No ratings yet</p>
			{/if}
		</section>

		{#if shoutouts.length > 0}
			<section class="mt-5">
				<h2 class="text-sm text-foreground font-semibold mb-3">Shoutouts</h2>
				<ul class="gap-2 grid">
					{#each shoutouts as shoutout (`${shoutout.play_id}:${shoutout.reviewer_username ?? shoutout.reviewer_display_name}`)}
						<li class="px-3 py-2 border border-border rounded-md bg-card/50">
							<p class="text-sm text-foreground">“{shoutout.shoutout}”</p>
							<div class="mt-1.5 flex gap-2 items-center">
								{#if shoutout.reviewer_username}
									<a
										href={resolve(`/${shoutout.reviewer_username}`)}
										aria-label={`View ${shoutout.reviewer_display_name}'s profile`}
										class="flex gap-2 min-w-0 items-center focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
									>
										<UserAvatar
											src={shoutout.reviewer_photo_url}
											nameForFallback={shoutout.reviewer_display_name}
											className="h-5 w-5 text-[10px]"
										/>
										<span class="text-xs text-muted truncate">
											{shoutout.reviewer_display_name}
										</span>
									</a>
								{:else}
									<UserAvatar
										src={shoutout.reviewer_photo_url}
										nameForFallback={shoutout.reviewer_display_name}
										className="h-5 w-5 text-[10px]"
									/>
									<span class="text-xs text-muted truncate">
										{shoutout.reviewer_display_name}
									</span>
								{/if}
								<p class="text-xs text-muted shrink-0">
									· {sportLabel(shoutout.sport)} · {formatDate(
										shoutout.play_starts_at,
										shoutout.play_timezone,
										{ weekday: undefined, year: 'numeric' }
									)}
								</p>
							</div>
						</li>
					{/each}
				</ul>
			</section>
		{/if}
	</section>
</div>
