<script lang="ts">
	import Check from '@lucide/svelte/icons/check';
	import Minus from '@lucide/svelte/icons/minus';
	import { Checkbox as CheckboxPrimitive } from 'bits-ui';
	import type { ClassValue } from 'svelte/elements';
	import { cn } from '$lib/utils/cn';

	type Props = CheckboxPrimitive.RootProps & {
		class?: ClassValue;
	};

	let {
		ref = $bindable(null),
		checked = $bindable(false),
		indeterminate = $bindable(false),
		class: className,
		children: childContent,
		...restProps
	}: Props = $props();
</script>

<CheckboxPrimitive.Root
	bind:ref
	bind:checked
	bind:indeterminate
	data-slot="checkbox"
	class={cn(
		'text-primary-foreground border border-input-border rounded-md bg-input shrink-0 size-4 shadow-sm transition-colors',
		'inline-flex cursor-pointer items-center justify-center',
		'focus-visible:outline-2 focus-visible:outline-ring focus-visible:outline-offset-1',
		'data-[state=checked]:border-primary data-[state=checked]:bg-primary',
		'data-[state=indeterminate]:border-primary data-[state=indeterminate]:bg-primary',
		'disabled:opacity-50 disabled:pointer-events-none',
		'data-[readonly]:opacity-80 data-[readonly]:cursor-default',
		className
	)}
	{...restProps}
>
	{#snippet children({ checked, indeterminate })}
		{#if indeterminate}
			<Minus class="size-3" aria-hidden="true" />
		{:else if checked}
			<Check class="size-3" aria-hidden="true" />
		{/if}
		{@render childContent?.({ checked, indeterminate })}
	{/snippet}
</CheckboxPrimitive.Root>
