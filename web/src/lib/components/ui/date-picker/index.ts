import DatePicker from './date-picker.svelte';
import Input from './date-picker-input.svelte';
import Segment from './date-picker-segment.svelte';
import Trigger from './date-picker-trigger.svelte';
import Content from './date-picker-content.svelte';
import Calendar from './date-picker-calendar.svelte';

export {
	// Batteries-included component
	DatePicker,
	DatePicker as default,
	// Primitives for custom composition
	Input,
	Segment,
	Trigger,
	Content,
	Calendar,
	//
	Input as DatePickerInput,
	Segment as DatePickerSegment,
	Trigger as DatePickerTrigger,
	Content as DatePickerContent,
	Calendar as DatePickerCalendar
};
