<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import ScrollArea from '$lib/components/ui/scroll-area/scroll-area.svelte';
	import {
		capitalize,
		formatDate,
		formatPlayFee,
		formatLevel,
		formatTime
	} from '$lib/utils/formatting';
	import type { Play } from './types';

	let { plays }: { plays: Play[] } = $props();
</script>

<div class="hidden lg:block">
	<ScrollArea orientation="horizontal" viewportClasses="pb-2.5">
		<table class="w-full border-collapse">
			<thead>
				<tr
					class="text-muted border-b border-border *:font-medium *:p-2 *:text-start *:whitespace-nowrap"
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
				{#each plays as play (play.id)}
					<tr class="border-b border-border *:p-2 hover:bg-card *:whitespace-nowrap">
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
							<span
								class={`text-xs px-2 py-0.5 border rounded-full inline-flex items-center ${play.created_by != null ? 'text-sky-700 border-sky-300/60 bg-sky-100/40 dark:text-sky-300 dark:border-sky-700/60 dark:bg-sky-900/20' : 'text-muted border-border bg-card/70'}`}
							>
								{capitalize(play.sport)}
							</span>
						</td>
						<td>{formatLevel(play.level_min, play.level_max)}</td>
						<td>{formatPlayFee(play)}</td>
						<td
							>{#if play.listing_type === 'sell_booking'}To let go{:else}{play.slots_left ?? '-'} / {play.max_players ??
									'-'}{/if}</td
						>
						<td>
							<Button href={`/play/${play.id}`} size="xs" variant="outline">View</Button>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</ScrollArea>
</div>
