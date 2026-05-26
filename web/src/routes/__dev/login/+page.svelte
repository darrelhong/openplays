<script lang="ts">
	import { enhance } from '$app/forms';
	import Button from '$lib/components/ui/button.svelte';

	let { data, form } = $props();
</script>

<svelte:head>
	<title>Dev Login - OpenPlays</title>
</svelte:head>

<section class="mx-auto flex min-h-[70vh] w-full max-w-2xl flex-col justify-center px-4 py-10">
	<div class="space-y-8">
		<div class="space-y-2">
			<p class="text-sm font-medium uppercase tracking-wide text-emerald-700">Local development</p>
			<h1 class="text-3xl font-semibold text-zinc-950">Dev login</h1>
		</div>

		{#if form?.error}
			<p class="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
				{form.error}
			</p>
		{/if}

		<div class="divide-y divide-zinc-200 overflow-hidden rounded-lg border border-zinc-200 bg-white">
			{#each data.seedUsers as user}
				<form method="POST" action="?/login" use:enhance class="flex items-center justify-between gap-4 p-4">
					<div class="min-w-0">
						<p class="font-medium text-zinc-950">{user.displayName}</p>
						<p class="text-sm text-zinc-600">{user.description}</p>
						<p class="mt-1 font-mono text-xs text-zinc-500">{user.id}</p>
					</div>
					<input type="hidden" name="user_id" value={user.id} />
					<Button type="submit">Log in</Button>
				</form>
			{/each}
		</div>
	</div>
</section>
