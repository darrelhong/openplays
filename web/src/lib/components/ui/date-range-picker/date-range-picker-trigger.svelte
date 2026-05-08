<script lang="ts">
	import { DateRangePicker as DateRangePickerPrimitive } from 'bits-ui';
	import Calendar from '@lucide/svelte/icons/calendar';
	import { cn } from '$lib/utils/cn';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: DateRangePickerPrimitive.TriggerProps = $props();
</script>

<DateRangePickerPrimitive.Trigger
	bind:ref
	data-slot="date-range-picker-trigger"
	class={cn(
		'text-muted ml-auto p-1 rounded-md inline-flex transition-colors items-center justify-center hover:bg-accent',
		className
	)}
	{...restProps}
>
	<Calendar size={16} />
</DateRangePickerPrimitive.Trigger>
