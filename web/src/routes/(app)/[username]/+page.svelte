<script lang="ts">
	import { enhance } from '$app/forms';
	import UserAvatar from '$lib/components/ui/avatar/user-avatar.svelte';
	import Button from '$lib/components/ui/button.svelte';
	import { SPORTS } from '$lib/consts/index';
	import type { ActionData, PageData } from './$types';

	let { data, form }: { data: PageData; form?: ActionData } = $props();

	const profile = $derived(data.profile);
	const sports = $derived(profile.sports ?? []);
	const isOwnProfile = $derived(data.user.id === profile.id);

	function sportLabel(value: string) {
		return SPORTS.find((sport) => sport.value === value)?.label ?? value;
	}

	function gamesLabel(count: number) {
		return count === 1 ? '1 game' : `${count} games`;
	}
</script>

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
					<h1 class="text-lg text-foreground font-semibold break-words">
						{profile.display_name}
					</h1>
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
						<div class="px-3 py-2 border border-border rounded-md bg-card/50">
							<div class="flex gap-3 items-center justify-between">
								<p class="text-sm text-foreground font-medium">{sportLabel(sport.sport)}</p>
								<p class="text-xs text-muted">{gamesLabel(sport.rostered_play_count)}</p>
							</div>
							<p class="text-xs text-muted mt-1">
								Rating <span class="text-foreground">{sport.rating_code ?? 'Not set'}</span>
							</p>
						</div>
					{/each}
				</div>
			{:else}
				<p class="text-sm text-muted">No ratings yet</p>
			{/if}
		</section>
	</section>
</div>
