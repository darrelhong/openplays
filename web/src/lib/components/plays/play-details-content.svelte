<script lang="ts">
	import Check from '@lucide/svelte/icons/check';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import * as Dialog from '$lib/components/ui/dialog/index';
	import BaseAvatar from '$lib/components/ui/avatar/base-avatar.svelte';
	import { Badge, RatingBadge } from '$lib/components/ui/badge/index';
	import Button from '$lib/components/ui/button.svelte';
	import UserAvatar from '$lib/components/ui/avatar/user-avatar.svelte';
	import {
		capitalize,
		formatDate,
		formatDateTime,
		formatPlayFee,
		formatLevel,
		formatTime
	} from '$lib/utils/formatting';
	import { getPlayJoinLabel } from '$lib/utils/play-join-label';
	import type { components } from '$lib/api/types.gen';
	import type { Play } from './types';

	type User = components['schemas']['User'];
	type Participant = components['schemas']['PlayParticipantPreviewPublic'];
	type ActionForm = { error?: string } | null | undefined;

	let {
		play,
		user = null,
		form = null,
		dialog = true
	}: { play: Play; user?: User | null; form?: ActionForm; dialog?: boolean } = $props();

	const knownMetaKeys = ['fee_male', 'fee_female', 'shuttle', 'air_con', 'details'];
	const meta = $derived(play.meta ?? {});
	const confirmedParticipants = $derived(
		play.confirmed_participants ?? play.participant_preview ?? []
	);
	const waitlist = $derived(play.waitlist ?? []);
	const confirmedCount = $derived(play.confirmed_count ?? confirmedParticipants.length);
	const openSlots = $derived(Math.max(play.slots_left ?? 0, 0));
	const openSlotRows = $derived(
		Array.from({ length: Math.min(openSlots, 12) }, (_, index) => confirmedCount + index + 1)
	);
	const hiddenOpenSlotCount = $derived(Math.max(openSlots - openSlotRows.length, 0));
	const playerCountLabel = $derived(
		play.max_players == null ? String(confirmedCount) : `${confirmedCount}/${play.max_players}`
	);
	const extraEntries = $derived(
		Object.entries(meta).filter(([key]) => !knownMetaKeys.includes(key))
	);
	const hasVenueCoordinates = $derived(play.venue_latitude != null && play.venue_longitude != null);
	const mapsHref = $derived.by(() => {
		if (!hasVenueCoordinates) return '';
		return `https://www.google.com/maps?q=${play.venue_latitude},${play.venue_longitude}`;
	});
	const isUserCreated = $derived(play.created_by != null);
	const sourceLabel = $derived(isUserCreated ? 'User created' : 'Auto-created from Telegram');
	const viewerState = $derived(play.viewer_state ?? 'not_joined');
	const canManage = $derived(play.can_manage ?? false);
	const waitlistCount = $derived(play.waitlist_count ?? waitlist.length);
	const joinLabel = $derived(getPlayJoinLabel(play, user));

	function participantName(participant: Participant) {
		return participant.display_name ?? (participant.is_guest ? 'Guest player' : 'Player');
	}
</script>

{#snippet confirmedBadge()}
	<Badge variant="success" class="font-semibold h-6">Confirmed</Badge>
{/snippet}

{#snippet waitlistBadge()}
	<Badge variant="warning" size="sm">On waitlist</Badge>
{/snippet}

{#snippet playerAvatar(participant: Participant)}
	<div class="shrink-0 relative">
		<UserAvatar
			src={participant.photo_url}
			nameForFallback={participantName(participant)}
			className="h-9 w-9 text-xs"
		/>
		{#if participant.rating_code}
			<RatingBadge value={participant.rating_code} class="absolute -bottom-1 -right-1" />
		{/if}
	</div>
{/snippet}

{#snippet confirmedPlayer(participant: Participant)}
	<li class="py-2 flex gap-3 items-center justify-between">
		<div class="flex gap-3 min-w-0 items-center">
			{@render playerAvatar(participant)}
			<div class="min-w-0">
				<p class="text-sm text-foreground font-medium break-words">
					{participantName(participant)}
				</p>
				{#if participant.is_host}
					<p class="text-xs text-muted">Host</p>
				{:else if participant.is_guest}
					<p class="text-xs text-muted">Guest</p>
				{/if}
			</div>
		</div>
		<div class="flex shrink-0 flex-wrap gap-2 items-center justify-end">
			{@render confirmedBadge()}
			{#if canManage && !participant.is_host}
				<form method="POST" action="?/removeParticipant">
					<input type="hidden" name="participant_id" value={participant.id} />
					<Button
						type="submit"
						size="xs"
						variant="outline"
						aria-label={`Remove ${participantName(participant)}`}
						title="Remove player"
						class="gap-1.5"
					>
						<Trash2 class="h-3.5 w-3.5" aria-hidden="true" />
						Remove
					</Button>
				</form>
			{/if}
		</div>
	</li>
{/snippet}

{#snippet waitlistPlayer(participant: Participant)}
	<li class="py-2 flex gap-3 items-center justify-between">
		<div class="flex gap-3 min-w-0 items-center">
			{@render playerAvatar(participant)}
			<div class="min-w-0">
				<p class="text-sm text-foreground font-medium break-words">
					{participantName(participant)}
				</p>
				{#if participant.is_guest}
					<p class="text-xs text-muted">Guest</p>
				{/if}
			</div>
		</div>
		{#if canManage}
			<div class="flex shrink-0 flex-wrap gap-2 items-center justify-end">
				<form method="POST" action="?/acceptParticipant">
					<input type="hidden" name="participant_id" value={participant.id} />
					<Button
						type="submit"
						size="xs"
						variant="secondary"
						disabled={openSlots <= 0}
						aria-label={`Accept ${participantName(participant)}`}
						title={openSlots > 0 ? 'Accept player' : 'No open slots'}
						class="gap-1.5"
					>
						<Check class="h-3.5 w-3.5" aria-hidden="true" />
						Accept
					</Button>
				</form>
				<form method="POST" action="?/removeParticipant">
					<input type="hidden" name="participant_id" value={participant.id} />
					<Button
						type="submit"
						size="xs"
						variant="outline"
						aria-label={`Remove ${participantName(participant)}`}
						title="Remove player"
						class="gap-1.5"
					>
						<Trash2 class="h-3.5 w-3.5" aria-hidden="true" />
						Remove
					</Button>
				</form>
			</div>
		{/if}
	</li>
{/snippet}

{#snippet openPlayerSlot()}
	<li class="py-2 flex gap-3 items-center justify-between">
		<div class="flex gap-3 min-w-0 items-center">
			<BaseAvatar variant="dotted" className="h-9 w-9" />
		</div>
		<Badge variant="outline" class="h-6">Open</Badge>
	</li>
{/snippet}

{#snippet rosterSection(title: string, participants: Participant[])}
	<section>
		<div class="mb-2 flex gap-3 items-center justify-between">
			<h2 class="text-sm text-foreground font-semibold">{title}</h2>
			<span class="text-xs text-muted">{participants.length}</span>
		</div>
		{#if participants.length > 0}
			<ul class="px-3 border border-border rounded-md divide-border divide-y">
				{#each participants as participant (participant.id)}
					{@render waitlistPlayer(participant)}
				{/each}
			</ul>
		{:else}
			<p class="text-sm text-muted px-3 py-3 border border-border rounded-md border-dashed">
				None yet
			</p>
		{/if}
	</section>
{/snippet}

{#snippet details()}
	<section class="py-2 md:py-3">
		<header>
			<div class="mb-2 flex flex-wrap gap-2 items-center">
				<Badge>
					{capitalize(play.sport)}
				</Badge>
				<Badge variant={isUserCreated ? 'info' : 'muted'}>
					{sourceLabel}
				</Badge>
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
		{#if isUserCreated}
			<div class="mt-4 pt-4 border-t border-border space-y-4">
				<section class="p-3 border border-border rounded-md bg-card/50 md:max-w-lg">
					<div class="mb-3 flex flex-wrap gap-3 items-center justify-between">
						<div>
							<h2 class="text-sm text-foreground font-semibold">Players</h2>
							<p class="text-sm text-muted">
								{playerCountLabel} • {confirmedCount} confirmed • {waitlistCount} waitlisted
							</p>
						</div>
						{#if canManage}
							<Button href={`/play/${play.id}/edit`} size="sm" variant="outline">Edit</Button>
						{/if}
					</div>

					{#if form?.error}
						<p
							class="text-sm text-red-700 mb-3 px-3 py-2 border border-red-200 rounded-md bg-red-50"
						>
							{form.error}
						</p>
					{/if}

					<ul class="px-3 border border-border rounded-md divide-border divide-y">
						{#each confirmedParticipants as participant (participant.id)}
							{@render confirmedPlayer(participant)}
						{/each}
						{#each openSlotRows as slotNumber (slotNumber)}
							{@render openPlayerSlot()}
						{/each}
						{#if hiddenOpenSlotCount > 0}
							<li class="text-sm text-muted py-2">+{hiddenOpenSlotCount} more open slots</li>
						{/if}
					</ul>

					{#if canManage}
						<div class="mt-4">
							{@render rosterSection('Waitlist', waitlist)}
						</div>
					{/if}

					<div class="mt-4 flex flex-wrap gap-2 items-center justify-start">
						{#if !user}
							<Button href="/login" size="sm">Sign in to join</Button>
						{:else if viewerState === 'creator'}
							<Badge variant="info" size="sm">Hosting</Badge>
						{:else if viewerState === 'confirmed'}
							{@render confirmedBadge()}
							<form method="POST" action="?/leave">
								<Button type="submit" size="sm" variant="outline">Leave game</Button>
							</form>
						{:else if viewerState === 'waitlisted'}
							{@render waitlistBadge()}
							<form method="POST" action="?/leave">
								<Button type="submit" size="sm" variant="outline">Leave waitlist</Button>
							</form>
						{:else}
							<form method="POST" action="?/join">
								<Button type="submit" size="sm">{joinLabel}</Button>
							</form>
						{/if}
					</div>
				</section>
			</div>
		{/if}
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
		{#if play.created_at}
			<div class="mt-3 pt-3 border-t border-border">
				<dl class="text-xs text-muted-foreground space-y-1">
					<div class="flex gap-4">
						<dt class="w-24">Created</dt>
						<dd>{formatDateTime(play.created_at)}</dd>
					</div>
					{#if play.updated_at && play.updated_at !== play.created_at}
						<div class="flex gap-4">
							<dt class="w-24">Updated</dt>
							<dd>{formatDateTime(play.updated_at)}</dd>
						</div>
					{/if}
				</dl>
			</div>
		{/if}
	</section>
{/snippet}

{#if dialog}
	<Dialog.Content variant="right">
		{@render details()}
	</Dialog.Content>
{:else}
	{@render details()}
{/if}
