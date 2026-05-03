<script lang="ts">
	import { Select as SelectPrimitive, Label, type WithoutChildrenOrChild } from 'bits-ui';
	import SelectTrigger from './select-trigger.svelte';
	import SelectContent from './select-content.svelte';
	import SelectItem from './select-item.svelte';

	type Props = SelectPrimitive.RootProps & {
		items: { value: string; label: string; disabled?: boolean }[];
		placeholder?: string;
		label?: string;
		required?: boolean;
		contentProps?: WithoutChildrenOrChild<SelectPrimitive.ContentProps>;
	};

	let {
		items = [],
		value = $bindable(),
		placeholder = 'Select…',
		label,
		required = false,
		contentProps,
		...restProps
	}: Props = $props();

	const stableId = Math.random().toString(36).slice(2, 8);
	const triggerId = $derived(label ? `select-${stableId}` : undefined);

	const selectedLabel = $derived(items.find((item) => item.value === value)?.label);
</script>

<SelectPrimitive.Root {items} bind:value={value as never} {...restProps}>
	{#if label}
		<Label.Root
			for={triggerId}
			class="text-sm text-muted mb-1 block"
		>{label}{#if required}<span class="text-destructive/70 ml-0.5">*</span>{/if}</Label.Root>
	{/if}
	<SelectTrigger id={triggerId}>
		<span class="truncate" class:text-muted-foreground={!selectedLabel}>
			{selectedLabel ?? placeholder}
		</span>
	</SelectTrigger>
	<SelectContent {...contentProps}>
		{#each items as item (item.value)}
			<SelectItem value={item.value} label={item.label} disabled={item.disabled} />
		{/each}
	</SelectContent>
</SelectPrimitive.Root>
