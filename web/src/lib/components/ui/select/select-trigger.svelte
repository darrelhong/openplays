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
		'text-sm text-stone-100 px-3 border border-stone-500 rounded-lg bg-stone-800 h-9 w-full',
		'flex gap-2 items-center',
		'placeholder:text-stone-500 focus:outline-none focus:ring-1 focus:ring-stone-400',
		'disabled:opacity-50 disabled:cursor-not-allowed',
		className
	)}
	{...restProps}
>
	{@render children?.()}
	<ChevronsUpDown size={16} class="text-stone-400 ml-auto shrink-0" />
</SelectPrimitive.Trigger>
