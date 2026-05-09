<script lang="ts">
	import { DatePicker as DatePickerPrimitive } from 'bits-ui';
	import ChevronLeft from '@lucide/svelte/icons/chevron-left';
	import ChevronRight from '@lucide/svelte/icons/chevron-right';
	import { cn } from '$lib/utils/cn';

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: DatePickerPrimitive.CalendarProps = $props();
</script>

<DatePickerPrimitive.Calendar
	bind:ref
	data-slot="date-picker-calendar"
	class={cn('', className)}
	{...restProps}
>
	{#snippet children({ months, weekdays })}
		<DatePickerPrimitive.Header class="mb-4 flex items-center justify-between">
			<DatePickerPrimitive.PrevButton
				class="text-muted rounded-md inline-flex size-8 transition-colors items-center justify-center hover:text-foreground hover:bg-accent"
			>
				<ChevronLeft size={16} />
			</DatePickerPrimitive.PrevButton>
			<DatePickerPrimitive.Heading class="text-sm text-foreground font-medium" />
			<DatePickerPrimitive.NextButton
				class="text-muted rounded-md inline-flex size-8 transition-colors items-center justify-center hover:text-foreground hover:bg-accent"
			>
				<ChevronRight size={16} />
			</DatePickerPrimitive.NextButton>
		</DatePickerPrimitive.Header>
		{#each months as month (month.value)}
			<DatePickerPrimitive.Grid class="w-full select-none border-collapse space-y-1">
				<DatePickerPrimitive.GridHead>
					<DatePickerPrimitive.GridRow class="mb-1 flex w-full justify-between">
						{#each weekdays as day (day)}
							<DatePickerPrimitive.HeadCell
								class="text-xs text-muted-foreground font-normal text-center w-8"
							>
								{day.slice(0, 2)}
							</DatePickerPrimitive.HeadCell>
						{/each}
					</DatePickerPrimitive.GridRow>
				</DatePickerPrimitive.GridHead>
				<DatePickerPrimitive.GridBody>
					{#each month.weeks as weekDates (weekDates)}
						<DatePickerPrimitive.GridRow class="flex w-full">
							{#each weekDates as date (date)}
								<DatePickerPrimitive.Cell
									{date}
									month={month.value}
									class="text-sm p-0 text-center size-8 relative"
								>
									<DatePickerPrimitive.Day
										class={cn(
											'text-sm text-foreground border border-transparent rounded-md bg-transparent inline-flex size-8 transition-colors items-center justify-center',
											'hover:border-border',
											'data-[selected]:text-primary-foreground data-[selected]:font-medium data-[selected]:bg-primary',
											'data-[disabled]:text-muted-foreground data-[disabled]:pointer-events-none',
											'data-[unavailable]:text-muted-foreground data-[unavailable]:line-through data-[unavailable]:pointer-events-none',
											'data-[outside-month]:text-accent data-[outside-month]:pointer-events-none',
											'data-[today]:font-semibold',
											'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring'
										)}
									>
										{date.day}
									</DatePickerPrimitive.Day>
								</DatePickerPrimitive.Cell>
							{/each}
						</DatePickerPrimitive.GridRow>
					{/each}
				</DatePickerPrimitive.GridBody>
			</DatePickerPrimitive.Grid>
		{/each}
	{/snippet}
</DatePickerPrimitive.Calendar>
