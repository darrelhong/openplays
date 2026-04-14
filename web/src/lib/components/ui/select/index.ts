import Select from './select.svelte';
import Trigger from './select-trigger.svelte';
import Content from './select-content.svelte';
import Item from './select-item.svelte';
import ScrollUpButton from './select-scroll-up-button.svelte';
import ScrollDownButton from './select-scroll-down-button.svelte';

export {
	// Batteries-included component
	Select,
	Select as default,
	// Primitives for custom composition
	Trigger,
	Content,
	Item,
	ScrollUpButton,
	ScrollDownButton,
	//
	Trigger as SelectTrigger,
	Content as SelectContent,
	Item as SelectItem,
	ScrollUpButton as SelectScrollUpButton,
	ScrollDownButton as SelectScrollDownButton
};
