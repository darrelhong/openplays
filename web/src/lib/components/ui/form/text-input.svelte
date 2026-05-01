<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';
	import { cn } from '$lib/utils/cn';

	type Props = Omit<HTMLInputAttributes, 'value'> & {
		value?: string;
		class?: string;
	};

	let { value = $bindable(), disabled = false, class: className, ...rest }: Props = $props();
</script>

<input
	bind:value
	{disabled}
	class={cn(
		'text-foreground px-3 py-2 border border-input-border rounded-lg bg-input w-full placeholder:text-muted-foreground focus:outline-none focus:border-ring',
		disabled && 'opacity-60 cursor-not-allowed',
		className
	)}
	{...rest}
/>
