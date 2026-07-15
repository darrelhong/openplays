<script lang="ts">
	import PlayDetailsContent from '$lib/components/plays/play-details-content.svelte';
	import type { ActionData, PageData } from './$types';

	let { data, form }: { data: PageData; form?: ActionData } = $props();
</script>

<div class="pb-8 w-full">
	<PlayDetailsContent
		play={data.play}
		user={data.user}
		{form}
		dialog={false}
		showLoginButton={data.showLoginButton}
		reviewedUsernames={data.reviewedUsernames}
	/>
</div>
