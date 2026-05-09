<script lang="ts">
	import * as Dialog from '$lib/components/ui/dialog/index';
	import { capitalize, formatDate, formatPlayFee, formatLevel, formatTime } from '$lib/utils/formatting';
	import PlayDetailsContent from './play-details-content.svelte';
	import type { Play } from './types';

	let { plays }: { plays: Play[] } = $props();
</script>

<div class="gap-2 grid md:grid-cols-2 lg:hidden">
	{#each plays as play (play.id)}
		<Dialog.Root>
			<Dialog.Trigger>
				{#snippet child({ props })}
					<div
						{...props}
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
							<span class="text-xs text-muted px-2 py-0.5 border border-border rounded-full shrink-0">
								{capitalize(play.sport)}
							</span>
						</div>

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
							{#if play.game_type}
								<div>
									<dt class="text-xs text-muted tracking-wide">Type</dt>
									<dd class="mt-0.5">{capitalize(play.game_type)}</dd>
								</div>
							{/if}
						</dl>
					</div>
				{/snippet}
			</Dialog.Trigger>
			<PlayDetailsContent {play} />
		</Dialog.Root>
	{/each}
</div>
