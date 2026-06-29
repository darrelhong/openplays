<script lang="ts">
	import { resolve } from '$app/paths';
	import Check from '@lucide/svelte/icons/check';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import * as Dialog from '$lib/components/ui/dialog/index';
	import ActionConfirmDialog from '$lib/components/ui/dialog/action-confirm-dialog.svelte';
	import BaseAvatar from '$lib/components/ui/avatar/base-avatar.svelte';
	import { Badge } from '$lib/components/ui/badge/index';
	import Button from '$lib/components/ui/button.svelte';
	import {
		capitalize,
		formatDate,
		formatDateTime,
		formatPlayFee,
		formatLevel,
		formatTime
	} from '$lib/utils/formatting';
	import { canDirectJoin, getPlayJoinLabel } from '$lib/utils/play-join-label';
	import PlayFavouriteButton from './play-favourite-button.svelte';
	import PlayParticipantIdentity from './play-participant-identity.svelte';
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
	}: {
		play: Play;
		user?: User | null;
		form?: ActionForm;
		dialog?: boolean;
	} = $props();

	const knownMetaKeys = ['fee_male', 'fee_female', 'shuttle', 'air_con', 'details'];
	const meta = $derived(play.meta ?? {});
	const confirmedParticipants = $derived(
		play.confirmed_participants ?? play.participant_preview ?? []
	);
	const addedParticipants = $derived(play.added_participants ?? []);
	const waitlist = $derived(play.waitlist ?? []);
	const historyEvents = $derived(play.history_events ?? []);
	const confirmedCount = $derived(play.confirmed_count ?? confirmedParticipants.length);
	const addedCount = $derived(play.added_count ?? addedParticipants.length);
	const rosteredCount = $derived(confirmedCount + addedCount);
	const isCancelled = $derived(play.cancelled_at != null);
	const openSlots = $derived(isCancelled ? 0 : Math.max(play.slots_left ?? 0, 0));
	const openSlotRows = $derived(
		Array.from({ length: Math.min(openSlots, 12) }, (_, index) => rosteredCount + index + 1)
	);
	const slotsLeftLabel = $derived(isCancelled ? '0' : (play.slots_left ?? '-'));
	const hiddenOpenSlotCount = $derived(Math.max(openSlots - openSlotRows.length, 0));
	const playerCountLabel = $derived(
		play.max_players == null ? String(rosteredCount) : `${rosteredCount}/${play.max_players}`
	);
	const extraEntries = $derived(
		Object.entries(meta).filter(([key]) => !knownMetaKeys.includes(key))
	);
	const hasVenueCoordinates = $derived(play.venue_latitude != null && play.venue_longitude != null);
	const hasVenueMapLink = $derived(Boolean(play.venue_google_place_id) || hasVenueCoordinates);
	const playTitle = $derived(play.name || play.venue_name);
	const mapsHref = $derived.by(() => {
		if (play.venue_google_place_id) {
			const query = encodeURIComponent(play.venue_name);
			const placeID = encodeURIComponent(play.venue_google_place_id);
			return `https://www.google.com/maps/search/?api=1&query=${query}&query_place_id=${placeID}`;
		}
		if (!hasVenueCoordinates) return '';
		return `https://www.google.com/maps?q=${play.venue_latitude},${play.venue_longitude}`;
	});
	const isUserCreated = $derived(play.created_by != null);
	const sourceLabel = $derived(isUserCreated ? 'User created' : 'Auto-created from Telegram');
	const viewerState = $derived(play.viewer_state ?? 'not_joined');
	const canManage = $derived(play.can_manage ?? false);
	const canManageActive = $derived(canManage && !isCancelled);
	const waitlistCount = $derived(play.waitlist_count ?? waitlist.length);
	const rosterSummary = $derived(
		`${confirmedCount} confirmed${addedCount > 0 ? ` • ${addedCount} added` : ''} • ${waitlistCount} waitlisted`
	);
	const joinsWaitlist = $derived(!canDirectJoin(play, user));
	const joinLabel = $derived(getPlayJoinLabel(play, user));
	function participantName(participant: Participant) {
		return participant.display_name ?? (participant.is_guest ? 'Guest player' : 'Player');
	}
</script>

{#snippet confirmedBadge()}
	<Badge variant="success" class="font-semibold">Confirmed</Badge>
{/snippet}

{#snippet waitlistBadge()}
	<Badge variant="warning">On waitlist</Badge>
{/snippet}

{#snippet addedBadge()}
	<Badge variant="info">Added</Badge>
{/snippet}

{#snippet addParticipantDialog(participant: Participant)}
	<ActionConfirmDialog
		title={`Add ${participantName(participant)}?`}
		description="They will be added to the game and can confirm or decline their spot."
		action="?/acceptParticipant"
		confirmLabel="Add player"
	>
		{#snippet trigger({ props })}
			<Button
				type="button"
				size="xs"
				variant="secondary"
				disabled={openSlots <= 0}
				aria-label={`Add ${participantName(participant)}`}
				title={openSlots > 0 ? 'Add player' : 'No open slots'}
				class="gap-1.5"
				{...props}
			>
				<Check class="h-3.5 w-3.5" aria-hidden="true" />
				Add
			</Button>
		{/snippet}
		{#snippet fields()}
			<input type="hidden" name="participant_id" value={participant.id} />
		{/snippet}
	</ActionConfirmDialog>
{/snippet}

{#snippet removeParticipantDialog(participant: Participant)}
	<ActionConfirmDialog
		title={`Remove ${participantName(participant)}?`}
		action="?/removeParticipant"
		confirmLabel="Remove player"
		confirmVariant="destructive"
	>
		{#snippet trigger({ props })}
			<Button
				type="button"
				size="xs"
				variant="outline"
				aria-label={`Remove ${participantName(participant)}`}
				title="Remove player"
				class="gap-1.5"
				{...props}
			>
				<Trash2 class="h-3.5 w-3.5" aria-hidden="true" />
				Remove
			</Button>
		{/snippet}
		{#snippet fields()}
			<input type="hidden" name="participant_id" value={participant.id} />
		{/snippet}
	</ActionConfirmDialog>
{/snippet}

{#snippet confirmedPlayer(participant: Participant)}
	<li class="py-2">
		<div class="flex gap-3 items-center justify-between">
			<PlayParticipantIdentity {participant} />
			<div class="shrink-0">
				{@render confirmedBadge()}
			</div>
		</div>

		{#if canManageActive && !participant.is_host}
			<div class="ms-12 mt-2 flex flex-wrap gap-2 justify-end">
				{@render removeParticipantDialog(participant)}
			</div>
		{/if}
	</li>
{/snippet}

{#snippet addedPlayer(participant: Participant)}
	<li class="py-2">
		<div class="flex gap-3 items-center justify-between">
			<PlayParticipantIdentity
				{participant}
				secondary={participant.is_guest ? 'Guest' : 'Awaiting player confirmation'}
			/>
			<div class="shrink-0">
				{@render addedBadge()}
			</div>
		</div>

		{#if canManageActive}
			<div class="ms-12 mt-2 flex flex-wrap gap-2 justify-end">
				{@render removeParticipantDialog(participant)}
			</div>
		{:else if viewerState === 'added'}
			<div class="ms-12 mt-2 flex flex-wrap gap-2 justify-end">
				<ActionConfirmDialog
					title="Confirm spot?"
					description="You will join the roster for this game."
					action="?/confirmParticipant"
					confirmLabel="Confirm spot"
				>
					{#snippet trigger({ props })}
						<Button type="button" size="xs" {...props}>Confirm spot</Button>
					{/snippet}
				</ActionConfirmDialog>
				<ActionConfirmDialog
					title="Decline spot?"
					description="You will be removed from this game."
					action="?/leave"
					confirmLabel="Decline"
					confirmVariant="destructive"
				>
					{#snippet trigger({ props })}
						<Button type="button" size="xs" variant="outline" {...props}>Decline</Button>
					{/snippet}
				</ActionConfirmDialog>
			</div>
		{/if}
	</li>
{/snippet}

{#snippet waitlistPlayer(participant: Participant)}
	<li class="py-2 flex gap-3 items-center justify-between">
		<PlayParticipantIdentity {participant} />
		{#if canManageActive}
			<div class="flex shrink-0 flex-wrap gap-2 items-center justify-end">
				{@render addParticipantDialog(participant)}
				{@render removeParticipantDialog(participant)}
			</div>
		{:else}
			<div class="shrink-0">
				{@render waitlistBadge()}
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

{#snippet activitySection()}
	<section class="pt-4 border-t border-border">
		<div class="md:max-w-lg">
			<div class="mb-2">
				<h2 class="text-sm text-foreground font-semibold">Activity</h2>
			</div>
			<ul class="space-y-2">
				{#each historyEvents as event (event.id)}
					<li class="ps-3 border-s border-border gap-1 grid">
						<p class="text-sm text-foreground">{event.message}</p>
						<time
							datetime={event.created_at}
							title={formatDateTime(event.created_at)}
							class="text-xs text-muted"
						>
							{event.relative_time}
						</time>
					</li>
				{/each}
			</ul>
		</div>
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
				{#if play.visibility === 'unlisted'}
					<Badge variant="outline">Unlisted</Badge>
				{/if}
				{#if isCancelled}
					<Badge variant="warning">Cancelled</Badge>
				{/if}
			</div>
			<div class="flex gap-3 items-start justify-between">
				<h1 class="text-2xl font-semibold pe-6">{playTitle}</h1>
				<PlayFavouriteButton {play} />
			</div>
			{#if play.description}
				<p class="text-sm text-muted mb-2 mt-1 whitespace-pre-line">{play.description}</p>
			{/if}
			{#if !play.name && hasVenueMapLink}
				<p class="mb-2">
					<a
						href={mapsHref}
						target="_blank"
						rel="external noopener noreferrer"
						class="text-xs text-muted hover:text-foreground hover:underline"
					>
						View in map ↗
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
			{#if play.name}
				<div class="flex gap-4">
					<dt class="text-muted w-24">Location</dt>
					<dd>
						<span>{play.venue_name}</span>
						{#if hasVenueMapLink}
							<a
								href={mapsHref}
								target="_blank"
								rel="external noopener noreferrer"
								class="text-xs text-muted ms-2 hover:text-foreground hover:underline"
							>
								View in map ↗
							</a>
						{/if}
					</dd>
				</div>
			{/if}
			<div class="flex gap-4">
				<dt class="text-muted w-24">Host</dt>
				<dd>
					{#if play.creator_username}
						<a href={resolve(`/${play.creator_username}`)} class="hover:underline"
							>{play.host_name}</a
						>
					{:else}
						{play.host_name}
					{/if}
				</dd>
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
				<dd>{slotsLeftLabel} / {play.max_players ?? '-'}</dd>
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
								{playerCountLabel} • {rosterSummary}
							</p>
						</div>
						{#if canManageActive}
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
						{#each addedParticipants as participant (participant.id)}
							{@render addedPlayer(participant)}
						{/each}
						{#each openSlotRows as slotNumber (slotNumber)}
							{@render openPlayerSlot()}
						{/each}
						{#if hiddenOpenSlotCount > 0}
							<li class="text-sm text-muted py-2">+{hiddenOpenSlotCount} more open slots</li>
						{/if}
					</ul>

					{#if canManageActive || waitlist.length > 0}
						<div class="mt-4 space-y-4">
							{@render rosterSection('Waitlist', waitlist)}
						</div>
					{/if}

					<div class="mt-4 flex flex-wrap gap-2 items-center justify-start">
						{#if isCancelled}
							<Badge variant="warning" size="sm">Cancelled</Badge>
						{:else if !user}
							<Button href="/login" size="sm">Sign in to join</Button>
						{:else if viewerState === 'creator'}
							<Badge variant="info" size="sm">Hosting</Badge>
						{:else if viewerState === 'confirmed'}
							{@render confirmedBadge()}
							<ActionConfirmDialog
								title="Leave game?"
								action="?/leave"
								confirmLabel="Leave game"
								confirmVariant="destructive"
							>
								{#snippet trigger({ props })}
									<Button type="button" size="sm" variant="outline" {...props}>Leave game</Button>
								{/snippet}
							</ActionConfirmDialog>
						{:else if viewerState === 'waitlisted'}
							<ActionConfirmDialog
								title="Leave waitlist?"
								action="?/leave"
								confirmLabel="Leave waitlist"
								confirmVariant="destructive"
							>
								{#snippet trigger({ props })}
									<Button type="button" size="sm" variant="outline" {...props}
										>Leave waitlist</Button
									>
								{/snippet}
							</ActionConfirmDialog>
						{:else if viewerState === 'added'}
							{@render addedBadge()}
						{:else if joinsWaitlist}
							<ActionConfirmDialog
								title="Join waitlist?"
								description="Please double-check your availability before joining. Hosts can then add you into the game."
								action="?/join"
								confirmLabel="Join waitlist"
							>
								{#snippet trigger({ props })}
									<Button type="button" size="sm" {...props}>{joinLabel}</Button>
								{/snippet}
							</ActionConfirmDialog>
						{:else}
							<ActionConfirmDialog
								title="Join game?"
								description="Please double-check your availability before joining."
								action="?/join"
								confirmLabel="Join game"
							>
								{#snippet trigger({ props })}
									<Button type="button" size="sm" {...props}>{joinLabel}</Button>
								{/snippet}
							</ActionConfirmDialog>
						{/if}
					</div>
				</section>
				{#if historyEvents.length > 0}
					{@render activitySection()}
				{/if}
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
