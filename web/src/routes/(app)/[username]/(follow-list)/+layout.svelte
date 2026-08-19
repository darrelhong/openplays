<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import type { LayoutData } from './$types';

	let { data, children }: { data: LayoutData; children: import('svelte').Snippet } = $props();
	const isFollowing = $derived(page.url.pathname.endsWith('/following'));
</script>

<div class="mx-auto mt-8 max-w-xl w-full">
	<section class="py-4">
		<a
			href={resolve(`/${data.profile.username}`)}
			class="text-lg text-foreground font-semibold rounded-sm w-fit focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2"
		>
			<h1>@{data.profile.username}</h1>
		</a>

		<nav class="mt-5 border-b border-border flex gap-2" aria-label="Profile connections">
			<a
				href={resolve(`/${data.profile.username}/followers`)}
				aria-current={!isFollowing ? 'page' : undefined}
				class={`px-3 py-2 text-sm font-medium rounded-t-md transition-colors focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 ${
					!isFollowing
						? 'text-foreground border-b-2 border-primary'
						: 'text-muted hover:text-foreground'
				}`}
			>
				Followers
			</a>
			<a
				href={resolve(`/${data.profile.username}/following`)}
				aria-current={isFollowing ? 'page' : undefined}
				class={`px-3 py-2 text-sm font-medium rounded-t-md transition-colors focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-2 ${
					isFollowing
						? 'text-foreground border-b-2 border-primary'
						: 'text-muted hover:text-foreground'
				}`}
			>
				Following
			</a>
		</nav>

		{@render children()}
	</section>
</div>
