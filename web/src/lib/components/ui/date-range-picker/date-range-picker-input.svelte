<script lang="ts">
	import { DateRangePicker as DateRangePickerPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/cn';

	let {
		ref = $bindable(null),
		class: className,
		type,
		...restProps
	}: DateRangePickerPrimitive.InputProps = $props();
</script>

<DateRangePickerPrimitive.Input
	bind:ref
	data-slot="date-range-picker-input"
	class={cn('flex min-w-0 items-center', className)}
	{type}
	{...restProps}
/>
