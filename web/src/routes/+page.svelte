<script lang="ts">
	import { browser } from '$app/environment';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
	import { CalendarDate } from '@internationalized/date';
	import type { DateValue } from '@internationalized/date';
	import type { PageData } from './$types';
	import Button from '$lib/components/ui/button.svelte';
	import { Combobox } from '$lib/components/ui/combobox/index';
	import { Select } from '$lib/components/ui/select/index';
	import { DateRangePicker } from '$lib/components/ui/date-range-picker/index';
	import { BADMINTON_LEVELS, TENNIS_LEVELS, SPORTS, levelIndex } from '$lib/consts/index';
	import PlaysDesktopTable from '$lib/components/plays/plays-desktop-table.svelte';
	import PlaysMobileGrid from '$lib/components/plays/plays-mobile-grid.svelte';

	type DateRange = {
		start: DateValue | undefined;
		end: DateValue | undefined;
	};

	let { data }: { data: PageData } = $props();

	let selectedSport = $state<string>(page.url.searchParams.get('sport') || '');
	let selectedVenue = $state<string>(getInitialVenue());
	let selectedDateRange = $state<DateRange>({
		start: getInitialDate('starts_after'),
		end: getInitialDate('starts_before')
	});
	let selectedLevel = $state<string>(page.url.searchParams.get('level_min') || '');
	let selectedLevelMax = $state<string>(page.url.searchParams.get('level_max') || '');
	const browserTimeZone = browser ? Intl.DateTimeFormat().resolvedOptions().timeZone : undefined;
	const today = $derived(
		new CalendarDate(new Date().getFullYear(), new Date().getMonth() + 1, new Date().getDate())
	);

	const venueItems = $derived(data.venues.map((v) => ({ value: String(v.id), label: v.name })));
	const showViewerState = $derived(
		data.plays.items?.some((play) => play.viewer_state != null) ?? false
	);

	// Disable levels above selected max for min, and below selected min for max
	const levelOptions = $derived(selectedSport === 'tennis' ? TENNIS_LEVELS : BADMINTON_LEVELS);

	const levelMinItems = $derived(
		levelOptions.map((item, idx) => {
			const maxIdx = levelIndex(levelOptions, selectedLevelMax);
			return { ...item, disabled: maxIdx !== -1 && idx > maxIdx };
		})
	);

	const levelMaxItems = $derived(
		levelOptions.map((item, idx) => {
			const minIdx = levelIndex(levelOptions, selectedLevel);
			return { ...item, disabled: minIdx !== -1 && idx < minIdx };
		})
	);

	function getInitialVenue(): string {
		const lat = page.url.searchParams.get('lat');
		const lng = page.url.searchParams.get('lng');
		if (!lat || !lng) return '';

		const latNum = Number(lat);
		const lngNum = Number(lng);
		const match = data.venues.find(
			(v) => String(v.latitude) === String(latNum) && String(v.longitude) === String(lngNum)
		);
		return match ? String(match.id) : '';
	}

	function getInitialDate(paramName: 'starts_after' | 'starts_before'): DateValue | undefined {
		const dateStr = page.url.searchParams.get(paramName);
		if (!dateStr) return undefined;
		const [y, m, d] = dateStr.split('-').map(Number);
		if (!y || !m || !d) return undefined;
		return new CalendarDate(y, m, d);
	}

	function formatDateParam(value: DateValue): string {
		return `${value.year}-${String(value.month).padStart(2, '0')}-${String(value.day).padStart(2, '0')}`;
	}

	function ensureTimezoneParam(params: SvelteURLSearchParams) {
		if (browserTimeZone && !params.get('timezone')) {
			params.set('timezone', browserTimeZone);
		}
	}

	function handleVenueChange(value: string) {
		const venue = value ? data.venues.find((v) => String(v.id) === value) : undefined;
		const params = new SvelteURLSearchParams(page.url.searchParams);
		ensureTimezoneParam(params);

		if (venue) {
			params.set('lat', String(venue.latitude));
			params.set('lng', String(venue.longitude));
		} else {
			params.delete('lat');
			params.delete('lng');
		}
		params.delete('cursor');

		// eslint-disable-next-line svelte/no-navigation-without-resolve
		goto(`?${params.toString()}`, { keepFocus: true, noScroll: true });
	}

	function handleDateRangeChange(value: DateRange) {
		selectedDateRange = value;
		const params = new SvelteURLSearchParams(page.url.searchParams);
		ensureTimezoneParam(params);
		if (value.start) {
			params.set('starts_after', formatDateParam(value.start));
		} else {
			params.delete('starts_after');
		}
		if (value.end) {
			params.set('starts_before', formatDateParam(value.end));
		} else {
			params.delete('starts_before');
		}
		params.delete('cursor');

		// eslint-disable-next-line svelte/no-navigation-without-resolve
		goto(`?${params.toString()}`, { keepFocus: true, noScroll: true });
	}

	function handleSportChange(value: string) {
		selectedSport = value;
		selectedLevel = '';
		selectedLevelMax = '';
		const params = new SvelteURLSearchParams(page.url.searchParams);
		ensureTimezoneParam(params);
		if (value) {
			params.set('sport', value);
		} else {
			params.delete('sport');
		}
		params.delete('level_min');
		params.delete('level_max');
		params.delete('cursor');

		// eslint-disable-next-line svelte/no-navigation-without-resolve
		goto(`?${params.toString()}`, { keepFocus: true, noScroll: true });
	}

	function handleLevelChange() {
		const params = new SvelteURLSearchParams(page.url.searchParams);
		ensureTimezoneParam(params);
		if (selectedLevel) {
			params.set('level_min', selectedLevel);
		} else {
			params.delete('level_min');
		}
		if (selectedLevelMax) {
			params.set('level_max', selectedLevelMax);
		} else {
			params.delete('level_max');
		}
		params.delete('cursor');

		// eslint-disable-next-line svelte/no-navigation-without-resolve
		goto(`?${params.toString()}`, { keepFocus: true, noScroll: true });
	}

	function clearFilters() {
		selectedSport = '';
		selectedVenue = '';
		selectedDateRange = { start: undefined, end: undefined };
		selectedLevel = '';
		selectedLevelMax = '';
		const params = new SvelteURLSearchParams(page.url.searchParams);
		ensureTimezoneParam(params);
		params.delete('sport');
		params.delete('lat');
		params.delete('lng');
		params.delete('starts_after');
		params.delete('starts_before');
		params.delete('level_min');
		params.delete('level_max');
		params.delete('cursor');

		// eslint-disable-next-line svelte/no-navigation-without-resolve
		goto(`?${params.toString()}`, { keepFocus: true, noScroll: true });
	}

	function getNextPageUrl(nextCursor: string): string {
		const params = new SvelteURLSearchParams(page.url.searchParams);
		ensureTimezoneParam(params);
		params.set('cursor', nextCursor);
		return `?${params.toString()}`;
	}
</script>

<h1 class="text-xl font-semibold mb-2">Plays</h1>

<div class="mb-4 flex flex-wrap gap-2 items-end">
	<div class="w-full sm:w-52">
		<Select
			type="single"
			items={SPORTS}
			bind:value={selectedSport}
			onValueChange={handleSportChange}
			placeholder="Any sport"
			label="Sport"
			allowDeselect
		/>
	</div>
	<div class="w-full sm:w-70">
		<label for="venue-filter" class="text-sm text-muted mb-1 block">Sort near venue</label>
		<Combobox
			type="single"
			items={venueItems}
			bind:value={selectedVenue}
			onValueChange={handleVenueChange}
			placeholder="Search venues"
			openOnClick
			inputProps={{ id: 'venue-filter' }}
		/>
	</div>
	<div class="w-full sm:w-72">
		<DateRangePicker
			label="Play dates"
			bind:value={selectedDateRange}
			onValueChange={handleDateRangeChange}
			minValue={today}
		/>
	</div>
	<div class="flex flex-wrap gap-2 w-full sm:flex-[0_1_22rem] sm:w-auto">
		<div class="flex-1">
			<Select
				type="single"
				items={levelMinItems}
				bind:value={selectedLevel}
				onValueChange={() => handleLevelChange()}
				placeholder="From"
				label="Level (min)"
				allowDeselect
			/>
		</div>
		<div class="flex-1">
			<Select
				type="single"
				items={levelMaxItems}
				bind:value={selectedLevelMax}
				onValueChange={() => handleLevelChange()}
				placeholder="To"
				label="Level (max)"
				allowDeselect
			/>
		</div>
	</div>
	{#if selectedSport || selectedVenue || selectedDateRange.start || selectedDateRange.end || selectedLevel || selectedLevelMax}
		<Button class="w-full sm:w-auto" variant="outline" onclick={clearFilters}>Clear filters</Button>
	{/if}
</div>

<p class="text-muted mb-1">
	Showing {data.plays.total} upcoming {data.plays.total === 1 ? 'game' : 'games'}
</p>

{#if data.plays.items && data.plays.items.length > 0}
	<PlaysMobileGrid plays={data.plays.items} {showViewerState} />
	<PlaysDesktopTable plays={data.plays.items} {showViewerState} />
{:else}
	<p>No plays match these filters.</p>
{/if}

<div class="my-6 flex gap-4 w-full">
	{#if page.url.searchParams.has('cursor')}
		<Button variant="outline" onclick={() => history.back()}>Previous</Button>
	{/if}
	{#if data.plays.has_more && data.plays.next_cursor != null}
		<Button
			class="ms-auto"
			variant="outline"
			href={getNextPageUrl(data.plays.next_cursor)}
			data-sveltekit-noscroll>Next</Button
		>
	{/if}
</div>
