<script lang="ts">
	import { cn } from '$lib/utils/cn';
	import { Slider as SliderPrimitive, Label } from 'bits-ui';
	import type { Snippet } from 'svelte';

	type TickItem = {
		value: number;
		index: number;
	};

	type ThumbItem = {
		value: number;
		index: number;
	};

	type SliderSnippetProps = {
		ticks: number[];
		thumbs: number[];
		tickItems: TickItem[];
		thumbItems: ThumbItem[];
	};

	type MultipleSliderRootProps = Extract<SliderPrimitive.RootProps, { type: 'multiple' }>;

	type Props = Omit<MultipleSliderRootProps, 'type' | 'value' | 'children' | 'child'> & {
		value?: number[];
		label?: string;
		required?: boolean;
		minLabel?: string;
		maxLabel?: string;
		ticks?: Snippet<[SliderSnippetProps]>;
	};

	let {
		value = $bindable([3, 4]),
		min = 1,
		max = 7,
		step = 0.1,
		label,
		required = false,
		minLabel = 'Min',
		maxLabel = 'Max',
		autoSort = false,
		disabled = false,
		class: className,
		ticks: ticksSnippet,
		...restProps
	}: Props = $props();

	const stableId = Math.random().toString(36).slice(2, 8);
	const sliderId = $derived(label ? `slider-${stableId}` : undefined);

	const minValue = $derived(value[0]?.toFixed(1) ?? '');
	const maxValue = $derived(value[1]?.toFixed(1) ?? '');

	function getValue() {
		return value;
	}

	function setValue(next: number[]) {
		const minVal = Math.max(min, Math.min(max, next[0] ?? min));
		const maxVal = Math.max(min, Math.min(max, next[1] ?? max));
		value = [Math.min(minVal, maxVal), Math.max(minVal, maxVal)];
	}
</script>

<div>
	{#if label}
		<Label.Root for={sliderId} class="text-sm text-muted mb-2 block"
			>{label}{#if required}<span class="text-destructive/70 ml-0.5">*</span>{/if}</Label.Root
		>
	{/if}
	<div class="px-2">
		<SliderPrimitive.Root
			{min}
			{max}
			{step}
			type="multiple"
			{autoSort}
			{disabled}
			bind:value={getValue, setValue}
			class={cn('flex w-full select-none items-center relative touch-none', className)}
			{...restProps}
		>
			{#snippet children(slider)}
				<span
					class="rounded-full bg-foreground/20 grow h-2 w-full cursor-pointer relative overflow-hidden"
				>
					<SliderPrimitive.Range class="bg-foreground h-full absolute" />
					{@render ticksSnippet?.(slider)}
				</span>
				{#each slider.thumbItems as { index } (index)}
					<SliderPrimitive.Thumb
						{index}
						class="data-active:bg-background data-active:ring-foreground data-active:shadow-none rounded-full bg-background size-4 block cursor-pointer ring-2 ring-foreground/60 shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
					/>
				{/each}
			{/snippet}
		</SliderPrimitive.Root>
	</div>
	{#if minLabel || maxLabel}
		<div class="text-sm text-muted-foreground mt-2 flex justify-between">
			{#if minLabel}
				<span>{minLabel} {minValue}</span>
			{/if}
			{#if maxLabel}
				<span>{maxLabel} {maxValue}</span>
			{/if}
		</div>
	{/if}
</div>
