<script lang="ts">
	import { page } from '$app/state';
	import { enhance } from '$app/forms';
	import { FormField, FormLabel, TextInput } from '$lib/components/ui/form';
	import { Select } from '$lib/components/ui/select/index';
	import { TennisSlider } from '$lib/components/ui/slider/index';
	import { Combobox } from '$lib/components/ui/combobox/index';
	import { DatePicker } from '$lib/components/ui/date-picker/index';
	import Button from '$lib/components/ui/button.svelte';
	import { BADMINTON_LEVELS, SPORTS, GAME_TYPES, DURATIONS } from '$lib/consts/index';
	import { CalendarDate } from '@internationalized/date';
	import type { DateValue } from '@internationalized/date';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const venueItems = $derived(data.venues.map((v) => ({ value: v.name, label: v.name })));

	let selectedSport = $state('');
	let selectedVenue = $state('');
	let selectedGameType = $state('');
	let selectedLevelMin = $state('');
	let selectedLevelMax = $state('');
	let tennisRange = $state<number[]>([3, 4]);
	let lastSport = $state('');

	let selectedDate = $state<DateValue | undefined>(undefined);
	let startTime = $state('');
	let selectedDuration = $state('120'); // default 2 hours

	const today = $derived(
		new CalendarDate(new Date().getFullYear(), new Date().getMonth() + 1, new Date().getDate())
	);

	const TZ_OFFSET = '+08:00';

	// Derive date string from DateValue
	let dateStr = $derived(
		selectedDate
			? `${selectedDate.year}-${String(selectedDate.month).padStart(2, '0')}-${String(selectedDate.day).padStart(2, '0')}`
			: ''
	);

	let startsAt = $derived(dateStr && startTime ? `${dateStr}T${startTime}:00${TZ_OFFSET}` : '');

	const tennisLevelMin = $derived(tennisRange[0]?.toFixed(1) ?? '');
	const tennisLevelMax = $derived(tennisRange[1]?.toFixed(1) ?? '');

	$effect(() => {
		if (!selectedSport) {
			return;
		}
		if (selectedSport !== lastSport) {
			selectedLevelMin = '';
			selectedLevelMax = '';
			tennisRange = [3, 4];
			lastSport = selectedSport;
		}
	});
</script>

<div class="mx-auto mt-8 max-w-lg w-full">
	<h1 class="text-xl text-foreground font-bold mb-6">Create Game</h1>

	<form method="POST" use:enhance class="flex flex-col gap-4">
		<!-- Hidden computed fields -->
		<input type="hidden" name="starts_at" value={startsAt} />
		<input type="hidden" name="duration_minutes" value={selectedDuration} />
		<input type="hidden" name="timezone" value="Asia/Singapore" />

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
		<div>
			<input type="hidden" name="venue" value={selectedVenue} />
			<FormLabel for="venue-input" required>Venue</FormLabel>
			<Combobox
				type="single"
				items={venueItems}
				bind:value={selectedVenue}
				placeholder="Search or type venue name"
				openOnClick
				allowCustom
				inputProps={{ id: 'venue-input' }}
			/>
		</div>

		<!-- Date -->
		<div>
			<DatePicker label="Date" bind:value={selectedDate} minValue={today} required />
		</div>

		<!-- Start Time / Duration -->
		<div class="gap-4 grid grid-cols-2">
			<FormField label="Start Time" id="start_time" required>
				<input
					id="start_time"
					name="start_time"
					type="time"
					bind:value={startTime}
					required
					class="text-sm text-foreground px-3 border border-input-border rounded-lg bg-input h-9 w-full placeholder:text-muted-foreground focus:outline-none focus:border-ring"
				/>
			</FormField>

			<div>
				<Select
					type="single"
					items={DURATIONS}
					bind:value={selectedDuration}
					placeholder="Duration"
					label="Duration"
					required
				/>
			</div>
		</div>

		<!-- Game Type -->
		<div>
			<input type="hidden" name="game_type" value={selectedGameType} />
			<Select
				type="single"
				items={GAME_TYPES}
				bind:value={selectedGameType}
				placeholder="Select game type…"
				label="Game Type"
				allowDeselect
			/>
		</div>

		<!-- Level Min / Max -->
		{#if selectedSport === 'badminton'}
			<div class="gap-4 grid grid-cols-2">
				<div>
					<input type="hidden" name="level_min" value={selectedLevelMin} />
					<Select
						type="single"
						items={BADMINTON_LEVELS}
						bind:value={selectedLevelMin}
						placeholder="Min level…"
						label="Level Min"
						allowDeselect
					/>
				</div>

				<div>
					<input type="hidden" name="level_max" value={selectedLevelMax} />
					<Select
						type="single"
						items={BADMINTON_LEVELS}
						bind:value={selectedLevelMax}
						placeholder="Max level…"
						label="Level Max"
						allowDeselect
					/>
				</div>
			</div>
		{:else if selectedSport === 'tennis'}
			<div>
				<input type="hidden" name="level_min" value={tennisLevelMin} />
				<input type="hidden" name="level_max" value={tennisLevelMax} />
				<TennisSlider bind:value={tennisRange} label="Range restriction" />
			</div>
		{/if}

		<!-- Fee -->
		<FormField label="Fee ($)" id="fee">
			<TextInput id="fee" name="fee" type="number" step="0.01" placeholder="0.00" />
		</FormField>

		<!-- Max Players -->
		<FormField label="Max Players" id="max_players" required>
			<TextInput id="max_players" name="max_players" type="number" min="1" step="1" required />
		</FormField>

		<!-- Courts -->
		<FormField label="Courts" id="courts">
			<TextInput id="courts" name="courts" type="number" />
		</FormField>

		{#if page.form?.error}
			<p class="text-sm text-destructive">{page.form.error}</p>
		{/if}

		<Button type="submit" class="mt-4 w-full">Create Game</Button>
	</form>
</div>
