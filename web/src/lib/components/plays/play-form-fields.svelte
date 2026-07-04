<script lang="ts">
	import { CalendarDate } from '@internationalized/date';
	import type { DateValue } from '@internationalized/date';
	import { FormField, TextInput } from '$lib/components/ui/form';
	import { Select } from '$lib/components/ui/select/index';
	import { TennisSlider } from '$lib/components/ui/slider/index';
	import { DatePicker } from '$lib/components/ui/date-picker/index';
	import { BADMINTON_LEVELS, DURATIONS, GAME_TYPES } from '$lib/consts/index';
	import PlayFormSettings from './play-form-settings.svelte';

	type PlayFormFieldsValues = {
		name?: string;
		description?: string;
		date?: string;
		start_time?: string;
		duration_minutes?: string;
		game_type?: string;
		level_min?: string;
		level_max?: string;
		fee?: string;
		max_players?: string;
		courts?: string;
		visibility?: string;
		require_waitlist?: string;
	};

	let {
		sport,
		values = {},
		minPlayers = 1,
		timezone = 'Asia/Singapore',
		tzOffset = '+08:00'
	}: {
		sport: string;
		values?: PlayFormFieldsValues;
		minPlayers?: number;
		timezone?: string;
		tzOffset?: string;
	} = $props();

	// Seed field state once from the provided values; the inputs own it afterwards
	const seed = seedValues();
	function seedValues() {
		return { ...values, sport };
	}

	let selectedDate = $state<DateValue | undefined>(calendarDateFromString(seed.date));
	let startTime = $state(seed.start_time ?? '');
	let selectedDuration = $state(seed.duration_minutes || '120');
	let selectedGameType = $state(seed.game_type ?? '');
	let selectedLevelMin = $state(seed.level_min ?? '');
	let selectedLevelMax = $state(seed.level_max ?? '');
	let tennisRange = $state<number[]>(tennisRangeFromValues(seed));
	let requireWaitlist = $state(seed.require_waitlist === 'true');
	let unlisted = $state(seed.visibility === 'unlisted');
	let lastSport = $state(seed.sport);

	const today = $derived(
		new CalendarDate(new Date().getFullYear(), new Date().getMonth() + 1, new Date().getDate())
	);

	const dateStr = $derived(
		selectedDate
			? `${selectedDate.year}-${String(selectedDate.month).padStart(2, '0')}-${String(selectedDate.day).padStart(2, '0')}`
			: ''
	);
	const startsAt = $derived(dateStr && startTime ? `${dateStr}T${startTime}:00${tzOffset}` : '');

	const tennisLevelMin = $derived(tennisRange[0]?.toFixed(1) ?? '');
	const tennisLevelMax = $derived(tennisRange[1]?.toFixed(1) ?? '');

	// An existing play may have a duration outside the standard options
	const durationItems = $derived(
		DURATIONS.some((item) => item.value === selectedDuration)
			? DURATIONS
			: [{ value: selectedDuration, label: `${selectedDuration} min` }, ...DURATIONS]
	);

	// Level ranges are sport-specific: reset them when the sport changes
	$effect(() => {
		if (!sport) {
			return;
		}
		if (sport !== lastSport) {
			selectedLevelMin = '';
			selectedLevelMax = '';
			tennisRange = [3, 4];
			lastSport = sport;
		}
	});

	function calendarDateFromString(value: string | undefined) {
		if (!value) return undefined;
		const [year, month, day] = value.split('-').map(Number);
		if (!year || !month || !day) return undefined;
		return new CalendarDate(year, month, day);
	}

	function tennisRangeFromValues(seed: PlayFormFieldsValues) {
		const min = Number(seed.level_min);
		const max = Number(seed.level_max);
		return Number.isFinite(min) && Number.isFinite(max) ? [min, max] : [3, 4];
	}
</script>

<!-- Shared field set for the create and edit play forms. Must render inside the
     <form>: hidden inputs bridge component state into the POST body. -->
<input type="hidden" name="date" value={dateStr} />
<input type="hidden" name="starts_at" value={startsAt} />
<input type="hidden" name="tz_offset" value={tzOffset} />
<input type="hidden" name="timezone" value={timezone} />
<input type="hidden" name="duration_minutes" value={selectedDuration} />
<input type="hidden" name="game_type" value={selectedGameType} />

<!-- Name -->
<FormField label="Name" id="name">
	<TextInput id="name" name="name" maxlength={80} value={values.name ?? ''} />
</FormField>

<!-- Description -->
<FormField label="Description" id="description">
	<textarea
		id="description"
		name="description"
		maxlength={1000}
		rows="3"
		class="text-sm text-foreground px-3 py-2 border border-input-border rounded-lg bg-input min-h-20 w-full placeholder:text-muted-foreground focus:outline-none focus:border-ring"
		>{values.description ?? ''}</textarea
	>
</FormField>

<!-- Date -->
<div>
	<DatePicker label="Date" bind:value={selectedDate} minValue={today} required />
</div>

<!-- Start Time / Duration -->
<div class="gap-4 grid grid-cols-2">
	<FormField label="Start Time" id="start_time" required>
		<TextInput id="start_time" name="start_time" type="time" bind:value={startTime} required />
	</FormField>

	<div>
		<Select
			type="single"
			items={durationItems}
			bind:value={selectedDuration}
			placeholder="Duration"
			label="Duration"
			required
		/>
	</div>
</div>

<!-- Game Type -->
<div>
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
{#if sport === 'badminton'}
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
{:else if sport === 'tennis'}
	<div>
		<input type="hidden" name="level_min" value={tennisLevelMin} />
		<input type="hidden" name="level_max" value={tennisLevelMax} />
		<TennisSlider bind:value={tennisRange} label="Level restriction" />
	</div>
{:else}
	<input type="hidden" name="level_min" value="" />
	<input type="hidden" name="level_max" value="" />
{/if}

<!-- Fee -->
<FormField label="Fee ($)" id="fee">
	<TextInput
		id="fee"
		name="fee"
		type="number"
		min="0"
		step="0.01"
		placeholder="0.00"
		value={values.fee ?? ''}
	/>
</FormField>

<!-- Max Players -->
<FormField label="Max Players" id="max_players" required>
	<TextInput
		id="max_players"
		name="max_players"
		type="number"
		min={minPlayers}
		step="1"
		value={values.max_players ?? ''}
		required
	/>
</FormField>

<!-- Courts -->
<FormField label="Courts" id="courts">
	<TextInput id="courts" name="courts" type="number" min="1" step="1" value={values.courts ?? ''} />
</FormField>

<PlayFormSettings bind:requireWaitlist bind:unlisted />
