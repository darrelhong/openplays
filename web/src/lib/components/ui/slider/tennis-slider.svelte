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
	}

	let {
		value = $bindable([3, 4]),
		label,
		required = false
	}: Props = $props();

	const selectedRange = $derived(`${value[0]?.toFixed(1) ?? '1.0'} - ${value[1]?.toFixed(1) ?? '7.0'}`);

	function isHalfStep(tick: TickItem) {
		return Math.round(tick.value * 10) % 5 === 0;
	}

	function isMajorStep(tick: TickItem) {
		return Math.round(tick.value * 10) % 10 === 0;
	}
</script>

<Slider bind:value min={1} max={7} step={0.1} {label} {required} minLabel="" maxLabel="">
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
<div class="text-xs text-muted-foreground mt-2 grid grid-cols-3 items-center">
	<span>1.0</span>
	<span class="text-sm text-primary font-semibold text-center">{selectedRange}</span>
	<span class="text-right">7.0</span>
</div>
