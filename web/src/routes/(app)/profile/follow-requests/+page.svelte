<script lang="ts">
	import { page } from '$app/state';
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
	import type { PageData } from './$types';
	import Button from '$lib/components/ui/button.svelte';
	import UserAvatar from '$lib/components/ui/avatar/user-avatar.svelte';

	let { data }: { data: PageData } = $props();
	const requests = $derived(data.requests.items ?? []);

	function nextPageUrl(nextCursor: string): string {
		const params = new SvelteURLSearchParams(page.url.searchParams);
		params.set('cursor', nextCursor);
		return `?${params.toString()}`;
	}
</script>

<div class="mx-auto mt-8 max-w-xl w-full">
	<section>
		<div class="mb-6">
			<h1 class="text-xl text-foreground font-bold">Follow requests</h1>
			<p class="text-sm text-muted mt-1">Approve people who want to follow you.</p>
		</div>

		{#if page.form?.error}
			<p class="text-sm text-destructive mb-4" role="alert">{page.form.error}</p>
		{/if}

		{#if requests.length === 0}
			<section class="p-6 border border-border rounded-lg border-dashed">
				<p class="text-lg font-semibold">No follow requests</p>
				<p class="text-sm text-muted mt-1">New requests will appear here.</p>
			</section>
		{:else}
			<p class="text-sm text-muted mb-4">{data.requests.total} pending requests</p>
			<ul class="gap-2 grid">
				{#each requests as requestUser (requestUser.id)}
					<li
						class="p-3 border border-border rounded-lg bg-card/50 flex gap-3 items-center justify-between"
					>
						<a
							href={requestUser.username ? resolve(`/${requestUser.username}`) : undefined}
							class="flex gap-3 min-w-0 items-center focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
						>
							<UserAvatar
								src={requestUser.photo_url}
								nameForFallback={requestUser.display_name}
								className="h-10 w-10 text-sm"
							/>
							<span class="text-sm text-foreground font-medium flex flex-col min-w-0 break-all">
								<span>{requestUser.display_name}</span>
								{#if requestUser.username}
									<span class="text-muted font-normal">@{requestUser.username}</span>
								{/if}
							</span>
						</a>

						<div class="flex shrink-0 flex-wrap gap-2 justify-end">
							<form method="POST" action="?/accept" use:enhance>
								<input type="hidden" name="user_id" value={requestUser.id} />
								<Button type="submit" size="xs" class="sm:text-sm sm:px-2.5 sm:h-8">Accept</Button>
							</form>
							<form method="POST" action="?/decline" use:enhance>
								<input type="hidden" name="user_id" value={requestUser.id} />
								<Button
									type="submit"
									size="xs"
									variant="outline"
									class="sm:text-sm sm:px-2.5 sm:h-8">Decline</Button
								>
							</form>
						</div>
					</li>
				{/each}
			</ul>
		{/if}

		{#if page.url.searchParams.has('cursor') || (data.requests.has_more && data.requests.next_cursor)}
			<div class="my-6 flex gap-4 w-full">
				{#if page.url.searchParams.has('cursor')}
					<Button variant="outline" onclick={() => history.back()}>Previous</Button>
				{/if}
				{#if data.requests.has_more && data.requests.next_cursor}
					<Button
						class="ms-auto"
						href={nextPageUrl(data.requests.next_cursor)}
						variant="outline"
						data-sveltekit-noscroll
					>
						Next
					</Button>
				{/if}
			</div>
		{/if}
	</section>
</div>
