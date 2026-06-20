<script lang="ts">
	import { Checkbox as CheckboxPrimitive } from 'bits-ui';
	import type { ClassValue } from 'svelte/elements';
	import { cn } from '$lib/utils/cn';

	type Props = CheckboxPrimitive.GroupLabelProps & {
		class?: ClassValue;
	};

	let { ref = $bindable(null), class: className, children, ...restProps }: Props = $props();
</script>

<CheckboxPrimitive.GroupLabel
	bind:ref
	data-slot="checkbox-group-label"
	class={cn('text-sm text-foreground font-medium', 'data-[disabled]:opacity-50', className)}
	{...restProps}
>
	{@render children?.()}
</CheckboxPrimitive.GroupLabel>
