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

	let { play }: { play: Play } = $props();

	const knownMetaKeys = ['fee_male', 'fee_female', 'shuttle', 'air_con', 'details'];
</script>

<Dialog.Content variant="right">
	<Dialog.Header>
		<Dialog.Title class="text-xl pe-6">{play.venue_name}</Dialog.Title>
		<p class="text-lg mb-4">
			{formatDate(play.starts_at, play.timezone, { year: 'numeric' })} · {formatTime(
				play.starts_at,
				play.timezone
			)} - {formatTime(play.ends_at, play.timezone)}
		</p>
	</Dialog.Header>
	<dl class="text-sm space-y-2">
		<div class="flex gap-4">
			<dt class="text-muted w-24">Host</dt>
			<dd>{play.host_name}</dd>
		</div>
		<div class="flex gap-4">
			<dt class="text-muted w-24">Sport</dt>
			<dd>
				{capitalize(play.sport)}{play.game_type ? ` · ${capitalize(play.game_type)}` : ''}
			</dd>
		</div>
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
	{@const meta = play.meta ?? {}}
	{@const extraEntries = Object.entries(meta).filter(([key]) => !knownMetaKeys.includes(key))}
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
</Dialog.Content>
