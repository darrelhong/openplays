import DateRangePicker from './date-range-picker.svelte';
import Input from './date-range-picker-input.svelte';
import Segment from './date-range-picker-segment.svelte';
import Trigger from './date-range-picker-trigger.svelte';
import Content from './date-range-picker-content.svelte';
import Calendar from './date-range-picker-calendar.svelte';

export {
	// Batteries-included component
	DateRangePicker,
	DateRangePicker as default,
	// Primitives for custom composition
	Input,
	Segment,
	Trigger,
	Content,
	Calendar,
	//
	Input as DateRangePickerInput,
	Segment as DateRangePickerSegment,
	Trigger as DateRangePickerTrigger,
	Content as DateRangePickerContent,
	Calendar as DateRangePickerCalendar
};
