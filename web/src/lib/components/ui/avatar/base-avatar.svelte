<script lang="ts">
	import { cn } from '$lib/utils/cn';
	import { Avatar } from 'bits-ui';
	import type { ClassValue } from 'svelte/elements';
	import type { Snippet } from 'svelte';

	type Variant = 'default' | 'dotted';

	type Props = Avatar.RootProps & {
		variant?: Variant;
		className?: ClassValue;
		contentClassName?: ClassValue;
		children?: Snippet;
	};

	const variantClasses: Record<Variant, string> = {
		default:
			'border bg-card data-[status=loaded]:border-border data-[status=loading]:border-transparent',
		dotted: 'border border-muted-foreground/50 border-dashed bg-background'
	};

	let {
		variant = 'default',
		className = '',
		contentClassName = '',
		children,
		ref = $bindable(null),
		...restProps
	}: Props = $props();
</script>

<Avatar.Root
	class={cn(
		'text-muted font-medium rounded-full shrink-0 h-8 w-8 uppercase',
		variantClasses[variant],
		className
	)}
	bind:ref
	{...restProps}
>
	<div
		class={cn(
			'border-2 border-transparent rounded-full flex h-full w-full items-center justify-center overflow-hidden',
			contentClassName
		)}
	>
		{@render children?.()}
	</div>
</Avatar.Root>
