<script lang="ts">
	import { Badge } from '$lib/components/ui/badge/index';
	import type { Play } from './types';

	type ViewerState = Play['viewer_state'];

	let { state }: { state: ViewerState } = $props();

	const labels = {
		creator: 'Hosting',
		confirmed: 'Going',
		added: 'Pending',
		waitlisted: 'Waitlisted',
		not_joined: undefined
	} as const;

	const variants = {
		creator: 'info',
		confirmed: 'success',
		added: 'warning',
		waitlisted: 'outline',
		not_joined: 'muted'
	} as const;

	const label = $derived(state ? labels[state] : undefined);
	const variant = $derived(state ? variants[state] : 'muted');
</script>

{#if label}
	<Badge {variant}>{label}</Badge>
{/if}
