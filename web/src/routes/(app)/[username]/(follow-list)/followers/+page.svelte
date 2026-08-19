<script lang="ts">
	import FollowList from '../follow-list.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<FollowList list={data.list} forbidden={data.forbidden} kind="followers" />
