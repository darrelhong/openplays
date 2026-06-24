<script lang="ts">
	import { Popover as PopoverPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
	import type { Snippet } from 'svelte';
	import { cn } from '$lib/utils/cn';

	let {
		ref = $bindable(null),
		class: className,
		children,
		sideOffset = 8,
		align = 'center',
		collisionPadding = 16,
		...restProps
	}: WithoutChildrenOrChild<PopoverPrimitive.ContentProps> & {
		children: Snippet;
	} = $props();
</script>

<PopoverPrimitive.Portal>
	<PopoverPrimitive.Content
		bind:ref
		data-slot="popover-content"
		class={cn(
			'text-sm text-foreground p-0 outline-none border border-border rounded-lg bg-card max-w-[calc(100vw-2rem)] w-80 shadow-lg z-50',
			'opacity-100 translate-y-0 scale-100 transition-all duration-150 ease-out',
			'data-[state=closed]:opacity-0 data-[state=closed]:translate-y-1 data-[state=closed]:scale-98 data-[state=closed]:duration-100 data-[state=closed]:ease-in',
			'data-[starting-style]:opacity-0 data-[starting-style]:translate-y-1 data-[starting-style]:scale-98',
			className
		)}
		{sideOffset}
		{align}
		{collisionPadding}
		{...restProps}
	>
		{@render children?.()}
	</PopoverPrimitive.Content>
</PopoverPrimitive.Portal>
