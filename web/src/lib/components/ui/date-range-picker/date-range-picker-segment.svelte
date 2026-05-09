<script lang="ts">
	import { DateRangePicker as DateRangePickerPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/cn';

	let {
		ref = $bindable(null),
		class: className,
		part,
		...restProps
	}: DateRangePickerPrimitive.SegmentProps = $props();
</script>

<DateRangePickerPrimitive.Segment
	bind:ref
	data-slot="date-range-picker-segment"
	class={cn(
		part === 'literal'
			? 'text-muted-foreground px-0.5'
			: 'px-1 py-0.5 rounded aria-[valuetext=Empty]:text-muted-foreground focus:text-foreground focus-visible:outline-none focus:bg-accent hover:bg-accent focus-visible:ring-0',
		className
	)}
	{part}
	{...restProps}
/>
