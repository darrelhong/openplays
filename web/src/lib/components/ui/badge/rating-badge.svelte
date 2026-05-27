<script lang="ts">
	import { cn } from '$lib/utils/cn';
	import type { ClassValue, SvelteHTMLElements } from 'svelte/elements';

	type Size = 'xs' | 'sm';

	type Props = Omit<SvelteHTMLElements['span'], 'class'> & {
		value: string;
		size?: Size;
		class?: ClassValue;
	};

	let { value, size = 'sm', class: className, ...rest }: Props = $props();

	const sizeClasses: Record<Size, string> = {
		xs: 'text-[0.55rem] font-bold px-1 h-4 min-w-4',
		sm: 'text-[10px] font-semibold px-1 h-4 min-w-6'
	};
</script>

<span
	class={cn(
		'text-stone-900 leading-none border border-background rounded-full bg-white inline-flex shadow-sm items-center justify-center',
		sizeClasses[size],
		className
	)}
	{...rest}
>
	{value}
</span>
