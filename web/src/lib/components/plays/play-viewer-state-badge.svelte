<script lang="ts">
	import { Badge } from '$lib/components/ui/badge/index';
	import type { Play } from './types';

	type ViewerState = Play['viewer_state'];

	let {
		state,
		requireWaitlist = false
	}: {
		state: ViewerState;
		requireWaitlist?: boolean;
	} = $props();

	// On classic plays the pending queue is presented as requests, so a
	// waitlisted row reads "Requested"; only require-waitlist plays have a
	// real (host-parked) waitlist
	const labels = $derived({
		creator: 'Hosting',
		confirmed: 'Going',
		added: 'Pending',
		requested: 'Requested',
		waitlisted: requireWaitlist ? 'Waitlisted' : 'Requested',
		not_joined: undefined
	} as const);

	// Anything presented as "Requested" uses the warning variant, matching the
	// roster badges on the play detail page
	const variants = $derived({
		creator: 'info',
		confirmed: 'success',
		added: 'warning',
		requested: 'warning',
		waitlisted: requireWaitlist ? 'outline' : 'warning',
		not_joined: 'muted'
	} as const);

	const label = $derived(state ? labels[state] : undefined);
	const variant = $derived(state ? variants[state] : 'muted');
</script>

{#if label}
	<Badge {variant}>{label}</Badge>
{/if}
