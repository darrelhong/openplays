import Combobox from './combobox.svelte';
import Root from './combobox-root.svelte';
import Input from './combobox-input.svelte';
import Trigger from './combobox-trigger.svelte';
import Content from './combobox-content.svelte';
import Item from './combobox-item.svelte';
import Empty from './combobox-empty.svelte';
import ScrollUpButton from './combobox-scroll-up-button.svelte';
import ScrollDownButton from './combobox-scroll-down-button.svelte';

export {
	// Batteries-included component
	Combobox,
	Combobox as default,
	// Primitives for custom composition
	Root,
	Input,
	Trigger,
	Content,
	Item,
	Empty,
	ScrollUpButton,
	ScrollDownButton,
	//
	Root as ComboboxRoot,
	Input as ComboboxInput,
	Trigger as ComboboxTrigger,
	Content as ComboboxContent,
	Item as ComboboxItem,
	Empty as ComboboxEmpty,
	ScrollUpButton as ComboboxScrollUpButton,
	ScrollDownButton as ComboboxScrollDownButton
};
