<script lang="ts">
	import { onDestroy } from 'svelte';
	import { FormLabel } from '$lib/components/ui/form';
	import {
		ComboboxContent,
		ComboboxInput,
		ComboboxPrimitive,
		ComboboxRoot
	} from '$lib/components/ui/combobox';
	import type { components } from '$lib/api/types.gen';

	type VenuePublic = components['schemas']['VenuePublic'];
	type VenueSearchItem = components['schemas']['VenueSearchItem'];
	type VenueComboboxItem = {
		value: string;
		label: string;
		item: VenueSearchItem;
		disabled?: boolean;
	};
	type VenueSearchCache = Record<string, VenueSearchItem[] | undefined>;

	type Props = {
		value?: string;
		venueId?: string;
	};

	const venueSearchCache: VenueSearchCache = {};

	let { value = $bindable(''), venueId = $bindable('') }: Props = $props();

	let inputValue = $state(value);
	let selectedValue = $state('');
	let open = $state(false);
	let pendingSearch = $state(false);
	let loading = $state(false);
	let resolvingPlaceID = $state('');
	let errorMessage = $state('');
	let items = $state<VenueSearchItem[]>([]);
	let searchedQuery = $state('');
	let sessionToken = $state(newSessionToken());
	const normalizedInputQuery = $derived(normalizeVenueSearchQuery(inputValue));
	const hasSearchQuery = $derived(normalizedInputQuery.length >= 2);
	const hasSearchedCurrentQuery = $derived(searchedQuery === normalizedInputQuery);
	const isSearching = $derived(pendingSearch || loading || !hasSearchedCurrentQuery);
	const comboboxItems = $derived.by(() =>
		items.map((item, index) => ({
			value: itemValue(item, index),
			label: item.name,
			item,
			disabled: resolvingPlaceID !== '' && resolvingPlaceID === item.google_place_id
		}))
	);

	let searchTimer: ReturnType<typeof setTimeout> | undefined;
	let abortController: AbortController | undefined;
	let activeSearchKey = '';

	function newSessionToken() {
		return crypto.randomUUID();
	}

	function normalizeVenueSearchQuery(query: string) {
		return query.trim().replace(/\s+/g, ' ').toLowerCase();
	}

	function venueSearchCacheKey(query: string, token: string) {
		return `${token}:${query}`;
	}

	function cachedVenueSearch(query: string, token: string) {
		return venueSearchCache[venueSearchCacheKey(query, token)];
	}

	function cacheVenueSearch(query: string, token: string, nextItems: VenueSearchItem[]) {
		venueSearchCache[venueSearchCacheKey(query, token)] = nextItems;
	}

	function applySearchResults(query: string, nextItems: VenueSearchItem[]) {
		items = nextItems;
		searchedQuery = query;
		pendingSearch = false;
		loading = false;
		errorMessage = '';
	}

	function itemValue(item: VenueSearchItem, index: number) {
		if (item.id != null) {
			return `venue-${item.id}`;
		}
		return `suggestion-${item.google_place_id ?? `${item.name}-${index}`}`;
	}

	function venueToSearchItem(venue: VenuePublic): VenueSearchItem {
		return {
			id: venue.id,
			name: venue.name,
			address: venue.address,
			postal_code: venue.postal_code,
			latitude: venue.latitude,
			longitude: venue.longitude,
			google_place_id: venue.google_place_id
		};
	}

	function needsResolve(item: VenueSearchItem) {
		return item.id == null && Boolean(item.google_place_id);
	}

	function scheduleSearch(query: string) {
		if (searchTimer) clearTimeout(searchTimer);
		const normalizedQuery = normalizeVenueSearchQuery(query);
		if (normalizedQuery.length < 2) {
			abortController?.abort();
			activeSearchKey = '';
			pendingSearch = false;
			loading = false;
			open = false;
			items = [];
			searchedQuery = '';
			return;
		}
		const cached = cachedVenueSearch(normalizedQuery, sessionToken);
		if (cached) {
			abortController?.abort();
			activeSearchKey = '';
			applySearchResults(normalizedQuery, cached);
			return;
		}
		pendingSearch = true;
		errorMessage = '';
		searchTimer = setTimeout(() => {
			void searchVenues(query);
		}, 250);
	}

	async function searchVenues(query: string) {
		const normalizedQuery = normalizeVenueSearchQuery(query);
		const searchSessionToken = sessionToken;
		const searchKey = venueSearchCacheKey(normalizedQuery, searchSessionToken);
		const cached = cachedVenueSearch(normalizedQuery, searchSessionToken);
		if (cached) {
			applySearchResults(normalizedQuery, cached);
			return;
		}
		if (activeSearchKey === searchKey && loading) {
			pendingSearch = false;
			return;
		}

		abortController?.abort();
		const controller = new AbortController();
		abortController = controller;
		activeSearchKey = searchKey;
		pendingSearch = false;
		loading = true;
		errorMessage = '';

		const params = new URLSearchParams({
			q: query.trim(),
			session_token: searchSessionToken,
			limit: '5'
		});

		try {
			const response = await fetch(`/create/venues?${params}`, { signal: controller.signal });
			if (!response.ok) {
				throw new Error('Failed to search venues');
			}
			const data = (await response.json()) as { items?: VenueSearchItem[] };
			const nextItems = data.items ?? [];
			cacheVenueSearch(normalizedQuery, searchSessionToken, nextItems);
			applySearchResults(normalizedQuery, nextItems);
		} catch (err) {
			if ((err as DOMException).name !== 'AbortError') {
				errorMessage = 'Failed to search venues';
				items = [];
				searchedQuery = normalizedQuery;
			}
		} finally {
			if (abortController === controller) {
				loading = false;
				activeSearchKey = '';
			}
		}
	}

	function handleInput(event: Event & { currentTarget: HTMLInputElement }) {
		inputValue = event.currentTarget.value;
		value = inputValue;
		venueId = '';
		selectedValue = '';
		errorMessage = '';
		open = inputValue.trim().length >= 2;
		scheduleSearch(inputValue);
	}

	function handleFocus() {
		if (normalizedInputQuery.length >= 2) {
			open = true;
			if (!hasSearchedCurrentQuery || errorMessage) {
				scheduleSearch(inputValue);
			}
		} else {
			open = false;
			items = [];
		}
	}

	function handleOpenChange(nextOpen: boolean) {
		if (nextOpen) {
			handleFocus();
			return;
		}
		commitTypedValue();
	}

	function commitTypedValue() {
		inputValue = inputValue.trim();
		value = inputValue;
		if (inputValue === '') {
			venueId = '';
			selectedValue = '';
		}
	}

	function handleValueChange(nextValue: string | string[]) {
		if (Array.isArray(nextValue) || nextValue === '') {
			return;
		}
		const selected = comboboxItems.find((item) => item.value === nextValue);
		if (!selected) {
			return;
		}
		void selectItem(selected);
	}

	async function selectItem(selected: VenueComboboxItem) {
		errorMessage = '';
		selectedValue = selected.value;
		const item = selected.item;
		if (needsResolve(item)) {
			await resolveGoogleVenue(item, selected.value);
			return;
		}

		value = item.name;
		venueId = item.id ? String(item.id) : '';
		inputValue = item.name;
		searchedQuery = normalizeVenueSearchQuery(item.name);
		open = false;
	}

	async function resolveGoogleVenue(item: VenueSearchItem, fallbackValue: string) {
		if (!item.google_place_id) {
			return;
		}

		resolvingPlaceID = item.google_place_id;
		try {
			const response = await fetch('/create/venues', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					google_place_id: item.google_place_id,
					session_token: sessionToken,
					query: inputValue
				})
			});
			if (!response.ok) {
				throw new Error('Failed to resolve venue');
			}
			const venue = (await response.json()) as VenuePublic;
			const resolved = venueToSearchItem(venue);
			items = [
				resolved,
				...items.filter((existing) => existing.google_place_id !== venue.google_place_id)
			];
			value = venue.name;
			venueId = String(venue.id);
			inputValue = venue.name;
			selectedValue = `venue-${venue.id}`;
			searchedQuery = normalizeVenueSearchQuery(venue.name);
			open = false;
			sessionToken = newSessionToken();
		} catch {
			errorMessage = 'Failed to add venue';
			selectedValue = fallbackValue;
			open = true;
		} finally {
			resolvingPlaceID = '';
		}
	}

	onDestroy(() => {
		if (searchTimer) clearTimeout(searchTimer);
		abortController?.abort();
		activeSearchKey = '';
	});
</script>

<div>
	<input type="hidden" name="venue" {value} />
	<input type="hidden" name="venue_id" value={venueId} />
	<FormLabel for="venue-input" required>Venue</FormLabel>
	<ComboboxRoot
		type="single"
		items={comboboxItems}
		{inputValue}
		bind:value={selectedValue}
		bind:open
		onOpenChange={handleOpenChange}
		onValueChange={handleValueChange}
	>
		<div class="relative">
			<ComboboxInput
				id="venue-input"
				required
				placeholder="Search or type venue name"
				oninput={handleInput}
				onfocus={handleFocus}
			/>
		</div>
		{#if hasSearchQuery}
			<ComboboxContent>
				{#if isSearching}
					<p class="text-sm text-muted-foreground px-2 py-1.5">Searching...</p>
				{:else if errorMessage}
					<p class="text-sm text-destructive px-2 py-1.5">{errorMessage}</p>
				{:else if items.length > 0}
					{#each comboboxItems as option (option.value)}
						<ComboboxPrimitive.Item
							value={option.value}
							label={option.label}
							disabled={option.disabled}
							class="px-2 py-2 text-left outline-none rounded-md flex gap-3 w-full cursor-default select-none items-start data-[highlighted]:bg-accent data-[disabled]:opacity-50 data-[disabled]:pointer-events-none"
						>
							<span class="flex-1 min-w-0">
								<span class="text-sm text-foreground font-medium block truncate">
									{option.item.name}
								</span>
								{#if option.item.address}
									<span class="text-xs text-muted-foreground block truncate">
										{option.item.address}
									</span>
								{/if}
							</span>
							{#if needsResolve(option.item) && resolvingPlaceID === option.item.google_place_id}
								<span class="text-[11px] text-muted-foreground pt-0.5 shrink-0"> Adding </span>
							{/if}
						</ComboboxPrimitive.Item>
					{/each}
				{:else}
					<p class="text-sm text-muted-foreground px-2 py-1.5">No results found.</p>
				{/if}
			</ComboboxContent>
		{/if}
	</ComboboxRoot>
</div>
