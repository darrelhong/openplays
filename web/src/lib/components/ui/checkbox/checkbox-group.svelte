<script lang="ts">
	import { Checkbox as CheckboxPrimitive } from 'bits-ui';
	import type { ClassValue } from 'svelte/elements';
	import { cn } from '$lib/utils/cn';

	type Props = CheckboxPrimitive.GroupProps & {
		class?: ClassValue;
	};

	let {
		ref = $bindable(null),
		value = $bindable([]),
		class: className,
		children,
		...restProps
	}: Props = $props();
</script>

<CheckboxPrimitive.Group
	bind:ref
	bind:value
	data-slot="checkbox-group"
	class={cn('gap-2 grid', 'data-[disabled]:opacity-50', className)}
	{...restProps}
>
	{@render children?.()}
</CheckboxPrimitive.Group>
