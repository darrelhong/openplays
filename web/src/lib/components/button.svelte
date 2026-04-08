<script lang="ts">
	import type { ClassValue } from 'svelte/elements';
	import { Button, type ButtonRootProps } from 'bits-ui';

	type Variant = 'default' | 'secondary' | 'outline' | 'ghost';
	type Size = 'sm' | 'md' | 'lg';

	type Props = ButtonRootProps & {
		variant?: Variant;
		size?: Size;
		class?: ClassValue;
	};

	let { variant = 'default', size = 'md', children, class: className, ...rest }: Props = $props();

	const variantClasses: Record<Variant, string> = {
		default: 'text-stone-950 bg-stone-100 hover:bg-stone-100/85',
		secondary: 'text-stone-100 bg-stone-700 hover:bg-stone-700/80',
		outline:
			'text-stone-100 border border-stone-500 bg-stone-800 shadow-xl hover:bg-stone-700/70',
		ghost: 'text-stone-100 hover:bg-stone-800'
	};
	//  default: "bg-primary text-primary-foreground hover:bg-primary/80",
    //     secondary: "bg-secondary text-secondary-foreground hover:bg-secondary/80 aria-expanded:bg-secondary aria-expanded:text-secondary-foreground",
    //     outline: "border-border bg-background hover:bg-muted hover:text-foreground dark:bg-input/30 dark:border-input dark:hover:bg-input/50 aria-expanded:bg-muted aria-expanded:text-foreground shadow-xs",
    //     ghost: "hover:bg-muted hover:text-foreground dark:hover:bg-muted/50 aria-expanded:bg-muted aria-expanded:text-foreground",
    //     destructive: "bg-destructive/10 hover:bg-destructive/20 focus-visible:ring-destructive/20 dark:focus-visible:ring-destructive/40 dark:bg-destructive/20 text-destructive focus-visible:border-destructive/40 dark:hover:bg-destructive/30",
    //     link: "text-primary underline-offset-4 hover:underline",

	const sizeClasses: Record<Size, string> = {
		sm: 'text-sm px-2.5 h-8',
		md: 'text-base px-3 py-1.5',
		lg: 'text-base px-2.5 h-10'
	};
</script>

<Button.Root
	class={[
		'font-medium rounded-lg inline-flex cursor-pointer transition-colors items-center justify-center',
		'focus-visible:outline-2 focus-visible:outline-stone-400 focus-visible:outline-offset-1',
		'disabled:opacity-50 disabled:pointer-events-none',
		variantClasses[variant],
		sizeClasses[size],
		className
	]}
	{...rest}
>
	{@render children?.()}
</Button.Root>
