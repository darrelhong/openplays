<script lang="ts">
	import { Select as SelectPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/cn';
	import ChevronsUp from '@lucide/svelte/icons/chevrons-up';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: SelectPrimitive.ScrollUpButtonProps = $props();
</script>

<SelectPrimitive.ScrollUpButton
	bind:ref
	data-slot="select-scroll-up-button"
	class={cn('text-stone-400 py-1 flex w-full items-center justify-center', className)}
	{...restProps}
>
	<ChevronsUp size={14} />
</SelectPrimitive.ScrollUpButton>
