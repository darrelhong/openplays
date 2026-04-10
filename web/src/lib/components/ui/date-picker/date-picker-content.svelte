<script lang="ts">
	import { DatePicker as DatePickerPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/cn';
	import type { Snippet } from 'svelte';

	let {
		ref = $bindable(null),
		class: className,
		children,
		sideOffset = 8,
		...restProps
	}: DatePickerPrimitive.ContentProps & { children: Snippet } = $props();
</script>

<DatePickerPrimitive.Content
	bind:ref
	data-slot="date-picker-content"
	class={cn(
		'p-4 border border-stone-500 rounded-lg bg-stone-900 shadow-xl z-50',
		'opacity-100 scale-100 transition-all duration-200 ease-out',
		'data-[state=closed]:opacity-0 data-[state=closed]:scale-95 data-[state=closed]:duration-150 data-[state=closed]:ease-in',
		'data-[starting-style]:opacity-0 data-[starting-style]:scale-95',
		className
	)}
	{sideOffset}
	{...restProps}
>
	{@render children?.()}
</DatePickerPrimitive.Content>
