<script lang="ts">
	import PlayEditContent from '$lib/components/plays/play-edit-content.svelte';
	import type { ActionData, PageData } from './$types';

	let { data, form }: { data: PageData; form?: ActionData } = $props();
</script>

<PlayEditContent play={data.play} {form} />
