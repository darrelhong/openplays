<script lang="ts">
	import { browser } from '$app/environment';
	import { DateRangePicker as DateRangePickerPrimitive } from 'bits-ui';
	import type { DateValue } from '@internationalized/date';
	import DateRangePickerInput from './date-range-picker-input.svelte';
	import DateRangePickerSegment from './date-range-picker-segment.svelte';
	import DateRangePickerTrigger from './date-range-picker-trigger.svelte';
	import DateRangePickerContent from './date-range-picker-content.svelte';
	import DateRangePickerCalendar from './date-range-picker-calendar.svelte';
	import { cn } from '$lib/utils/cn';

	export type DateRange = {
		start: DateValue | undefined;
		end: DateValue | undefined;
	};

	const inputTypes = ['start', 'end'] as const;

	type Props = {
		value?: DateRange;
		placeholder?: DateValue;
		onValueChange?: (value: DateRange) => void;
		disabled?: boolean;
		minValue?: DateValue;
		maxValue?: DateValue;
		granularity?: 'day' | 'hour' | 'minute' | 'second';
		weekStartsOn?: 0 | 1 | 2 | 3 | 4 | 5 | 6;
		fixedWeeks?: boolean;
		locale?: string;
		closeOnRangeSelect?: boolean;
		label?: string;
		required?: boolean;
		numberOfMonths?: number;
		pagedNavigation?: boolean;
		class?: string;
	};

	let {
		value = $bindable({ start: undefined, end: undefined }),
		placeholder = $bindable(),
		onValueChange,
		disabled = false,
		minValue,
		maxValue,
		granularity = 'day',
		weekStartsOn,
		fixedWeeks = true,
		locale = "en-SG",
		closeOnRangeSelect = true,
		label,
		required = false,
		numberOfMonths = 2,
		pagedNavigation = true,
		class: className
	}: Props = $props();
</script>

<DateRangePickerPrimitive.Root
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
	{closeOnRangeSelect}
	{numberOfMonths}
	{pagedNavigation}
	calendarLabel={label ?? 'Date range'}
	weekdayFormat="short"
>
	<div class={className}>
		{#if label}
			<DateRangePickerPrimitive.Label class="text-sm text-muted mb-1 block">
				{label}{#if required}<span class="text-destructive/70 ml-0.5">*</span>{/if}
			</DateRangePickerPrimitive.Label>
		{/if}
		<div
			class={cn(
				'text-sm text-foreground pe-1.5 ps-3 border border-border rounded-lg bg-card h-9 w-full',
				'flex select-none items-center',
				'focus-within:outline-none focus-within:ring-1 focus-within:ring-ring'
			)}
		>
			{#each inputTypes as type (type)}
				<DateRangePickerInput {type}>
					{#snippet children({ segments })}
						{#each segments as { part, value: segmentValue }, i (part + i)}
							<DateRangePickerSegment {part}>
								{segmentValue}
							</DateRangePickerSegment>
						{/each}
					{/snippet}
				</DateRangePickerInput>
				{#if type === 'start'}
					<span aria-hidden="true" class="text-muted-foreground px-1">to</span>
				{/if}
			{/each}
			<DateRangePickerTrigger />
		</div>
	</div>
	<DateRangePickerContent>
		<DateRangePickerCalendar />
	</DateRangePickerContent>
</DateRangePickerPrimitive.Root>
