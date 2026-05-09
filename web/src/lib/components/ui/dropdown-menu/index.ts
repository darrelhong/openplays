import { DropdownMenu as DropdownMenuPrimitive } from 'bits-ui';
import Content from './dropdown-menu-content.svelte';
import Item from './dropdown-menu-item.svelte';
import Separator from './dropdown-menu-separator.svelte';

const Root = DropdownMenuPrimitive.Root;
const Trigger = DropdownMenuPrimitive.Trigger;
const Portal = DropdownMenuPrimitive.Portal;
const Group = DropdownMenuPrimitive.Group;
const GroupHeading = DropdownMenuPrimitive.GroupHeading;
const CheckboxItem = DropdownMenuPrimitive.CheckboxItem;
const CheckboxGroup = DropdownMenuPrimitive.CheckboxGroup;
const RadioGroup = DropdownMenuPrimitive.RadioGroup;
const RadioItem = DropdownMenuPrimitive.RadioItem;
const Sub = DropdownMenuPrimitive.Sub;
const SubTrigger = DropdownMenuPrimitive.SubTrigger;
const SubContent = DropdownMenuPrimitive.SubContent;
const Arrow = DropdownMenuPrimitive.Arrow;

export {
	Root,
	Trigger,
	Portal,
	Content,
	Item,
	Separator,
	Group,
	GroupHeading,
	CheckboxItem,
	CheckboxGroup,
	RadioGroup,
	RadioItem,
	Sub,
	SubTrigger,
	SubContent,
	Arrow,
	Root as DropdownMenu,
	Trigger as DropdownMenuTrigger,
	Portal as DropdownMenuPortal,
	Content as DropdownMenuContent,
	Item as DropdownMenuItem,
	Separator as DropdownMenuSeparator,
	Group as DropdownMenuGroup,
	GroupHeading as DropdownMenuGroupHeading,
	CheckboxItem as DropdownMenuCheckboxItem,
	CheckboxGroup as DropdownMenuCheckboxGroup,
	RadioGroup as DropdownMenuRadioGroup,
	RadioItem as DropdownMenuRadioItem,
	Sub as DropdownMenuSub,
	SubTrigger as DropdownMenuSubTrigger,
	SubContent as DropdownMenuSubContent,
	Arrow as DropdownMenuArrow
};
