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
		'text-stone-100 px-3 py-2 border border-stone-600 rounded-lg bg-stone-900 w-full placeholder:text-stone-500 focus:outline-none focus:border-stone-400',
		disabled && 'opacity-60 cursor-not-allowed',
		className
	)}
	{...rest}
/>
