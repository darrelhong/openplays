<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
	import type { PageData } from './$types';
	import Button from '$lib/components/button.svelte';
	import * as Dialog from '$lib/components/dialog/index';
	import { Combobox } from '$lib/components/combobox/index';
	import {
		capitalize,
		formatDate,
		formatTime,
		formatDateTime,
		formatPlayFee,
		formatLevel
	} from '$lib/utils/formatting';

	let { data }: { data: PageData } = $props();

	let selectedVenue = $state<string>(getInitialVenue());

	const venueItems = $derived(data.venues.map((v) => ({ value: String(v.id), label: v.name })));

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

	function handleVenueChange(value: string) {
		const venue = value ? data.venues.find((v) => String(v.id) === value) : undefined;
		const params = new SvelteURLSearchParams(page.url.searchParams);

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

	function resetVenue() {
		selectedVenue = '';
		handleVenueChange('');
	}

	function getNextPageUrl(nextCursor: string): string {
		const params = new SvelteURLSearchParams(page.url.searchParams);
		params.set('cursor', nextCursor);
		return `?${params.toString()}`;
	}
</script>

<h1 class="text-xl font-semibold mb-2">Plays</h1>

{#if data.plays.items && data.plays.items.length > 0}
	<div class="mb-4 flex gap-2 items-end">
		<div class="w-70">
			<label for="venue-filter" class="text-sm text-stone-400 mb-1 block">Sort by distance</label>
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
		{#if selectedVenue}
			<Button variant="outline" onclick={resetVenue}>Reset</Button>
		{/if}
	</div>

	<p class="text-stone-300 mb-1">Showing {data.plays.total} plays</p>

	<div class="overflow-auto">
		<table class="w-full border-collapse">
			<thead>
				<tr
					class="text-stone-400 border-b border-neutral-500 *:font-medium *:p-2 *:text-start *:whitespace-nowrap"
				>
					<th>Venue</th>
					<th>Date</th>
					<th>Time</th>
					<th>Host</th>
					<th>Sport</th>
					<th>Level</th>
					<th>Fee</th>
					<th>Slots/Type</th>
					<th>Details</th>
				</tr>
			</thead>
			<tbody>
				{#each data.plays.items as play (play.id)}
					<tr class="border-b border-neutral-700 *:p-2 hover:bg-stone-800 *:whitespace-nowrap">
						<td>{play.venue_name || play.venue}</td>
						<td>{formatDate(play.starts_at, play.timezone)}</td>
						<td
							>{formatTime(play.starts_at, play.timezone)} - {formatTime(
								play.ends_at,
								play.timezone
							)}</td
						>
						<td>{play.host_name}</td>
						<td>{capitalize(play.sport)}</td>
						<td>{formatLevel(play.level_min, play.level_max)}</td>
						<td>{formatPlayFee(play)}</td>
						<td
							>{#if play.listing_type === 'sell_booking'}To let go{:else}{play.slots_left ?? '-'} / {play.max_players ??
									'-'}{/if}</td
						>
						<td>
							<Dialog.Root>
								<Dialog.Trigger>
									{#snippet child({ props })}
										<Button {...props} size="xs" variant="outline">View</Button>
									{/snippet}
								</Dialog.Trigger>
								<Dialog.Content variant="right">
									<Dialog.Header>
										<Dialog.Title class="text-xl pe-6">{play.venue_name || play.venue}</Dialog.Title
										>
										<p class="text-lg mb-4">
											{formatDate(play.starts_at, play.timezone, { year: 'numeric' })} · {formatTime(
												play.starts_at,
												play.timezone
											)} - {formatTime(play.ends_at, play.timezone)}
										</p>
									</Dialog.Header>
									<dl class="text-sm space-y-2">
										<div class="flex gap-4">
											<dt class="text-stone-400 w-24">Host</dt>
											<dd>{play.host_name}</dd>
										</div>
										<div class="flex gap-4">
											<dt class="text-stone-400 w-24">Sport</dt>
											<dd>
												{capitalize(play.sport)}{play.game_type
													? ` · ${capitalize(play.game_type)}`
													: ''}
											</dd>
										</div>
										<div class="flex gap-4">
											<dt class="text-stone-400 w-24">Level</dt>
											<dd>{formatLevel(play.level_min, play.level_max)}</dd>
										</div>
										<div class="flex gap-4">
											<dt class="text-stone-400 w-24">Fee</dt>
											<dd>{formatPlayFee(play)}</dd>
										</div>
										<div class="flex gap-4">
											<dt class="text-stone-400 w-24">Slots</dt>
											<dd>{play.slots_left ?? '-'} / {play.max_players ?? '-'}</dd>
										</div>
										{#if play.courts != null}
											<div class="flex gap-4">
												<dt class="text-stone-400 w-24">Courts</dt>
												<dd>{play.courts}</dd>
											</div>
										{/if}
									</dl>
									{#if play.contacts?.length}
										<div class="mt-3 pt-3 border-t border-stone-700">
											<p class="text-stone-300 mb-2">Contacts</p>
											{#each play.contacts as contact (`${contact.type}:${contact.value}`)}
												<div class="text-sm flex gap-4">
													<dt class="text-stone-400 shrink-0 w-24">{contact.type}</dt>
													<dd>{contact.value}</dd>
												</div>
											{/each}
										</div>
									{/if}
									{@const meta = play.meta ?? {}}
									{@const knownMetaKeys = [
										'fee_male',
										'fee_female',
										'shuttle',
										'air_con',
										'details'
									]}
									{@const extraEntries = Object.entries(meta).filter(
										([key]) => !knownMetaKeys.includes(key)
									)}
									{#if meta.shuttle || meta.air_con != null || meta.details || extraEntries.length > 0}
										<div class="mt-3 pt-3 border-t border-stone-700">
											<p class="text-stone-300 mb-2">Info</p>
											<dl class="text-sm space-y-2">
												{#if meta.shuttle}
													<div class="flex gap-4">
														<dt class="text-stone-400 w-24">Shuttle</dt>
														<dd>{meta.shuttle}</dd>
													</div>
												{/if}
												{#if meta.air_con != null}
													<div class="flex gap-4">
														<dt class="text-stone-400 w-24">Air Con</dt>
														<dd>{meta.air_con ? 'Yes' : 'No'}</dd>
													</div>
												{/if}
												{#if meta.details}
													<div class="flex gap-4">
														<dt class="text-stone-400 w-24">Details</dt>
														<dd>{meta.details}</dd>
													</div>
												{/if}
												{#each extraEntries as [key, value] (key)}
													<div class="flex gap-4">
														<dt class="text-stone-400 w-24">
															{capitalize(key.replace(/_/g, ' '))}
														</dt>
														<dd>{value}</dd>
													</div>
												{/each}
											</dl>
										</div>
									{/if}
									{#if play.source === 'telegram' && play.source_link}
										<div class="mt-3 pt-3 border-t border-stone-700">
											<a
												rel="external noopener noreferrer"
												href={play.source_link}
												target="_blank"
												class="text-sm text-blue-400 hover:text-blue-300 hover:underline"
											>
												View in Telegram ↗
											</a>
										</div>
									{/if}
									<div class="mt-3 pt-3 border-t border-stone-700">
										<dl class="text-xs text-stone-500 space-y-1">
											<div class="flex gap-4">
												<dt class="w-24">Created</dt>
												<dd>{formatDateTime(play.created_at)}</dd>
											</div>
											{#if play.updated_at !== play.created_at}
												<div class="flex gap-4">
													<dt class="w-24">Updated</dt>
													<dd>{formatDateTime(play.updated_at)}</dd>
												</div>
											{/if}
										</dl>
									</div>
								</Dialog.Content>
							</Dialog.Root>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{:else}
	<p>No plays found.</p>
{/if}

<div class="my-6 flex gap-4 w-full">
	{#if page.url.searchParams.has('cursor')}
		<Button variant="outline" onclick={() => history.back()}>Previous</Button>
	{/if}
	<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
	{#if data.plays.has_more && data.plays.next_cursor != null}
		<Button class="ms-auto" variant="outline" href={getNextPageUrl(data.plays.next_cursor)}
			>Next</Button
		>
	{/if}
</div>
