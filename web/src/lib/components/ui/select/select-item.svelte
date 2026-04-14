<script lang="ts">
	import { Select as SelectPrimitive } from 'bits-ui';
	import Check from '@lucide/svelte/icons/check';
	import { cn } from '$lib/utils/cn';

	let {
		ref = $bindable(null),
		class: className,
		value,
		label,
		disabled,
		...restProps
	}: SelectPrimitive.ItemProps = $props();
</script>

<SelectPrimitive.Item
	bind:ref
	data-slot="select-item"
	class={cn(
		'text-sm text-stone-100 px-2 py-1.5 outline-none rounded-md flex w-full cursor-default select-none items-center',
		'data-[highlighted]:bg-stone-800',
		'data-[disabled]:opacity-50 data-[disabled]:pointer-events-none',
		className
	)}
	{value}
	{label}
	{disabled}
	{...restProps}
>
	{#snippet children({ selected })}
		<span class="flex-1 truncate">{label}</span>
		{#if selected}
			<Check size={16} class="text-stone-400 ml-2 shrink-0" />
		{/if}
	{/snippet}
</SelectPrimitive.Item>
