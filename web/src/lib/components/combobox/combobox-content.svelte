<script lang="ts">
	import { Combobox as ComboboxPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
	import { cn } from '$lib/utils/cn';
	import type { Snippet } from 'svelte';
	import ComboboxScrollUpButton from './combobox-scroll-up-button.svelte';
	import ComboboxScrollDownButton from './combobox-scroll-down-button.svelte';

	let {
		ref = $bindable(null),
		class: className,
		children,
		sideOffset = 8,
		...restProps
	}: WithoutChildrenOrChild<ComboboxPrimitive.ContentProps> & {
		children: Snippet;
	} = $props();
</script>

<ComboboxPrimitive.Portal>
	<ComboboxPrimitive.Content
		bind:ref
		data-slot="combobox-content"
		class={cn(
			'max-h-[min(24rem,var(--bits-combobox-content-available-height))] min-w-[var(--bits-combobox-anchor-width)] w-[var(--bits-combobox-anchor-width)] z-50',
			'py-1 border border-stone-500 rounded-lg bg-stone-900 select-none shadow-xl',
			'opacity-100 scale-100 transition-all duration-200 ease-out',
			'data-[state=closed]:opacity-0 data-[state=closed]:scale-95 data-[state=closed]:duration-150 data-[state=closed]:ease-in',
			'data-[starting-style]:opacity-0 data-[starting-style]:scale-95',
			className
		)}
		{sideOffset}
		{...restProps}
	>
		<ComboboxScrollUpButton />
		<ComboboxPrimitive.Viewport class="px-1">
			{@render children?.()}
		</ComboboxPrimitive.Viewport>
		<ComboboxScrollDownButton />
	</ComboboxPrimitive.Content>
</ComboboxPrimitive.Portal>
