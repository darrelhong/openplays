<script lang="ts">
	import { ScrollArea, type WithoutChild } from 'bits-ui';
	import { ScrollbarVertical, ScrollbarHorizontal } from '.';
	type Props = WithoutChild<ScrollArea.RootProps> & {
		orientation: 'vertical' | 'horizontal' | 'both';
		viewportClasses?: string;
	};

	let {
		ref = $bindable(null),
		orientation = 'vertical',
		viewportClasses,
		children,
		...restProps
	}: Props = $props();
</script>

<ScrollArea.Root bind:ref {...restProps}>
	<ScrollArea.Viewport class={viewportClasses}>
		{@render children?.()}
	</ScrollArea.Viewport>
	{#if orientation === 'vertical' || orientation === 'both'}
		<ScrollbarVertical />
	{/if}
	{#if orientation === 'horizontal' || orientation === 'both'}
		<ScrollbarHorizontal />
	{/if}
	<ScrollArea.Corner />
</ScrollArea.Root>
