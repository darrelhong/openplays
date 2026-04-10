<script lang="ts">
	import { Combobox as ComboboxPrimitive } from 'bits-ui';
	import ChevronsUpDown from '@lucide/svelte/icons/chevrons-up-down';
	import { cn } from '$lib/utils/cn';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: ComboboxPrimitive.TriggerProps = $props();
</script>

<ComboboxPrimitive.Trigger
	bind:ref
	data-slot="combobox-trigger"
	class={cn('text-stone-400 end-2 top-1/2 absolute -translate-y-1/2', className)}
	{...restProps}
>
	<ChevronsUpDown size={16} />
</ComboboxPrimitive.Trigger>
