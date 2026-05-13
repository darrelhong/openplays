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
					<UserAvatar
						src={slot.photoUrl}
						nameForFallback={slot.name}
						title={slot.name}
						className="text-[0.65rem] h-7 w-7 ring-2 ring-background"
					/>
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
