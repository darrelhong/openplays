<script lang="ts">
	import { DatePicker as DatePickerPrimitive } from 'bits-ui';
	import Calendar from '@lucide/svelte/icons/calendar';
	import { cn } from '$lib/utils/cn';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: DatePickerPrimitive.TriggerProps = $props();
</script>

<DatePickerPrimitive.Trigger
	bind:ref
	data-slot="date-picker-trigger"
	class={cn('text-stone-400 ml-auto p-1 rounded-md inline-flex transition-colors items-center justify-center hover:bg-stone-700', className)}
	{...restProps}
>
	<Calendar size={16} />
</DatePickerPrimitive.Trigger>
