<script lang="ts">
	import { Combobox as ComboboxPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/cn';
	import ChevronsUp from '@lucide/svelte/icons/chevrons-up';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: ComboboxPrimitive.ScrollUpButtonProps = $props();
</script>

<ComboboxPrimitive.ScrollUpButton
	bind:ref
	data-slot="combobox-scroll-up-button"
	class={cn('text-stone-400 py-1 flex w-full items-center justify-center', className)}
	{...restProps}
>
	<ChevronsUp size={14} />
</ComboboxPrimitive.ScrollUpButton>
