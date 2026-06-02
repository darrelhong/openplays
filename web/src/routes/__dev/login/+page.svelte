<script lang="ts">
	import { enhance } from '$app/forms';
	import Button from '$lib/components/ui/button.svelte';

	let { data, form } = $props();
</script>

<svelte:head>
	<title>Dev Login - OpenPlays</title>
</svelte:head>

<section class="mx-auto px-4 py-10 flex flex-col max-w-2xl min-h-[70vh] w-full justify-center">
	<div class="space-y-8">
		<div class="space-y-2">
			<p class="text-sm text-emerald-700 tracking-wide font-medium uppercase">Local development</p>
			<h1 class="text-3xl text-zinc-950 font-semibold">Dev login</h1>
		</div>

		{#if form?.error}
			<p class="text-sm text-red-700 px-4 py-3 border border-red-200 rounded-md bg-red-50">
				{form.error}
			</p>
		{/if}

		<div
			class="border border-zinc-200 rounded-lg bg-white overflow-hidden divide-zinc-200 divide-y"
		>
			{#each data.seedUsers as user (user.id)}
				<form
					method="POST"
					action="?/login"
					use:enhance
					class="p-4 flex gap-4 items-center justify-between"
				>
					<div class="min-w-0">
						<p class="text-zinc-950 font-medium">{user.displayName}</p>
						<p class="text-sm text-zinc-600">{user.description}</p>
						<p class="text-xs text-zinc-500 font-mono mt-1">{user.id}</p>
					</div>
					<input type="hidden" name="user_id" value={user.id} />
					<Button type="submit">Log in</Button>
				</form>
			{/each}
		</div>
	</div>
</section>
