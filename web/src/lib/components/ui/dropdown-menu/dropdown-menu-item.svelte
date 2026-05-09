<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
	import { cn } from '$lib/utils/cn';
	import type { Snippet } from 'svelte';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithoutChildrenOrChild<DropdownMenuPrimitive.ItemProps> & {
		children: Snippet;
	} = $props();
</script>

<DropdownMenuPrimitive.Item
	bind:ref
	data-slot="dropdown-menu-item"
	class={cn(
		'text-sm px-3 py-2 outline-none rounded-md cursor-pointer transition-colors',
		'hover:text-accent-foreground hover:bg-accent',
		'data-[highlighted]:text-accent-foreground data-[highlighted]:bg-accent',
		'data-[disabled]:opacity-50 data-[disabled]:pointer-events-none',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</DropdownMenuPrimitive.Item>
