<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
	import Button from '$lib/components/ui/button.svelte';
	import UserAvatar from '$lib/components/ui/avatar/user-avatar.svelte';
	import type { components } from '$lib/api/types.gen';

	type FollowListKind = 'followers' | 'following';
	type FollowListPage = components['schemas']['PageUserSummary'];

	let {
		list,
		forbidden,
		kind
	}: {
		list: FollowListPage | null;
		forbidden: boolean;
		kind: FollowListKind;
	} = $props();

	const users = $derived(list?.items ?? []);
	const countLabel = $derived(kind === 'followers' ? 'followers' : 'following');
	const emptyTitle = $derived(
		kind === 'followers' ? 'No followers yet' : 'Not following anyone yet'
	);
	const emptyDescription = $derived(
		kind === 'followers'
			? 'Accepted followers will appear here.'
			: 'People this user follows will appear here.'
	);

	function nextPageUrl(nextCursor: string): string {
		const params = new SvelteURLSearchParams(page.url.searchParams);
		params.set('cursor', nextCursor);
		return `?${params.toString()}`;
	}
</script>

{#if forbidden}
	<section class="my-5 p-6 border border-border rounded-lg border-dashed">
		<p class="text-lg font-semibold">This list is private</p>
		<p class="text-sm text-muted mt-1">Only accepted followers can see this list.</p>
	</section>
{:else if users.length > 0}
	<p class="text-sm text-muted my-4">
		{list?.total ?? 0}
		{countLabel}
	</p>
	<ul class="gap-2 grid">
		{#each users as user (user.id)}
			<li>
				<a
					href={user.username ? resolve(`/${user.username}`) : undefined}
					class="p-3 border border-border rounded-lg bg-card/50 flex gap-3 items-center focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 hover:border-primary/50"
				>
					<UserAvatar
						src={user.photo_url}
						nameForFallback={user.display_name}
						className="h-10 w-10 text-sm"
					/>
					<div>
						<p class="text-sm text-foreground font-medium">{user.display_name}</p>
						<p class="text-sm text-muted">@{user.username}</p>
					</div>
				</a>
			</li>
		{/each}
	</ul>
{:else}
	<section class="my-5 p-6 border border-border rounded-lg border-dashed">
		<p class="text-lg font-semibold">{emptyTitle}</p>
		<p class="text-sm text-muted mt-1">{emptyDescription}</p>
	</section>
{/if}

{#if !forbidden && list}
	<div class="my-6 flex gap-4 w-full">
		{#if page.url.searchParams.has('cursor')}
			<Button variant="outline" onclick={() => history.back()}>Previous</Button>
		{/if}
		{#if list.has_more && list.next_cursor != null}
			<Button
				class="ms-auto"
				variant="outline"
				href={nextPageUrl(list.next_cursor)}
				data-sveltekit-noscroll>Next</Button
			>
		{/if}
	</div>
{/if}
