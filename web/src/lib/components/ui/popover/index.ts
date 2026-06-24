import { Popover as PopoverPrimitive } from 'bits-ui';
import Root from './popover.svelte';
import Trigger from './popover-trigger.svelte';
import Content from './popover-content.svelte';
import Close from './popover-close.svelte';

const Portal = PopoverPrimitive.Portal;
const Arrow = PopoverPrimitive.Arrow;

export {
	Root,
	Trigger,
	Content,
	Close,
	Portal,
	Arrow,
	PopoverPrimitive,
	Root as Popover,
	Trigger as PopoverTrigger,
	Content as PopoverContent,
	Close as PopoverClose,
	Portal as PopoverPortal,
	Arrow as PopoverArrow
};
