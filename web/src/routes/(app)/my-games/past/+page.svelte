<script lang="ts">
	import MyGamesList from '../my-games-list.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<MyGamesList plays={data.plays} variant="past" />
