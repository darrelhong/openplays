<script lang="ts">
	import { Select as SelectPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/cn';
	import ChevronsDown from '@lucide/svelte/icons/chevrons-down';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: SelectPrimitive.ScrollDownButtonProps = $props();
</script>

<SelectPrimitive.ScrollDownButton
	bind:ref
	data-slot="select-scroll-down-button"
	class={cn('text-stone-400 py-1 flex w-full items-center justify-center', className)}
	{...restProps}
>
	<ChevronsDown size={14} />
</SelectPrimitive.ScrollDownButton>
