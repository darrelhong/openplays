<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
	import { cn } from '$lib/utils/cn';
	import type { Snippet } from 'svelte';

	let {
		ref = $bindable(null),
		class: className,
		children,
		sideOffset = 8,
		...restProps
	}: WithoutChildrenOrChild<DropdownMenuPrimitive.ContentProps> & {
		children: Snippet;
	} = $props();
</script>

<DropdownMenuPrimitive.Portal>
	<DropdownMenuPrimitive.Content
		bind:ref
		data-slot="dropdown-menu-content"
		class={cn(
			'p-1 outline-none border border-border rounded-lg bg-card min-w-44 shadow-lg z-50',
			'opacity-100 translate-y-0 scale-100 transition-all duration-150 ease-out',
			'data-[state=closed]:opacity-0 data-[state=closed]:translate-y-1 data-[state=closed]:scale-98 data-[state=closed]:duration-100 data-[state=closed]:ease-in',
			'data-[starting-style]:opacity-0 data-[starting-style]:translate-y-1 data-[starting-style]:scale-98',
			className
		)}
		{sideOffset}
		{...restProps}
	>
		{@render children?.()}
	</DropdownMenuPrimitive.Content>
</DropdownMenuPrimitive.Portal>
