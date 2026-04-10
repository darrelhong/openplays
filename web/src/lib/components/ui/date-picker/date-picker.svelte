<script lang="ts">
	import { DatePicker as DatePickerPrimitive } from 'bits-ui';
	import type { DateValue } from '@internationalized/date';
	import DatePickerInput from './date-picker-input.svelte';
	import DatePickerSegment from './date-picker-segment.svelte';
	import DatePickerTrigger from './date-picker-trigger.svelte';
	import DatePickerContent from './date-picker-content.svelte';
	import DatePickerCalendar from './date-picker-calendar.svelte';

	type Props = {
		value?: DateValue;
		placeholder?: DateValue;
		onValueChange?: (value: DateValue | undefined) => void;
		disabled?: boolean;
		minValue?: DateValue;
		maxValue?: DateValue;
		granularity?: 'day' | 'hour' | 'minute' | 'second';
		weekStartsOn?: 0 | 1 | 2 | 3 | 4 | 5 | 6;
		fixedWeeks?: boolean;
		locale?: string;
		closeOnDateSelect?: boolean;
		label?: string;
		class?: string;
	};

	let {
		value = $bindable(),
		placeholder = $bindable(),
		onValueChange,
		disabled = false,
		minValue,
		maxValue,
		granularity = 'day',
		weekStartsOn,
		fixedWeeks = true,
		locale = 'en',
		closeOnDateSelect = true,
		label,
		class: className
	}: Props = $props();
</script>

<DatePickerPrimitive.Root
	bind:value
	bind:placeholder
	{onValueChange}
	{disabled}
	{minValue}
	{maxValue}
	{granularity}
	{weekStartsOn}
	{fixedWeeks}
	{locale}
	{closeOnDateSelect}
	weekdayFormat="short"
>
	<div class={className}>
		{#if label}
			<DatePickerPrimitive.Label class="text-sm text-stone-400 mb-1 block">
				{label}
			</DatePickerPrimitive.Label>
		{/if}
		<DatePickerInput>
			{#snippet children({ segments })}
				{#each segments as { part, value: segmentValue }, i (part + i)}
					<DatePickerSegment {part}>
						{segmentValue}
					</DatePickerSegment>
				{/each}
				<DatePickerTrigger />
			{/snippet}
		</DatePickerInput>
	</div>
	<DatePickerContent>
		<DatePickerCalendar />
	</DatePickerContent>
</DatePickerPrimitive.Root>
