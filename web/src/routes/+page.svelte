<script lang="ts">
	import { page } from '$app/state';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
	import type { PageData } from './$types';
	import Button from '$lib/components/button.svelte';
	import * as Dialog from '$lib/components/dialog/index';
	import { capitalize } from '$lib/utils/formatting';

	type Play = NonNullable<PageData['items']>[number];

	let { data }: { data: PageData } = $props();

	function formatDate(iso: string, tz: string): string {
		const d = new Date(iso);
		return d.toLocaleDateString('en-SG', {
			weekday: 'short',
			month: 'short',
			day: 'numeric',
			timeZone: tz
		});
	}

	function formatTime(iso: string, tz: string): string {
		const d = new Date(iso);
		const minute = d.toLocaleString('en-SG', { minute: 'numeric', timeZone: tz });
		return d.toLocaleTimeString('en-SG', {
			hour: 'numeric',
			...(minute !== '0' && { minute: '2-digit' }),
			timeZone: tz
		});
	}

	function formatFee(cents: number, currency: string): string {
		const dollars = cents / 100;
		const minimumFractionDigits = cents % 100 === 0 ? 0 : cents >= 1000 && cents % 10 === 0 ? 1 : 2;

		return new Intl.NumberFormat('en-SG', {
			style: 'currency',
			currency,
			minimumFractionDigits,
			maximumFractionDigits: 2
		}).format(dollars);
	}

	function getNumericFee(value: unknown): number | null {
		return typeof value === 'number' && Number.isFinite(value) ? value : null;
	}

	function getMetaFee(meta: unknown, key: 'fee_male' | 'fee_female'): number | null {
		if (meta == null || typeof meta !== 'object') return null;

		return getNumericFee((meta as Record<string, unknown>)[key]);
	}

	function formatPlayFee(play: Play): string {
		const fee = getNumericFee(play.fee);
		if (fee != null) return formatFee(fee, play.currency);

		const feeMale = getMetaFee(play.meta, 'fee_male');
		const feeFemale = getMetaFee(play.meta, 'fee_female');
		const fees = [
			feeMale != null ? `${formatFee(feeMale, play.currency)} (M)` : null,
			feeFemale != null ? `${formatFee(feeFemale, play.currency)} (F)` : null
		].filter((value): value is string => value != null);

		return fees.length > 0 ? fees.join(', ') : '-';
	}

	function formatLevel(min?: string, max?: string): string {
		if (min && max) return `${min} - ${max}`;
		if (min) return `${min}+`;
		if (max) return `- ${max}`;
		return '-';
	}

	function getNextPageUrl(nextCursor: string): string {
		const params = new SvelteURLSearchParams(page.url.searchParams);
		params.set('cursor', nextCursor);
		return `?${params.toString()}`;
	}
</script>

<h1 class="text-xl font-semibold mb-2">Plays</h1>

{#if data.items && data.items.length > 0}
	<p class="text-stone-300 mb-1">Showing {data.total} plays</p>

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
				{#each data.items as play (play.id)}
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
								<Dialog.Trigger><Button size="xs" variant="outline">View</Button></Dialog.Trigger>
								<Dialog.Content variant="right">
									<Dialog.Header>
										<Dialog.Title class="text-xl pe-6">{play.venue_name || play.venue}</Dialog.Title
										>
										<p class="text-lg mb-4">
											{formatDate(play.starts_at, play.timezone)} · {formatTime(
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
											<p class="text-xs text-stone-400 mb-2">Contacts</p>
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
											<p class="text-xs text-stone-400 mb-2">Info</p>
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
	{#if data.has_more && data.next_cursor != null}
		<Button class="ms-auto" variant="outline" href={getNextPageUrl(data.next_cursor)}>Next</Button>
	{/if}
</div>
