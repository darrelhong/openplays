<script lang="ts">
	import { Dialog as DialogPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
	import { cn } from '$lib/utils/cn';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: WithoutChildrenOrChild<DialogPrimitive.OverlayProps> = $props();
</script>

<DialogPrimitive.Overlay
	bind:ref
	data-slot="dialog-overlay"
	class={cn(
		'bg-black/60 inset-0 fixed z-50',
		'opacity-100 transition-opacity duration-200 ease-out',
		'data-[state=closed]:opacity-0 data-[state=closed]:duration-150 data-[state=closed]:ease-in',
		'data-[starting-style]:opacity-0',
		className
	)}
	{...restProps}
/>
