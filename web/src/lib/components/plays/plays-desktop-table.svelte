<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import * as Dialog from '$lib/components/ui/dialog/index';
	import ScrollArea from '$lib/components/ui/scroll-area/scroll-area.svelte';
	import {
		capitalize,
		formatDate,
		formatPlayFee,
		formatLevel,
		formatTime
	} from '$lib/utils/formatting';
	import PlayDetailsContent from './play-details-content.svelte';
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
								<PlayDetailsContent {play} />
							</Dialog.Root>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</ScrollArea>
</div>
