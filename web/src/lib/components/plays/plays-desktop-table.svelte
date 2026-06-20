<script lang="ts">
	import { Badge } from '$lib/components/ui/badge/index';
	import Button from '$lib/components/ui/button.svelte';
	import ScrollArea from '$lib/components/ui/scroll-area/scroll-area.svelte';
	import {
		capitalize,
		formatDate,
		formatPlayFee,
		formatLevel,
		formatTime
	} from '$lib/utils/formatting';
	import PlayFavouriteButton from './play-favourite-button.svelte';
	import PlayRosterPreview from './play-roster-preview.svelte';
	import PlayViewerStateBadge from './play-viewer-state-badge.svelte';
	import type { Play } from './types';

	let { plays }: { plays: Play[] } = $props();

	const showFavourite = $derived(plays.some((play) => play.is_favourited != null));
	const hasViewerStateBadge = $derived(
		plays.some((play) => play.viewer_state && play.viewer_state !== 'not_joined')
	);
	const hasUnlisted = $derived(plays.some((play) => play.visibility === 'unlisted'));
</script>

<div class="hidden lg:block">
	<ScrollArea orientation="horizontal" viewportClasses="pb-2.5">
		<table class="w-full border-collapse">
			<thead>
				<tr
					class="text-muted border-b border-border *:font-medium *:p-2 *:text-start *:whitespace-nowrap"
				>
					{#if showFavourite}
						<th><span class="sr-only">Favourite</span></th>
					{/if}
					{#if hasViewerStateBadge}
						<th>My status</th>
					{/if}
					<th>Game</th>
					<th>Date</th>
					<th>Time</th>
					<th>Host</th>
					<th>Sport</th>
					{#if hasUnlisted}
						<th>Visibility</th>
					{/if}
					<th>Level</th>
					<th>Fee</th>
					<th>Slots/Type</th>
					<th>Details</th>
				</tr>
			</thead>
			<tbody>
				{#each plays as play (play.id)}
					<tr class="border-b border-border *:p-2 hover:bg-card *:whitespace-nowrap">
						{#if showFavourite}
							<td><PlayFavouriteButton {play} /></td>
						{/if}
						{#if hasViewerStateBadge}
							<td><PlayViewerStateBadge state={play.viewer_state} /></td>
						{/if}
						<td>
							<div class="max-w-56">
								<p class="font-medium truncate">{play.name || play.venue_name}</p>
								{#if play.name}
									<p class="text-xs text-muted truncate">{play.venue_name}</p>
								{/if}
							</div>
						</td>
						<td>{formatDate(play.starts_at, play.timezone)}</td>
						<td
							>{formatTime(play.starts_at, play.timezone)} - {formatTime(
								play.ends_at,
								play.timezone
							)}</td
						>
						<td>{play.host_name}</td>
						<td>
							<Badge variant={play.created_by != null ? 'info' : 'muted'}>
								{capitalize(play.sport)}
							</Badge>
						</td>
						{#if hasUnlisted}
							<td>
								{#if play.visibility === 'unlisted'}
									<Badge variant="outline">Unlisted</Badge>
								{/if}
							</td>
						{/if}
						<td>{formatLevel(play.level_min, play.level_max)}</td>
						<td>{formatPlayFee(play)}</td>
						<td>
							{#if play.listing_type === 'sell_booking'}
								To let go
							{:else if play.created_by && play.max_players}
								<PlayRosterPreview {play} maxVisibleSlots={6} />
							{:else}
								{play.slots_left ?? '-'} / {play.max_players ?? '-'}
							{/if}
						</td>
						<td>
							<Button href={`/play/${play.id}`} size="xs" variant="outline">View</Button>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</ScrollArea>
</div>
