<script lang="ts">
	import { Dialog as DialogPrimitive, type WithoutChildrenOrChild } from 'bits-ui';
	import DialogPortal from './dialog-portal.svelte';
	import type { Snippet } from 'svelte';
	import * as Dialog from './index';
	import { cn } from '$lib/utils/cn';
	import type { ComponentProps } from 'svelte';
	import Button from '$lib/components/ui/button.svelte';
	import XIcon from '@lucide/svelte/icons/x';

	type Variant = 'center' | 'right';

	let {
		ref = $bindable(null),
		class: className,
		portalProps,
		children,
		showCloseButton = true,
		variant = 'center',
		...restProps
	}: WithoutChildrenOrChild<DialogPrimitive.ContentProps> & {
		portalProps?: WithoutChildrenOrChild<ComponentProps<typeof DialogPortal>>;
		children: Snippet;
		showCloseButton?: boolean;
		variant?: Variant;
	} = $props();

	const baseClasses =
		'fixed z-50 text-sm p-6 outline-none bg-stone-900 text-stone-100 ring-1 ring-stone-500/10';

	const variantClasses: Record<Variant, string> = {
		center: cn(
			'left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2',
			'rounded-xl gap-6 grid max-w-[calc(100%-2rem)] w-full sm:max-w-md',
			'opacity-100 scale-100 transition-all duration-200 ease-out',
			'data-[state=closed]:opacity-0 data-[state=closed]:scale-95 data-[state=closed]:duration-150 data-[state=closed]:ease-in',
			'data-[starting-style]:opacity-0 data-[starting-style]:scale-95'
		),
		right: cn(
			'h-full max-w-md w-full right-0 top-0 lg:max-w-lg',
			'border-l border-stone-700 flex flex-col shadow-xl',
			'translate-x-0 transition-transform duration-200 ease-out',
			'data-[state=closed]:translate-x-full data-[state=closed]:duration-150 data-[state=closed]:ease-in',
			'data-[starting-style]:translate-x-full'
		)
	};
</script>

<DialogPortal {...portalProps}>
	<Dialog.Overlay />
	<DialogPrimitive.Content
		bind:ref
		data-slot="dialog-content"
		class={cn(baseClasses, variantClasses[variant], className)}
		{...restProps}
	>
		{@render children?.()}
		{#if showCloseButton}
			<DialogPrimitive.Close data-slot="dialog-close">
				{#snippet child({ props })}
					<Button variant="ghost" class="p-2 end-4 top-4 absolute" {...props}>
						<XIcon class="size-4" />
						<span class="sr-only">Close</span>
					</Button>
				{/snippet}
			</DialogPrimitive.Close>
		{/if}
	</DialogPrimitive.Content>
</DialogPortal>
