<script lang="ts">
	import { Combobox, type WithoutChildrenOrChild, mergeProps } from 'bits-ui';
	import ComboboxInput from './combobox-input.svelte';
	import ComboboxTrigger from './combobox-trigger.svelte';
	import ComboboxContent from './combobox-content.svelte';
	import ComboboxItem from './combobox-item.svelte';
	import ComboboxEmpty from './combobox-empty.svelte';

	type Props = Combobox.RootProps & {
		placeholder?: string;
		openOnClick?: boolean;
		allowCustom?: boolean;
		inputProps?: WithoutChildrenOrChild<Combobox.InputProps>;
		contentProps?: WithoutChildrenOrChild<Combobox.ContentProps>;
	};

	let {
		items = [],
		value = $bindable(),
		open = $bindable(false),
		placeholder = 'Search…',
		openOnClick = false,
		allowCustom = false,
		inputProps,
		contentProps,
		type,
		...restProps
	}: Props = $props();

	let searchValue = $state('');

	const filteredItems = $derived.by(() => {
		if (searchValue === '') return items;
		return items!.filter((item) => item.label.toLowerCase().includes(searchValue.toLowerCase()));
	});

	// Sync input text with the selected value's label.
	// When open, show nothing so the user can type to search.
	// When closed, show the selected item's label (or custom value if allowCustom).
	const inputValue = $derived.by(() => {
		if (open) return undefined;
		const selected = items?.find((item) => item.value === value);
		if (selected) return selected.label;
		// If allowCustom, show the raw value as typed
		if (allowCustom && value) return value as string;
		return '';
	});

	function handleInput(e: Event & { currentTarget: HTMLInputElement }) {
		searchValue = e.currentTarget.value;
	}

	function handleOpenChange(newOpen: boolean) {
		if (!newOpen) {
			// When closing: if allowCustom and nothing was selected from list, use typed text
			if (allowCustom && searchValue && !items?.find((item) => item.value === value)) {
				value = searchValue as typeof value;
			}
			searchValue = '';
		}
	}

	const mergedRootProps = $derived(mergeProps(restProps, { onOpenChange: handleOpenChange }));

	const mergedInputProps = $derived(
		mergeProps(inputProps ?? {}, {
			oninput: handleInput,
			onfocus: openOnClick ? () => (open = true) : undefined,
			onclick: openOnClick ? () => (open = true) : undefined,
			placeholder,
			clearOnDeselect: true
		})
	);
</script>

<Combobox.Root
	{type}
	{items}
	{inputValue}
	bind:value={value as never}
	bind:open
	{...mergedRootProps}
>
	<div class="relative">
		<ComboboxInput {...mergedInputProps} />
		<ComboboxTrigger />
	</div>
	<ComboboxContent {...contentProps}>
		{#each filteredItems as item, i (i + item.value)}
			<ComboboxItem value={item.value} label={item.label} disabled={item.disabled} />
		{:else}
			<ComboboxEmpty />
		{/each}
	</ComboboxContent>
</Combobox.Root>
