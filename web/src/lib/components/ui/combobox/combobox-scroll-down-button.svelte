<script lang="ts">
	import { Combobox as ComboboxPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/cn';
	import ChevronsDown from '@lucide/svelte/icons/chevrons-down';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: ComboboxPrimitive.ScrollDownButtonProps = $props();
</script>

<ComboboxPrimitive.ScrollDownButton
	bind:ref
	data-slot="combobox-scroll-down-button"
	class={cn('text-stone-400 py-1 flex w-full items-center justify-center', className)}
	{...restProps}
>
	<ChevronsDown size={14} />
</ComboboxPrimitive.ScrollDownButton>
