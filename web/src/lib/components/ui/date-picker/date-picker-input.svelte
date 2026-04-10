<script lang="ts">
	import { DatePicker as DatePickerPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/cn';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: DatePickerPrimitive.InputProps = $props();
</script>

<DatePickerPrimitive.Input
	bind:ref
	data-slot="date-picker-input"
	class={cn(
		'text-sm text-stone-100 pe-1.5 ps-3 border border-stone-500 rounded-lg bg-stone-800 h-9 w-full',
		'flex select-none items-center',
		'focus-within:outline-none focus-within:ring-1 focus-within:ring-stone-400',
		className
	)}
	{...restProps}
/>
