import { Checkbox as CheckboxPrimitive } from 'bits-ui';
import Checkbox from './checkbox.svelte';
import Root from './checkbox-root.svelte';
import Group from './checkbox-group.svelte';
import GroupLabel from './checkbox-group-label.svelte';

export {
	// Batteries-included component
	Checkbox,
	Checkbox as default,
	// Primitives for custom composition
	Root,
	Group,
	GroupLabel,
	//
	Root as CheckboxRoot,
	Group as CheckboxGroup,
	GroupLabel as CheckboxGroupLabel,
	CheckboxPrimitive
};
