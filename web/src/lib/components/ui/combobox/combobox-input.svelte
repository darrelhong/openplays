<script lang="ts">
	import { Combobox as ComboboxPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/cn';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: ComboboxPrimitive.InputProps = $props();
</script>

<ComboboxPrimitive.Input
	bind:ref
	data-slot="combobox-input"
	class={cn(
		'text-sm text-stone-100 px-3 pr-8 border border-stone-500 rounded-lg bg-stone-800 h-9 w-full',
		'placeholder:text-stone-500 focus:outline-none focus:ring-1 focus:ring-stone-400',
		'disabled:opacity-50 disabled:cursor-not-allowed',
		className
	)}
	autocomplete="off"
	{...restProps}
/>
