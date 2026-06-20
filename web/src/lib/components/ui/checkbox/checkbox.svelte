<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';
	import { Checkbox as CheckboxPrimitive } from 'bits-ui';
	import CheckboxRoot from './checkbox-root.svelte';
	import { cn } from '$lib/utils/cn';

	type Props = Omit<CheckboxPrimitive.RootProps, 'children' | 'class'> & {
		class?: ClassValue;
		labelClass?: ClassValue;
		rootClass?: ClassValue;
		children?: Snippet;
	};

	let {
		ref = $bindable(null),
		checked = $bindable(false),
		indeterminate = $bindable(false),
		disabled,
		class: className,
		labelClass,
		rootClass,
		children,
		...restProps
	}: Props = $props();
</script>

<label
	data-slot="checkbox-label"
	class={cn(
		'text-sm text-foreground gap-3 grid grid-cols-[auto_1fr] cursor-pointer select-none items-center',
		disabled && 'opacity-50 pointer-events-none',
		className
	)}
>
	<CheckboxRoot
		bind:ref
		bind:checked
		bind:indeterminate
		class={rootClass}
		{disabled}
		{...restProps}
	/>
	{#if children}
		<span class={cn('leading-none', labelClass)}>{@render children()}</span>
	{/if}
</label>
