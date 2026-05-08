<script lang="ts">
	import { DateRangePicker as DateRangePickerPrimitive } from 'bits-ui';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import { cn } from '$lib/utils/cn';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: DateRangePickerPrimitive.CalendarProps = $props();
</script>

<DateRangePickerPrimitive.Calendar
	bind:ref
	data-slot="date-range-picker-calendar"
	class={cn('', className)}
	{...restProps}
>
	{#snippet children({ months, weekdays })}
		<DateRangePickerPrimitive.Header class="mb-4 flex items-center justify-between">
			<DateRangePickerPrimitive.PrevButton
				class="text-muted rounded-md inline-flex size-8 transition-colors items-center justify-center hover:text-foreground hover:bg-accent"
			>
				<ChevronLeft size={16} />
			</DateRangePickerPrimitive.PrevButton>
			<DateRangePickerPrimitive.Heading class="text-sm text-foreground font-medium" />
			<DateRangePickerPrimitive.NextButton
				class="text-muted rounded-md inline-flex size-8 transition-colors items-center justify-center hover:text-foreground hover:bg-accent"
			>
				<ChevronRight size={16} />
			</DateRangePickerPrimitive.NextButton>
		</DateRangePickerPrimitive.Header>
		<div class="flex flex-col gap-4 sm:flex-row">
			{#each months as month (month.value)}
				<DateRangePickerPrimitive.Grid class="w-full select-none border-collapse space-y-1">
					<DateRangePickerPrimitive.GridHead>
						<DateRangePickerPrimitive.GridRow class="mb-1 flex w-full justify-between">
							{#each weekdays as day (day)}
								<DateRangePickerPrimitive.HeadCell
									class="text-xs text-muted-foreground font-normal text-center w-8"
								>
									{day.slice(0, 2)}
								</DateRangePickerPrimitive.HeadCell>
							{/each}
						</DateRangePickerPrimitive.GridRow>
					</DateRangePickerPrimitive.GridHead>
					<DateRangePickerPrimitive.GridBody>
						{#each month.weeks as weekDates (weekDates)}
							<DateRangePickerPrimitive.GridRow class="flex w-full">
								{#each weekDates as date (date)}
									<DateRangePickerPrimitive.Cell
										{date}
										month={month.value}
										class="text-sm p-0 text-center size-8 relative"
									>
										<DateRangePickerPrimitive.Day
											class={cn(
												'text-sm text-foreground border border-transparent rounded-md bg-transparent inline-flex size-8 transition-colors items-center justify-center',
												'hover:border-border',
												'data-[highlighted]:rounded-none data-[highlighted]:bg-accent',
												'data-[range-middle]:rounded-none data-[range-middle]:bg-accent',
												'data-[selection-start]:text-primary-foreground data-[selection-start]:font-medium data-[selection-start]:rounded-md data-[selection-start]:bg-primary',
												'data-[selection-end]:text-primary-foreground data-[selection-end]:font-medium data-[selection-end]:rounded-md data-[selection-end]:bg-primary',
												'data-[range-start]:text-primary-foreground data-[range-start]:font-medium data-[range-start]:bg-primary',
												'data-[range-end]:text-primary-foreground data-[range-end]:font-medium data-[range-end]:bg-primary',
												'data-[selected]:font-medium',
												'data-[disabled]:text-muted-foreground data-[disabled]:pointer-events-none',
												'data-[unavailable]:text-muted-foreground data-[unavailable]:line-through data-[unavailable]:pointer-events-none',
												'data-[outside-month]:text-accent data-[outside-month]:pointer-events-none',
												'data-[today]:font-semibold',
												'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring'
											)}
										>
											{date.day}
										</DateRangePickerPrimitive.Day>
									</DateRangePickerPrimitive.Cell>
								{/each}
							</DateRangePickerPrimitive.GridRow>
						{/each}
					</DateRangePickerPrimitive.GridBody>
				</DateRangePickerPrimitive.Grid>
			{/each}
		</div>
	{/snippet}
</DateRangePickerPrimitive.Calendar>
