<script lang="ts">
	import { Select as SelectPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
	import { cn } from '$lib/utils/cn';
	import type { Snippet } from 'svelte';
	import SelectScrollUpButton from './select-scroll-up-button.svelte';
	import SelectScrollDownButton from './select-scroll-down-button.svelte';

	let {
		ref = $bindable(null),
		class: className,
		children,
		sideOffset = 8,
		...restProps
	}: WithoutChildrenOrChild<SelectPrimitive.ContentProps> & {
		children: Snippet;
	} = $props();
</script>

<SelectPrimitive.Portal>
	<SelectPrimitive.Content
		bind:ref
		data-slot="select-content"
		class={cn(
			'max-h-[min(24rem,var(--bits-select-content-available-height))] min-w-[var(--bits-select-anchor-width)] w-[var(--bits-select-anchor-width)] z-50',
			'py-1 border border-stone-500 rounded-lg bg-stone-900 select-none shadow-xl',
			'opacity-100 scale-100 transition-all duration-200 ease-out',
			'data-[state=closed]:opacity-0 data-[state=closed]:scale-95 data-[state=closed]:duration-150 data-[state=closed]:ease-in',
			'data-[starting-style]:opacity-0 data-[starting-style]:scale-95',
			className
		)}
		{sideOffset}
		{...restProps}
	>
		<SelectScrollUpButton />
		<SelectPrimitive.Viewport class="px-1">
			{@render children?.()}
		</SelectPrimitive.Viewport>
		<SelectScrollDownButton />
	</SelectPrimitive.Content>
</SelectPrimitive.Portal>
