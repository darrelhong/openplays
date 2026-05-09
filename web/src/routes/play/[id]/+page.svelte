<script lang="ts">
	import PlayDetailsContent from '$lib/components/plays/play-details-content.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<div class="pb-8 w-full">
	<PlayDetailsContent play={data.play} dialog={false} />
</div>
