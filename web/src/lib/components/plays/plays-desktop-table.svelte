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
	import PlayRosterPreview from './play-roster-preview.svelte';
	import PlayViewerStateBadge from './play-viewer-state-badge.svelte';
	import type { Play } from './types';

	let { plays, showViewerState = false }: { plays: Play[]; showViewerState?: boolean } = $props();
</script>

<div class="hidden lg:block">
	<ScrollArea orientation="horizontal" viewportClasses="pb-2.5">
		<table class="w-full border-collapse">
			<thead>
				<tr
					class="text-muted border-b border-border *:font-medium *:p-2 *:text-start *:whitespace-nowrap"
				>
					{#if showViewerState}
						<th>My status</th>
					{/if}
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
				{#each plays as play (play.id)}
					<tr class="border-b border-border *:p-2 hover:bg-card *:whitespace-nowrap">
						{#if showViewerState}
							<td><PlayViewerStateBadge state={play.viewer_state} /></td>
						{/if}
						<td>{play.venue_name}</td>
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
