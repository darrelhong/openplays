<script lang="ts">
	import UserAvatar from '$lib/components/ui/avatar/user-avatar.svelte';
	import { getPlayRosterPreview } from '$lib/utils/play-roster-preview';
	import type { Play } from './types';

	let { play, maxVisibleSlots = 8 }: { play: Play; maxVisibleSlots?: number } = $props();

	const preview = $derived(getPlayRosterPreview(play, maxVisibleSlots));
</script>

{#if preview}
	<div class="flex gap-2 items-center" aria-label={preview.label}>
		<div class="flex items-center -space-x-1.5">
			{#each preview.slots as slot, index (index)}
				{#if slot.kind === 'known'}
					<span class="inline-flex shrink-0 relative">
						<UserAvatar
							src={slot.photoUrl}
							nameForFallback={slot.name}
							title={slot.ratingCode ? `${slot.name} (${slot.ratingCode})` : slot.name}
							className="text-[0.65rem] h-7 w-7 ring-2 ring-background"
						/>
						{#if slot.ratingCode}
							<span
								class="text-[0.55rem] text-primary-foreground leading-none font-bold px-1 border border-background rounded-full bg-primary inline-flex h-4 min-w-4 shadow-sm items-center justify-center absolute z-10 -bottom-1 -right-1.5"
								aria-hidden="true"
							>
								{slot.ratingCode}
							</span>
						{/if}
					</span>
				{:else if slot.kind === 'occupied'}
					<span
						title={slot.label}
						class="border border-border rounded-full bg-card inline-flex h-7 w-7 ring-2 ring-background items-center justify-center"
					>
						<span class="rounded-full bg-muted-foreground/70 h-2 w-2"></span>
						<span class="sr-only">{slot.label}</span>
					</span>
				{:else}
					<span
						title={slot.label}
						class="border border-muted-foreground/50 rounded-full border-dashed bg-background inline-flex h-7 w-7 ring-2 ring-background items-center justify-center"
					>
						<span class="sr-only">{slot.label}</span>
					</span>
				{/if}
			{/each}

			{#if preview.hiddenSlots > 0}
				<span
					class="text-[0.65rem] text-muted-foreground font-medium border border-border rounded-full bg-card inline-flex h-7 w-7 ring-2 ring-background items-center justify-center"
				>
					+{preview.hiddenSlots}
				</span>
			{/if}
		</div>
	</div>
{/if}
