<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog/index';
	import {
		capitalize,
		formatDate,
		formatDateTime,
		formatPlayFee,
		formatLevel,
		formatTime
	} from '$lib/utils/formatting';
	import type { Play } from './types';

	let { play, dialog = true }: { play: Play; dialog?: boolean } = $props();

	const knownMetaKeys = ['fee_male', 'fee_female', 'shuttle', 'air_con', 'details'];
	const meta = $derived(play.meta ?? {});
	const extraEntries = $derived(Object.entries(meta).filter(([key]) => !knownMetaKeys.includes(key)));
	const hasVenueCoordinates = $derived(
		play.venue_latitude != null && play.venue_longitude != null
	);
	const mapsHref = $derived.by(() => {
		if (!hasVenueCoordinates) return '';
		return `https://www.google.com/maps?q=${play.venue_latitude},${play.venue_longitude}`;
	});
	const isUserCreated = $derived(play.created_by != null);
	const sourceLabel = $derived(isUserCreated ? 'User created' : 'Auto-created from Telegram');
</script>

{#snippet details()}
	<section class="py-2 md:py-3">
		<header>
			<div class="mb-2 flex flex-wrap gap-2 items-center">
				<span class="text-xs px-2 py-0.5 border border-border rounded-full bg-card/70">
					{capitalize(play.sport)}
				</span>
				<span
					class={`text-xs px-2 py-0.5 border rounded-full ${isUserCreated ? 'text-sky-700 border-sky-300/60 bg-sky-100/40 dark:text-sky-300 dark:border-sky-700/60 dark:bg-sky-900/20' : 'text-muted border-border bg-card/50'}`}
				>
					{sourceLabel}
				</span>
			</div>
			<h1 class="text-2xl font-semibold pe-6">{play.venue_name}</h1>
			{#if hasVenueCoordinates}
				<p class="mb-2">
					<a
						href={mapsHref}
						target="_blank"
						rel="external noopener noreferrer"
						class="text-xs text-muted hover:text-foreground hover:underline"
					>
						View in maps ↗
					</a>
				</p>
			{/if}
			<p class="text-base mb-4 mt-1 md:text-lg">
				{formatDate(play.starts_at, play.timezone, { year: 'numeric' })} · {formatTime(
					play.starts_at,
					play.timezone
				)} - {formatTime(play.ends_at, play.timezone)}
			</p>
		</header>
		<dl class="text-sm space-y-2">
			<div class="flex gap-4">
				<dt class="text-muted w-24">Host</dt>
				<dd>{play.host_name}</dd>
			</div>
			{#if play.game_type}
				<div class="flex gap-4">
					<dt class="text-muted w-24">Type</dt>
					<dd>{capitalize(play.game_type)}</dd>
				</div>
			{/if}
			<div class="flex gap-4">
				<dt class="text-muted w-24">Level</dt>
				<dd>{formatLevel(play.level_min, play.level_max)}</dd>
			</div>
			<div class="flex gap-4">
				<dt class="text-muted w-24">Fee</dt>
				<dd>{formatPlayFee(play)}</dd>
			</div>
			<div class="flex gap-4">
				<dt class="text-muted w-24">Slots</dt>
				<dd>{play.slots_left ?? '-'} / {play.max_players ?? '-'}</dd>
			</div>
			{#if play.courts != null}
				<div class="flex gap-4">
					<dt class="text-muted w-24">Courts</dt>
					<dd>{play.courts}</dd>
				</div>
			{/if}
		</dl>
		{#if play.contacts?.length}
			<div class="mt-3 pt-3 border-t border-border">
				<p class="text-muted mb-2">Contacts</p>
				{#each play.contacts as contact (`${contact.type}:${contact.value}`)}
					<div class="text-sm flex gap-4">
						<dt class="text-muted shrink-0 w-24">{contact.type}</dt>
						<dd>{contact.value}</dd>
					</div>
				{/each}
			</div>
		{/if}
		{#if meta.shuttle || meta.air_con != null || meta.details || extraEntries.length > 0}
			<div class="mt-3 pt-3 border-t border-border">
				<p class="text-muted mb-2">Info</p>
				<dl class="text-sm space-y-2">
					{#if meta.shuttle}
						<div class="flex gap-4">
							<dt class="text-muted w-24">Shuttle</dt>
							<dd>{meta.shuttle}</dd>
						</div>
					{/if}
					{#if meta.air_con != null}
						<div class="flex gap-4">
							<dt class="text-muted w-24">Air Con</dt>
							<dd>{meta.air_con ? 'Yes' : 'No'}</dd>
						</div>
					{/if}
					{#if meta.details}
						<div class="flex gap-4">
							<dt class="text-muted w-24">Details</dt>
							<dd>{meta.details}</dd>
						</div>
					{/if}
					{#each extraEntries as [key, value] (key)}
						<div class="flex gap-4">
							<dt class="text-muted w-24">{capitalize(key.replace(/_/g, ' '))}</dt>
							<dd>{value}</dd>
						</div>
					{/each}
				</dl>
			</div>
		{/if}
		{#if play.source === 'telegram' && (play.source_link || play.source_sender_link)}
			<div class="mt-3 pt-3 border-t border-border flex flex-col gap-1">
				{#if play.source_link}
					<a
						rel="external noopener noreferrer"
						href={play.source_link}
						target="_blank"
						class="text-sm text-blue-400 hover:text-blue-300 hover:underline"
					>
						View in Telegram ↗
					</a>
				{/if}
				{#if play.source_sender_link}
					<a
						rel="external noopener noreferrer"
						href={play.source_sender_link}
						target="_blank"
						class="text-sm text-blue-400 hover:text-blue-300 hover:underline"
					>
						Message sender ↗
					</a>
				{/if}
			</div>
		{/if}
		<div class="mt-3 pt-3 border-t border-border">
			<dl class="text-xs text-muted-foreground space-y-1">
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
	</section>
{/snippet}

{#if dialog}
	<Dialog.Content variant="right">
		{@render details()}
	</Dialog.Content>
{:else}
	{@render details()}
{/if}
