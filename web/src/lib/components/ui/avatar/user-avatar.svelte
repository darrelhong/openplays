<script lang="ts">
	import { cn } from '$lib/utils/cn';
	import { Avatar, type WithoutChildrenOrChild } from 'bits-ui';

	let {
		src,
		nameForFallback,
		className = '',
		ref = $bindable(null),
		imageRef = $bindable(null),
		fallbackRef = $bindable(null),
		...restProps
	}: WithoutChildrenOrChild<Avatar.RootProps> & {
		src?: string;
		nameForFallback: string;
		className?: string;
		imageRef?: HTMLImageElement | null;
		fallbackRef?: HTMLElement | null;
	} = $props();
</script>

<Avatar.Root
	class={cn(
		'text-stone-300 font-medium border rounded-full bg-stone-800 h-8 w-8 uppercase data-[status=loaded]:border-stone-700 data-[status=loading]:border-transparent',
		className
	)}
	bind:ref
	{...restProps}
>
	<div
		class="border-2 border-transparent rounded-full flex h-full w-full items-center justify-center overflow-hidden"
	>
		<Avatar.Image bind:ref={imageRef} {src} alt={`${nameForFallback}'s avatar`} />
		<Avatar.Fallback bind:ref={fallbackRef}>{nameForFallback.charAt(0)}</Avatar.Fallback>
	</div>
</Avatar.Root>
