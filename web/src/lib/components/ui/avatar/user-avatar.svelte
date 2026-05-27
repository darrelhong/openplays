<script lang="ts">
	import { Avatar, type WithoutChildrenOrChild } from 'bits-ui';
	import BaseAvatar from './base-avatar.svelte';

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

<BaseAvatar {className} bind:ref {...restProps}>
	<Avatar.Image bind:ref={imageRef} {src} alt={`${nameForFallback}'s avatar`} />
	<Avatar.Fallback bind:ref={fallbackRef}>{nameForFallback.charAt(0)}</Avatar.Fallback>
</BaseAvatar>
