<script lang="ts">
	import { Select as SelectPrimitive } from 'bits-ui';
	import ChevronsUpDown from '@lucide/svelte/icons/chevrons-up-down';
	import { cn } from '$lib/utils/cn';
	import type { Snippet } from 'svelte';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: SelectPrimitive.TriggerProps & { children: Snippet } = $props();
</script>

<SelectPrimitive.Trigger
	bind:ref
	data-slot="select-trigger"
	class={cn(
		'text-sm text-foreground px-3 border border-border rounded-lg bg-card h-9 w-full',
		'flex gap-2 items-center',
		'placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring',
		'disabled:opacity-50 disabled:cursor-not-allowed',
		className
	)}
	{...restProps}
>
	{@render children?.()}
	<ChevronsUpDown size={16} class="text-muted ml-auto shrink-0" />
</SelectPrimitive.Trigger>
