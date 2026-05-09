<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
	import { cn } from '$lib/utils/cn';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: WithoutChildrenOrChild<DropdownMenuPrimitive.SeparatorProps> = $props();
</script>

<DropdownMenuPrimitive.Separator
	bind:ref
	data-slot="dropdown-menu-separator"
	class={cn('my-1 bg-border h-px', className)}
	{...restProps}
/>
