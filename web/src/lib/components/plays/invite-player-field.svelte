<script module lang="ts">
	import type { components } from '$lib/api/types.gen';

	type UserSummary = components['schemas']['UserSummary'];

	const inviteSearchCache: Record<string, UserSummary[] | undefined> = {};
</script>

<script lang="ts">
	import { enhance } from '$app/forms';
	import { onDestroy } from 'svelte';
	import type { SubmitFunction } from '@sveltejs/kit';
	import * as Dialog from '$lib/components/ui/dialog/index';
	import {
		ComboboxContent,
		ComboboxInput,
		ComboboxPrimitive,
		ComboboxRoot
	} from '$lib/components/ui/combobox';
	import { FormLabel } from '$lib/components/ui/form';
	import { RatingBadge } from '$lib/components/ui/badge/index';
	import Button from '$lib/components/ui/button.svelte';
	import UserAvatar from '$lib/components/ui/avatar/user-avatar.svelte';
	import type { operations } from '$lib/api/types.gen';

	type UserSummary = components['schemas']['UserSummary'];
	type SearchQuery = NonNullable<operations['search-users']['parameters']['query']>;
	type SearchSport = Exclude<NonNullable<SearchQuery['sport']>, ''>;

	let {
		playId,
		sport,
		openSlots
	}: {
		playId: string;
		sport: SearchSport;
		openSlots: number;
	} = $props();

	let inputValue = $state('');
	let selectedValue = $state('');
	let selectedUser = $state<UserSummary | null>(null);
	let dialogOpen = $state(false);
	let comboboxOpen = $state(false);
	let pendingSearch = $state(false);
	let loading = $state(false);
	let errorMessage = $state('');
	let items = $state<UserSummary[]>([]);
	let searchedQuery = $state('');
	let searchTimer: ReturnType<typeof setTimeout> | undefined;
	let abortController: AbortController | undefined;
	let activeSearchKey = '';

	const normalizedInputQuery = $derived(normalizeSearchQuery(inputValue));
	const hasSearchQuery = $derived(normalizedInputQuery.length >= 1);
	const hasSearchedCurrentQuery = $derived(searchedQuery === normalizedInputQuery);
	const isSearching = $derived(pendingSearch || loading || !hasSearchedCurrentQuery);
	const comboboxItems = $derived.by(() =>
		items.map((item) => ({
			value: item.id,
			label: item.display_name,
			item
		}))
	);

	function normalizeSearchQuery(query: string) {
		return query.trim().replace(/\s+/g, ' ').toLowerCase();
	}

	function cacheKey(query: string) {
		return `${playId}:${sport}:${query}`;
	}

	function cachedResults(query: string) {
		return inviteSearchCache[cacheKey(query)];
	}

	function applyResults(query: string, nextItems: UserSummary[]) {
		items = nextItems;
		searchedQuery = query;
		pendingSearch = false;
		loading = false;
		errorMessage = '';
	}

	function scheduleSearch(query: string) {
		if (searchTimer) clearTimeout(searchTimer);
		const normalizedQuery = normalizeSearchQuery(query);
		if (normalizedQuery.length < 1) {
			abortController?.abort();
			activeSearchKey = '';
			pendingSearch = false;
			loading = false;
			comboboxOpen = false;
			items = [];
			searchedQuery = '';
			return;
		}

		const cached = cachedResults(normalizedQuery);
		if (cached) {
			abortController?.abort();
			activeSearchKey = '';
			applyResults(normalizedQuery, cached);
			return;
		}

		pendingSearch = true;
		errorMessage = '';
		comboboxOpen = true;
		searchTimer = setTimeout(() => {
			void searchUsers(query);
		}, 250);
	}

	async function searchUsers(query: string) {
		const normalizedQuery = normalizeSearchQuery(query);
		const searchKey = cacheKey(normalizedQuery);
		const cached = cachedResults(normalizedQuery);
		if (cached) {
			applyResults(normalizedQuery, cached);
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
			sport,
			limit: '10'
		});

		try {
			const response = await fetch(
				`/play/${encodeURIComponent(playId)}/invite-users?${params.toString()}`,
				{ signal: controller.signal }
			);
			if (!response.ok) throw new Error('Failed to search users');
			const data = (await response.json()) as { items?: UserSummary[] };
			const nextItems = data.items ?? [];
			inviteSearchCache[searchKey] = nextItems;
			applyResults(normalizedQuery, nextItems);
		} catch (err) {
			if ((err as DOMException).name !== 'AbortError') {
				errorMessage = 'Failed to search users';
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
		selectedValue = '';
		selectedUser = null;
		errorMessage = '';
		scheduleSearch(inputValue);
	}

	function handleFocus() {
		if (normalizedInputQuery.length >= 1) {
			comboboxOpen = true;
			if (!hasSearchedCurrentQuery || errorMessage) scheduleSearch(inputValue);
		} else {
			comboboxOpen = false;
			items = [];
		}
	}

	function handleValueChange(nextValue: string | string[]) {
		if (Array.isArray(nextValue) || nextValue === '') return;
		const selected = comboboxItems.find((item) => item.value === nextValue);
		if (!selected) return;
		selectedValue = selected.value;
		selectedUser = selected.item;
		inputValue = selected.item.display_name;
		searchedQuery = normalizeSearchQuery(inputValue);
		comboboxOpen = false;
	}

	function resetPicker() {
		inputValue = '';
		selectedValue = '';
		selectedUser = null;
		comboboxOpen = false;
		errorMessage = '';
		items = [];
		searchedQuery = '';
	}

	const handleInviteSubmit: SubmitFunction = () => {
		return async ({ result, update }) => {
			if (result.type === 'success' || result.type === 'redirect') {
				dialogOpen = false;
				resetPicker();
			}
			await update();
		};
	};

	onDestroy(() => {
		if (searchTimer) clearTimeout(searchTimer);
		abortController?.abort();
		activeSearchKey = '';
	});
</script>

<Dialog.Root bind:open={dialogOpen}>
	<Dialog.Trigger>
		{#snippet child({ props })}
			<Button type="button" size="sm" variant="outline" {...props}>Invite players</Button>
		{/snippet}
	</Dialog.Trigger>
	<Dialog.Content class="border border-border shadow-card/30 shadow-lg">
		<Dialog.Header>
			<Dialog.Title>Invite a player</Dialog.Title>
			<Dialog.Description>Search for a player to invite to this game.</Dialog.Description>
		</Dialog.Header>

		{#if openSlots <= 0}
			<p class="text-sm text-amber-700 px-3 py-2 border border-amber-200 rounded-md bg-amber-50">
				The game is full. If they accept, they will be added to the waitlist.
			</p>
		{/if}

		<form method="POST" action="?/invitePlayer" use:enhance={handleInviteSubmit} class="space-y-4">
			<input type="hidden" name="user_id" value={selectedUser?.id ?? ''} />
			<div>
				<FormLabel for="invite-player-input" required>Player</FormLabel>
				<ComboboxRoot
					type="single"
					items={comboboxItems}
					{inputValue}
					bind:value={selectedValue}
					bind:open={comboboxOpen}
					onOpenChange={(nextOpen) => (comboboxOpen = nextOpen)}
					onValueChange={handleValueChange}
				>
					<div class="relative">
						<ComboboxInput
							id="invite-player-input"
							placeholder="Search by name or username"
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
										class="px-2 py-2 text-left outline-none rounded-md flex gap-3 w-full cursor-default select-none items-center data-[highlighted]:bg-accent"
									>
										<UserAvatar
											src={option.item.photo_url}
											nameForFallback={option.item.display_name}
											className="h-9 w-9 text-xs"
										/>
										<span class="flex-1 min-w-0">
											<span class="text-sm text-foreground font-medium block truncate">
												{option.item.display_name}
											</span>
											<span class="text-xs text-muted-foreground block truncate">
												{option.item.username ? `@${option.item.username}` : 'No username'}
											</span>
										</span>
										{#if option.item.rating_code}
											<RatingBadge value={option.item.rating_code} />
										{/if}
									</ComboboxPrimitive.Item>
								{/each}
							{:else}
								<p class="text-sm text-muted-foreground px-2 py-1.5">No players found.</p>
							{/if}
						</ComboboxContent>
					{/if}
				</ComboboxRoot>
			</div>

			<Dialog.Footer showCloseButton>
				<Button type="submit" disabled={selectedUser == null}>Send invite</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
