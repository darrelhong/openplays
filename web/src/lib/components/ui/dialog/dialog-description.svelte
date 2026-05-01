<script lang="ts">
	import { Dialog as DialogPrimitive } from 'bits-ui';
	import { cn } from '$lib/utils/cn';
	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: DialogPrimitive.DescriptionProps = $props();
</script>

<DialogPrimitive.Description
	bind:ref
	data-slot="dialog-description"
	class={cn(
		'*:[a]:hover:text-foreground *:[a]:underline *:[a]:underline-offset-3 text-sm text-muted',
		className
	)}
	{...restProps}
/>
