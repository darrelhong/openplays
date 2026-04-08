<script lang="ts">
	import { page } from '$app/state';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
	import type { PageData } from './$types';
	import Button from '$lib/components/button.svelte';

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
				<tr class="text-stone-400 border-b border-neutral-500 *:font-medium *:p-2 *:text-start">
					<th>Venue</th>
					<th>Date</th>
					<th>Time</th>
					<th>Host</th>
					<th>Sport</th>
					<th>Level</th>
					<th>Fee</th>
					<th>Slots</th>
					<th>Details</th>
				</tr>
			</thead>
			<tbody>
				{#each data.items as play, i (play.id)}
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
						<td>{play.sport}</td>
						<td>{formatLevel(play.level_min, play.level_max)}</td>
						<td>{formatPlayFee(play)}</td>
						<td>{play.slots_left ?? '-'} / {play.max_players ?? '-'}</td>
						<td class="relative">
							<details>
								<summary class="text-xs text-stone-400 cursor-pointer">Show</summary>
								<dl
									class="text-xs p-2 border border-stone-700 rounded bg-stone-800 max-w-[90vw] w-max shadow-lg right-0 absolute z-10 {data.items &&
									data.items.length - i <= 3
										? 'mb-1 bottom-full'
										: 'mt-1'}"
								>
									{#each Object.entries(play.meta ?? {}).filter(([key]) => key !== 'fee_male' && key !== 'fee_female') as [key, value] (key)}
										<div class="py-0.5 flex gap-2 whitespace-normal">
											<dt class="text-stone-500">{key.replace(/_/g, ' ')}</dt>
											<dd class="text-stone-200">{value}</dd>
										</div>
									{/each}
									{#if play.courts != null}
										<div class="py-0.5 flex gap-2 whitespace-normal">
											<dt class="text-stone-500">courts</dt>
											<dd class="text-stone-200">{play.courts}</dd>
										</div>
									{/if}
									<div class="py-0.5 flex gap-2 whitespace-normal">
										<dt class="text-stone-500">created</dt>
										<dd class="text-stone-200">
											{formatDate(play.created_at, play.timezone)}
											{formatTime(play.created_at, play.timezone)}
										</dd>
									</div>
									<div class="py-0.5 flex gap-2 whitespace-normal">
										<dt class="text-stone-500">updated</dt>
										<dd class="text-stone-200">
											{formatDate(play.updated_at, play.timezone)}
											{formatTime(play.updated_at, play.timezone)}
										</dd>
									</div>
									{#if play.contacts?.length}
										{#each play.contacts as contact (`${contact.type}:${contact.value}`)}
											<div class="py-0.5 flex gap-2 whitespace-normal">
												<dt class="text-stone-500">{contact.type}</dt>
												<dd class="text-stone-200">{contact.value}</dd>
											</div>
										{/each}
									{/if}
								</dl>
							</details>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{:else}
	<p>No plays found.</p>
{/if}

{#if data.has_more && data.next_cursor != null}
	<div class="my-6">
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
		<Button variant="outline" href={getNextPageUrl(data.next_cursor)}>Next</Button>
	</div>
{/if}
