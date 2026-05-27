<script lang="ts">
	import { resolve } from '$app/paths';
	import { Badge } from '$lib/components/ui/badge/index';
	import {
		capitalize,
		formatDate,
		formatPlayFee,
		formatLevel,
		formatTime
	} from '$lib/utils/formatting';
	import PlayRosterPreview from './play-roster-preview.svelte';
	import type { Play } from './types';

	let { plays }: { plays: Play[] } = $props();
</script>

<div class="gap-2 grid md:grid-cols-2 lg:hidden">
	{#each plays as play (play.id)}
		<a
			href={resolve(`/play/${play.id}`)}
			class="p-2.5 border border-border rounded-xl bg-card shadow-sm transition-colors hover:bg-card/80"
		>
			<div class="flex gap-1.5 items-start justify-between">
				<div class="min-w-0">
					<p class="leading-tight font-semibold">{play.venue_name}</p>
					<p class="text-sm text-muted mt-0.5">
						{formatDate(play.starts_at, play.timezone)} · {formatTime(
							play.starts_at,
							play.timezone
						)} - {formatTime(play.ends_at, play.timezone)}
					</p>
				</div>
				<Badge variant={play.created_by != null ? 'info' : 'muted'} class="shrink-0">
					{capitalize(play.sport)}
				</Badge>
			</div>

			{#if play.created_by && play.max_players}
				<div class="mt-3">
					<PlayRosterPreview {play} />
				</div>
			{/if}

			<dl class="text-sm mt-2 gap-x-2 gap-y-1 grid grid-cols-3">
				<div>
					<dt class="text-xs text-muted tracking-wide">Host</dt>
					<dd class="mt-0.5 truncate">{play.host_name}</dd>
				</div>
				<div>
					<dt class="text-xs text-muted tracking-wide">Level</dt>
					<dd class="mt-0.5">{formatLevel(play.level_min, play.level_max)}</dd>
				</div>
				<div>
					<dt class="text-xs text-muted tracking-wide">Fee</dt>
					<dd class="mt-0.5">{formatPlayFee(play)}</dd>
				</div>
				{#if play.listing_type === 'sell_booking' || !play.created_by || !play.max_players}
					<div>
						<dt class="text-xs text-muted tracking-wide">Slots</dt>
						<dd class="mt-0.5">
							{#if play.listing_type === 'sell_booking'}
								To let go
							{:else}
								{play.slots_left ?? '-'} / {play.max_players ?? '-'}
							{/if}
						</dd>
					</div>
				{/if}
				{#if play.game_type}
					<div>
						<dt class="text-xs text-muted tracking-wide">Type</dt>
						<dd class="mt-0.5">{capitalize(play.game_type)}</dd>
					</div>
				{/if}
			</dl>
		</a>
	{/each}
</div>
