<script lang="ts">
	import { MAX_PROPS_PER_REVIEW, reviewPropLabel } from '$lib/consts/review-props';

	let {
		name = 'props',
		options,
		selected = $bindable([]),
		max = MAX_PROPS_PER_REVIEW,
		class: className = ''
	}: {
		name?: string;
		options: string[];
		selected?: string[];
		max?: number;
		class?: string;
	} = $props();

	const atLimit = $derived(selected.length >= max);

	function toggle(slug: string) {
		selected = selected.includes(slug) ? selected.filter((s) => s !== slug) : [...selected, slug];
	}
</script>

<div class={`flex flex-wrap gap-1.5 ${className}`}>
	{#each options as slug (slug)}
		{@const checked = selected.includes(slug)}
		{@const disabled = !checked && atLimit}
		<label
			class={`text-xs px-2.5 py-1 border rounded-full inline-flex transition-colors items-center select-none ${
				checked
					? 'border-primary bg-primary/10 text-foreground font-medium'
					: disabled
						? 'border-border text-muted/50 cursor-not-allowed'
						: 'border-border text-muted cursor-pointer hover:border-primary/50 hover:text-foreground'
			}`}
		>
			<input
				type="checkbox"
				{name}
				value={slug}
				{checked}
				{disabled}
				onclick={() => toggle(slug)}
				class="sr-only"
			/>
			{reviewPropLabel(slug)}
		</label>
	{/each}
</div>
