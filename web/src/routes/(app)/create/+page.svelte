<script lang="ts">
	import { enhance } from '$app/forms';
	import PlayFormFields from '$lib/components/plays/play-form-fields.svelte';
	import VenueSearchField from '$lib/components/plays/venue-search-field.svelte';
	import { Select } from '$lib/components/ui/select/index';
	import Button from '$lib/components/ui/button.svelte';
	import { SPORTS } from '$lib/consts/index';
	import type { ActionData } from './$types';

	type CreateFormValues = {
		sport?: string;
		venue?: string;
		venue_id?: string;
		name?: string;
		description?: string;
		visibility?: string;
		date?: string;
		start_time?: string;
		starts_at?: string;
		duration_minutes?: string;
		game_type?: string;
		level_min?: string;
		level_max?: string;
		fee?: string;
		max_players?: string;
		courts?: string;
		require_waitlist?: string;
	};

	let { form }: { form?: ActionData } = $props();

	// Seed once from the failed submission, if any; live inputs keep their own state
	function initialFormValues(): CreateFormValues | undefined {
		return form?.values;
	}
	const initialValues = initialFormValues();

	let selectedSport = $state(initialValues?.sport ?? '');
	let selectedVenue = $state(initialValues?.venue ?? '');
	let selectedVenueID = $state(initialValues?.venue_id ?? '');
</script>

<div class="mx-auto mt-8 max-w-lg w-full">
	<h1 class="text-xl text-foreground font-bold mb-6">Create Game</h1>

	<form method="POST" use:enhance class="flex flex-col gap-4">
		<!-- Sport -->
		<div>
			<input type="hidden" name="sport" value={selectedSport} />
			<Select
				type="single"
				items={SPORTS}
				bind:value={selectedSport}
				placeholder="Select sport…"
				label="Sport"
				required
			/>
		</div>

		<!-- Venue -->
		<VenueSearchField bind:value={selectedVenue} bind:venueId={selectedVenueID} />

		<PlayFormFields sport={selectedSport} values={initialValues ?? {}} />

		{#if form?.error}
			<p class="text-sm text-destructive">{form.error}</p>
		{/if}

		<Button type="submit" class="mt-4 w-full">Create Game</Button>
	</form>
</div>
