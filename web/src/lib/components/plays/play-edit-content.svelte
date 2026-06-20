<script lang="ts">
	import Save from '@lucide/svelte/icons/save';
	import CircleX from '@lucide/svelte/icons/circle-x';
	import Button from '$lib/components/ui/button.svelte';
	import { Checkbox } from '$lib/components/ui/checkbox/index';
	import ActionConfirmDialog from '$lib/components/ui/dialog/action-confirm-dialog.svelte';
	import { TennisSlider } from '$lib/components/ui/slider/index';
	import { BADMINTON_LEVELS, DURATIONS, GAME_TYPES } from '$lib/consts/index';
	import { formatDate, formatTime } from '$lib/utils/formatting';
	import type { Play } from './types';

	type SelectItem = { value: string; label: string; disabled?: boolean };
	type EditFormValues = {
		name: string;
		description: string;
		visibility: string;
		date: string;
		start_time: string;
		duration_minutes: string;
		timezone: string;
		tz_offset: string;
		game_type: string;
		level_min: string;
		level_max: string;
		fee: string;
		max_players: string;
		courts: string;
	};
	type ActionForm =
		| { error?: string; intent?: 'update' | 'cancel'; values?: EditFormValues }
		| null
		| undefined;

	let { play, form = null }: { play: Play; form?: ActionForm } = $props();

	const confirmedParticipants = $derived(
		play.confirmed_participants ?? play.participant_preview ?? []
	);
	const confirmedCount = $derived(play.confirmed_count ?? confirmedParticipants.length);
	const minPlayers = $derived(Math.max(confirmedCount, 1));
	const TZ_OFFSET = '+08:00';
	const initialEditValues = initialEditFormValues();
	const editValues = $derived(form?.values ?? initialEditValues);
	let unlisted = $state(initialEditValues.visibility === 'unlisted');
	const editError = $derived(
		form?.intent === 'update' || form?.intent === 'cancel' ? form.error : undefined
	);
	const levelOptions = $derived(levelOptionsForSport(play.sport));
	const playTitle = $derived(play.name || play.venue_name);
	const durationItems = $derived(
		DURATIONS.some((item) => item.value === editValues.duration_minutes)
			? DURATIONS
			: [
					{ value: editValues.duration_minutes, label: `${editValues.duration_minutes} min` },
					...DURATIONS
				]
	);
	let tennisRange = $state(tennisRangeFromValues(initialEditValues));
	const tennisLevelMin = $derived(tennisRange[0]?.toFixed(1) ?? '');
	const tennisLevelMax = $derived(tennisRange[1]?.toFixed(1) ?? '');

	const fieldClass =
		'text-sm text-foreground px-3 border border-input-border rounded-lg bg-input h-9 w-full placeholder:text-muted-foreground focus:outline-none focus:border-ring';
	const textareaClass =
		'text-sm text-foreground px-3 py-2 border border-input-border rounded-lg bg-input min-h-20 w-full placeholder:text-muted-foreground focus:outline-none focus:border-ring';

	function initialEditFormValues() {
		return form?.values ?? editFormValuesFromPlay(play);
	}

	$effect(() => {
		if (form?.values) {
			unlisted = form.values.visibility === 'unlisted';
		}
	});

	function editFormValuesFromPlay(currentPlay: Play): EditFormValues {
		const startsAt = localDateTimeParts(currentPlay.starts_at, currentPlay.timezone);
		return {
			name: currentPlay.name ?? '',
			description: currentPlay.description ?? '',
			visibility: currentPlay.visibility ?? 'public',
			date: startsAt.date,
			start_time: startsAt.time,
			duration_minutes: durationMinutesValue(currentPlay),
			timezone: currentPlay.timezone || 'Asia/Singapore',
			tz_offset: TZ_OFFSET,
			game_type: currentPlay.game_type ?? '',
			level_min: currentPlay.level_min ?? '',
			level_max: currentPlay.level_max ?? '',
			fee: feeInputValue(currentPlay.fee),
			max_players:
				currentPlay.max_players == null ? String(minPlayers) : String(currentPlay.max_players),
			courts: currentPlay.courts == null ? '' : String(currentPlay.courts)
		};
	}

	function localDateTimeParts(iso: string, timezone: string) {
		try {
			const parts = new Intl.DateTimeFormat('en-GB', {
				timeZone: timezone || 'Asia/Singapore',
				year: 'numeric',
				month: '2-digit',
				day: '2-digit',
				hour: '2-digit',
				minute: '2-digit',
				hourCycle: 'h23'
			}).formatToParts(new Date(iso));
			const part = (type: Intl.DateTimeFormatPartTypes) =>
				parts.find((item) => item.type === type)?.value ?? '';
			return {
				date: `${part('year')}-${part('month')}-${part('day')}`,
				time: `${part('hour')}:${part('minute')}`
			};
		} catch {
			return { date: '', time: '' };
		}
	}

	function durationMinutesValue(currentPlay: Play) {
		const startsAt = new Date(currentPlay.starts_at).getTime();
		const endsAt = new Date(currentPlay.ends_at).getTime();
		const minutes = Math.round((endsAt - startsAt) / 60000);
		return Number.isFinite(minutes) && minutes > 0 ? String(minutes) : '120';
	}

	function feeInputValue(fee: number | null | undefined) {
		if (fee == null) return '';
		const dollars = fee / 100;
		return Number.isInteger(dollars) ? String(dollars) : dollars.toFixed(2);
	}

	function levelOptionsForSport(sport: Play['sport']): SelectItem[] {
		switch (sport) {
			case 'badminton':
				return BADMINTON_LEVELS;
			default:
				return [];
		}
	}

	function tennisRangeFromValues(values: EditFormValues) {
		const min = Number(values.level_min);
		const max = Number(values.level_max);
		return Number.isFinite(min) && Number.isFinite(max) ? [min, max] : [3, 4];
	}
</script>

<div class="mx-auto mt-8 pb-10 max-w-2xl w-full">
	<div class="mb-5">
		<h1 class="text-xl text-foreground font-bold">Edit game</h1>
		<p class="text-sm text-muted mt-1">
			{playTitle}{#if play.name}
				· {play.venue_name}
			{/if}
			· {formatDate(play.starts_at, play.timezone, { year: 'numeric' })} · {formatTime(
				play.starts_at,
				play.timezone
			)}
		</p>
	</div>

	<section class="p-4 border border-border rounded-md bg-card/50">
		{#if editError}
			<p class="text-sm text-red-700 mb-3 px-3 py-2 border border-red-200 rounded-md bg-red-50">
				{editError}
			</p>
		{/if}

		<form method="POST" action="?/update" class="space-y-3">
			<input type="hidden" name="timezone" value={editValues.timezone} />
			<input type="hidden" name="tz_offset" value={editValues.tz_offset} />
			<input type="hidden" name="visibility" value={unlisted ? 'unlisted' : 'public'} />

			<label for="edit-name" class="text-sm text-muted block">
				Name
				<input
					id="edit-name"
					name="name"
					type="text"
					maxlength={80}
					value={editValues.name}
					class="mt-1 {fieldClass}"
				/>
			</label>

			<label for="edit-description" class="text-sm text-muted block">
				Description
				<textarea
					id="edit-description"
					name="description"
					maxlength={1000}
					rows="3"
					class="mt-1 {textareaClass}">{editValues.description}</textarea
				>
			</label>

			<div class="gap-3 grid grid-cols-1 sm:grid-cols-2">
				<label for="edit-date" class="text-sm text-muted block">
					Date
					<input
						id="edit-date"
						name="date"
						type="date"
						value={editValues.date}
						required
						class="mt-1 {fieldClass}"
					/>
				</label>
				<label for="edit-start-time" class="text-sm text-muted block">
					Start
					<input
						id="edit-start-time"
						name="start_time"
						type="time"
						value={editValues.start_time}
						required
						class="mt-1 {fieldClass}"
					/>
				</label>
			</div>

			<div class="gap-3 grid grid-cols-1 sm:grid-cols-2">
				<label for="edit-duration" class="text-sm text-muted block">
					Duration
					<select id="edit-duration" name="duration_minutes" required class="mt-1 {fieldClass}">
						{#each durationItems as item (item.value)}
							<option value={item.value} selected={editValues.duration_minutes === item.value}>
								{item.label}
							</option>
						{/each}
					</select>
				</label>
				<label for="edit-game-type" class="text-sm text-muted block">
					Type
					<select id="edit-game-type" name="game_type" class="mt-1 {fieldClass}">
						<option value="" selected={editValues.game_type === ''}>Any</option>
						{#each GAME_TYPES as item (item.value)}
							<option value={item.value} selected={editValues.game_type === item.value}>
								{item.label}
							</option>
						{/each}
					</select>
				</label>
			</div>

			{#if play.sport === 'tennis'}
				<div>
					<input type="hidden" name="level_min" value={tennisLevelMin} />
					<input type="hidden" name="level_max" value={tennisLevelMax} />
					<TennisSlider bind:value={tennisRange} label="Level restriction" />
				</div>
			{:else if levelOptions.length > 0}
				<div class="gap-3 grid grid-cols-1 sm:grid-cols-2">
					<label for="edit-level-min" class="text-sm text-muted block">
						Min level
						<select id="edit-level-min" name="level_min" class="mt-1 {fieldClass}">
							<option value="" selected={editValues.level_min === ''}>Open</option>
							{#each levelOptions as item (item.value)}
								<option value={item.value} selected={editValues.level_min === item.value}>
									{item.label}
								</option>
							{/each}
						</select>
					</label>
					<label for="edit-level-max" class="text-sm text-muted block">
						Max level
						<select id="edit-level-max" name="level_max" class="mt-1 {fieldClass}">
							<option value="" selected={editValues.level_max === ''}>Open</option>
							{#each levelOptions as item (item.value)}
								<option value={item.value} selected={editValues.level_max === item.value}>
									{item.label}
								</option>
							{/each}
						</select>
					</label>
				</div>
			{:else}
				<input type="hidden" name="level_min" value="" />
				<input type="hidden" name="level_max" value="" />
			{/if}

			<div class="gap-3 grid grid-cols-1 sm:grid-cols-2">
				<label for="edit-max-players" class="text-sm text-muted block">
					Players
					<input
						id="edit-max-players"
						name="max_players"
						type="number"
						min={minPlayers}
						step="1"
						value={editValues.max_players}
						required
						class="mt-1 {fieldClass}"
					/>
				</label>
				<label for="edit-courts" class="text-sm text-muted block">
					Courts
					<input
						id="edit-courts"
						name="courts"
						type="number"
						min="1"
						step="1"
						value={editValues.courts}
						class="mt-1 {fieldClass}"
					/>
				</label>
			</div>

			<label for="edit-fee" class="text-sm text-muted block">
				Fee ($)
				<input
					id="edit-fee"
					name="fee"
					type="number"
					min="0"
					step="0.01"
					value={editValues.fee}
					class="mt-1 {fieldClass}"
				/>
			</label>

			<Checkbox
				bind:checked={unlisted}
				class="items-start"
				rootClass="mt-0.5"
				labelClass="grid gap-0.5"
			>
				<span>Set visibility as unlisted</span>
				<span class="text-muted">
					(Will not appear in public searches but anyone with the link can view)
				</span>
			</Checkbox>

			<Button type="submit" size="sm" class="mt-2 gap-1.5">
				<Save class="h-3.5 w-3.5" aria-hidden="true" />
				Save changes
			</Button>
		</form>

		<div class="mt-4 pt-4 border-t border-border">
			<ActionConfirmDialog
				title="Cancel game?"
				description="This action cannot be undone."
				action="?/cancelPlay"
				confirmLabel="Cancel game"
				confirmVariant="destructive"
				cancelLabel="Keep game"
			>
				{#snippet trigger({ props })}
					<Button type="button" size="sm" variant="outline" class="gap-1.5" {...props}>
						<CircleX class="h-3.5 w-3.5" aria-hidden="true" />
						Cancel game
					</Button>
				{/snippet}
			</ActionConfirmDialog>
		</div>
	</section>
</div>
