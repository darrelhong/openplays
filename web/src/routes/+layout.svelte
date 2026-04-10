<script lang="ts">
	import 'virtual:uno.css';
	import favicon from '$lib/assets/favicon.svg';
	import { resolve } from '$app/paths';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" type="image/svg+xml" href={favicon} />
	<title>OpenPlays</title>
	<meta name="description" content="Discover and organise sports sessions" />
</svelte:head>

<div class="text-stone-50 bg-stone-900 min-h-screen">
	<header class="p-4">
		<a href={resolve('/')} class="text-2xl text-white font-bold">OpenPlays</a>
	</header>
	<main class="px-4">
		{@render children()}
	</main>
</div>
