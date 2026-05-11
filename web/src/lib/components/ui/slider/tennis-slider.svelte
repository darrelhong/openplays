<script lang="ts">
	import { Slider as SliderPrimitive } from 'bits-ui';
	import Slider from './slider.svelte';

	type TickItem = {
		value: number;
		index: number;
	};

	interface Props {
		value?: number[];
		label?: string;
		required?: boolean;
		minLabel?: string;
		maxLabel?: string;
	}

	let {
		value = $bindable([3, 4]),
		label,
		required = false,
		minLabel = 'Min',
		maxLabel = 'Max'
	}: Props = $props();

	function isHalfStep(tick: TickItem) {
		return Math.round(tick.value * 10) % 5 === 0;
	}

	function isMajorStep(tick: TickItem) {
		return Math.round(tick.value * 10) % 10 === 0;
	}
</script>

<Slider bind:value min={1} max={7} step={0.1} {label} {required} {minLabel} {maxLabel}>
	{#snippet ticks({ tickItems })}
		{#each tickItems as tick (tick.index)}
			{#if isHalfStep(tick)}
				<SliderPrimitive.Tick
					index={tick.index}
					class={isMajorStep(tick)
						? 'bg-muted-foreground/60 h-full w-px top-0 absolute'
						: 'bg-muted-foreground/40 h-1/2 w-px top-1/4 absolute'}
				/>
			{/if}
		{/each}
	{/snippet}
</Slider>
