<script lang="ts">
	import Star from '@lucide/svelte/icons/star';

	let {
		name = 'rating',
		value = $bindable(null),
		label = 'Rating',
		class: className = ''
	}: {
		name?: string;
		value?: number | null;
		label?: string;
		class?: string;
	} = $props();

	const stars = [1, 2, 3, 4, 5];

	let hovered = $state<number | null>(null);

	function starClass(star: number) {
		// Committed rating fills the stars; hovering previews with an amber
		// outline only
		if (value != null && star <= value) {
			return 'fill-amber-400 text-amber-400';
		}
		if (hovered != null && star <= hovered) {
			return 'text-amber-400';
		}
		return 'text-muted';
	}

	function toggle(star: number) {
		// Ratings are optional: tapping the current value clears it
		value = value === star ? null : star;
	}
</script>

<fieldset>
	<legend class="sr-only">{label}</legend>
	<div
		class={`flex gap-1.5 items-center ${className}`}
		onmouseleave={() => (hovered = null)}
		role="presentation"
	>
		{#each stars as star (star)}
			<label class="cursor-pointer" onmouseenter={() => (hovered = star)}>
				<input
					type="radio"
					{name}
					value={star}
					checked={value === star}
					onclick={() => toggle(star)}
					class="sr-only"
				/>
				<span class="sr-only">{star} {star === 1 ? 'star' : 'stars'}</span>
				<Star aria-hidden="true" class={`h-6 w-6 transition-colors ${starClass(star)}`} />
			</label>
		{/each}
	</div>
	<!-- Fixed-height slot so the stars never shift when Clear appears -->
	<div class={`mt-1 h-4 flex items-center ${className}`}>
		{#if value != null}
			<button
				type="button"
				class="text-xs text-muted hover:text-foreground hover:underline"
				onclick={() => (value = null)}
			>
				Clear
			</button>
		{/if}
	</div>
</fieldset>
