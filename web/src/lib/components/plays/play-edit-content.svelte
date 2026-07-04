<script lang="ts">
	import Save from '@lucide/svelte/icons/save';
	import CircleX from '@lucide/svelte/icons/circle-x';
	import Button from '$lib/components/ui/button.svelte';
	import ActionConfirmDialog from '$lib/components/ui/dialog/action-confirm-dialog.svelte';
	import { formatDate, formatTime } from '$lib/utils/formatting';
	import { minEditableMaxPlayers } from '$lib/utils/play-capacity';
	import PlayFormFields from './play-form-fields.svelte';
	import type { Play } from './types';

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
		require_waitlist: string;
	};
	type ActionForm =
		| { error?: string; intent?: 'update' | 'cancel'; values?: EditFormValues }
		| null
		| undefined;

	let { play, form = null }: { play: Play; form?: ActionForm } = $props();

	const minPlayers = $derived(minEditableMaxPlayers(play));
	const TZ_OFFSET = '+08:00';
	const editValues = initialEditFormValues();
	const editError = $derived(
		form?.intent === 'update' || form?.intent === 'cancel' ? form.error : undefined
	);
	const playTitle = $derived(play.name || play.venue_name);

	function initialEditFormValues() {
		return form?.values ?? editFormValuesFromPlay(play);
	}

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
			courts: currentPlay.courts == null ? '' : String(currentPlay.courts),
			require_waitlist: currentPlay.require_waitlist ? 'true' : 'false'
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
			<PlayFormFields
				sport={play.sport}
				values={editValues}
				{minPlayers}
				timezone={editValues.timezone}
				tzOffset={editValues.tz_offset}
			/>

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
