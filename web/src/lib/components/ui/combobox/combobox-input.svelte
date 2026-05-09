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
		'text-sm text-foreground px-3 pr-8 border border-border rounded-lg bg-card h-9 w-full',
		'placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring',
		'disabled:opacity-50 disabled:cursor-not-allowed',
		className
	)}
	autocomplete="off"
	{...restProps}
/>
