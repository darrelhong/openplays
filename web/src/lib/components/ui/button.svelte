<script lang="ts">
	import type { ClassValue } from 'svelte/elements';
	import { Button, type ButtonRootProps } from 'bits-ui';
	import { cn } from '$lib/utils/cn';

	type Variant = 'default' | 'secondary' | 'outline' | 'ghost';
	type Size = 'xs' | 'sm' | 'md' | 'lg';

	type Props = ButtonRootProps & {
		variant?: Variant;
		size?: Size;
		class?: ClassValue;
	};

	let { variant = 'default', size = 'md', children, class: className, ...rest }: Props = $props();

	const variantClasses: Record<Variant, string> = {
		default: 'text-stone-950 bg-stone-100 hover:bg-stone-100/85',
		secondary: 'text-stone-100 bg-stone-700 hover:bg-stone-700/80',
		outline: 'text-stone-100 border border-stone-500 bg-stone-800 shadow-xl hover:bg-stone-700/70',
		ghost: 'text-stone-100 hover:bg-stone-800'
	};

	const sizeClasses: Record<Size, string> = {
		xs: 'text-xs px-2 h-6',
		sm: 'text-sm px-2.5 h-8',
		md: 'text-base px-3 py-1.5',
		lg: 'text-base px-2.5 h-10'
	};
</script>

<Button.Root
	class={cn(
		'font-medium rounded-lg inline-flex cursor-pointer transition-colors items-center justify-center',
		'focus-visible:outline-2 focus-visible:outline-stone-400 focus-visible:outline-offset-1',
		'disabled:opacity-50 disabled:pointer-events-none',
		variantClasses[variant],
		sizeClasses[size],
		className
	)}
	{...rest}
>
	{@render children?.()}
</Button.Root>
