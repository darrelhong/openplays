<script lang="ts">
	import { cn } from '$lib/utils/cn';
	import type { ClassValue, SvelteHTMLElements } from 'svelte/elements';

	type Variant = 'surface' | 'muted' | 'outline' | 'info' | 'success' | 'warning';
	type Size = 'xs' | 'sm';

	type Props = Omit<SvelteHTMLElements['span'], 'class'> & {
		variant?: Variant;
		size?: Size;
		class?: ClassValue;
	};

	let { variant = 'surface', size = 'xs', children, class: className, ...rest }: Props = $props();

	const variantClasses: Record<Variant, string> = {
		surface: 'text-foreground border border-border bg-card/70',
		muted: 'text-muted border border-border bg-card/70',
		outline: 'text-muted border border-border bg-background',
		info: 'text-sky-700 border border-sky-300/60 bg-sky-100/40 dark:text-sky-300 dark:border-sky-700/60 dark:bg-sky-900/20',
		success:
			'text-emerald-700 border border-emerald-300/70 bg-emerald-100/50 dark:text-emerald-200 dark:border-emerald-700/70 dark:bg-emerald-900/20',
		warning:
			'text-amber-800 border border-amber-300/70 bg-amber-100/50 dark:text-amber-200 dark:border-amber-700/70 dark:bg-amber-900/20'
	};

	const sizeClasses: Record<Size, string> = {
		xs: 'text-xs px-2 py-0.5',
		sm: 'text-sm px-3 py-1'
	};
</script>

<span
	class={cn(
		'font-medium rounded-full inline-flex whitespace-nowrap items-center justify-center',
		variantClasses[variant],
		sizeClasses[size],
		className
	)}
	{...rest}
>
	{@render children?.()}
</span>
