<script lang="ts">
	import { DatePicker as DatePickerPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/cn';

	let {
		ref = $bindable(null),
		class: className,
		part,
		...restProps
	}: DatePickerPrimitive.SegmentProps = $props();
</script>

<DatePickerPrimitive.Segment
	bind:ref
	data-slot="date-picker-segment"
	class={cn(
		part === 'literal'
			? 'text-stone-500 px-0.5'
			: 'px-1 py-0.5 rounded aria-[valuetext=Empty]:text-stone-500 focus:text-stone-100 focus-visible:outline-none focus:bg-stone-700 hover:bg-stone-700 focus-visible:ring-0',
		className
	)}
	{part}
	{...restProps}
/>
