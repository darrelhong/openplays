<script lang="ts">
	import { page } from '$app/state';
	import { enhance } from '$app/forms';
	import type { PageData } from './$types';
	import { FormField, TextInput, InputGroup } from '$lib/components/ui/form';
	import { Select } from '$lib/components/ui/select/index';
	import * as Dialog from '$lib/components/ui/dialog/index';
	import Button from '$lib/components/ui/button.svelte';
	import { BADMINTON_LEVELS } from '$lib/consts/index';
	import {
		formatTennisLevel,
		PROFILE_TENNIS_LEVELS,
		PROFILE_SPORT_OPTIONS,
		type ProfileSport
	} from '$lib/utils/sports-profile';
	import Plus from '@lucide/svelte/icons/plus';
	import Trash2 from '@lucide/svelte/icons/trash-2';

	let { data }: { data: PageData } = $props();

	// After successful update, use the returned user; otherwise use load data
	let user = $derived(page.form?.user ?? data.user);
	let displayName = $state(initialUser().display_name);
	let username = $state(initialUser().username ?? '');
	let activeSports = $state<ProfileSport[]>([...initialSportsProfileForm().activeSports]);
	let badmintonLevel = $state(initialSportsProfileForm().badmintonLevel);
	let tennisLevel = $state(formatTennisLevel(initialSportsProfileForm().tennisLevel));
	let addSportOpen = $state(false);

	let hasBadminton = $derived(activeSports.includes('badminton'));
	let hasTennis = $derived(activeSports.includes('tennis'));
	let availableSports = $derived(
		PROFILE_SPORT_OPTIONS.filter((sport) => !activeSports.includes(sport.value))
	);

	function initialSportsProfileForm() {
		return data.sportsProfileForm;
	}

	function initialUser() {
		return data.user;
	}

	$effect(() => {
		const updatedUser = page.form?.user;
		if (updatedUser) {
			displayName = updatedUser.display_name;
			username = updatedUser.username ?? username;
		}

		const profile = page.form?.sportsProfileForm;
		if (!profile) {
			return;
		}

		activeSports = [...profile.activeSports];
		badmintonLevel = profile.badmintonLevel;
		tennisLevel = formatTennisLevel(profile.tennisLevel);
	});

	function addSport(sport: ProfileSport) {
		if (!activeSports.includes(sport)) {
			activeSports = [...activeSports, sport];
		}
		addSportOpen = false;
	}

	function removeSport(sport: ProfileSport) {
		activeSports = activeSports.filter((activeSport) => activeSport !== sport);
		if (sport === 'badminton') {
			badmintonLevel = '';
		}
	}

</script>

<div class="mx-auto mt-8 max-w-md w-full">
	<div class="p-6 border border-border rounded-xl bg-card">
		<h1 class="text-xl text-foreground font-bold mb-6">Edit Profile</h1>

		<form method="POST" action="?/update" use:enhance class="flex flex-col gap-4">
			<FormField label="Email" id="email">
				<TextInput id="email" type="email" value={user.email} disabled />
			</FormField>

			<FormField label="Display Name" id="display_name">
				<TextInput
					id="display_name"
					name="display_name"
					bind:value={displayName}
					placeholder="Your display name"
					required
				/>
			</FormField>

			<FormField label="Username" id="username">
				<InputGroup prefix="@">
					<TextInput
						id="username"
						name="username"
						bind:value={username}
						placeholder="username"
						class="border-none bg-transparent"
						required
					/>
				</InputGroup>
			</FormField>

			<section class="flex flex-col gap-3">
				<div class="flex gap-3 items-center justify-between">
					<h2 class="text-sm text-foreground font-medium">Sports</h2>

					{#if availableSports.length > 0}
						<Dialog.Root bind:open={addSportOpen}>
							<Dialog.Trigger>
								{#snippet child({ props })}
									<Button variant="outline" size="sm" class="gap-1.5" {...props}>
										<Plus class="size-4" />
										Add Sport
									</Button>
								{/snippet}
							</Dialog.Trigger>
							<Dialog.Content
								class="border border-border w-[min(20rem,calc(100%-2rem))] shadow-card/30 shadow-lg sm:max-w-xs"
							>
								<Dialog.Header>
									<Dialog.Title>Add Sport</Dialog.Title>
								</Dialog.Header>

								<div class="gap-2 grid">
									{#each availableSports as sport (sport.value)}
										<button
											type="button"
											class="px-3 py-2 text-left border border-border rounded-lg bg-card transition-colors hover:bg-accent/70"
											onclick={() => addSport(sport.value)}
										>
											{sport.label}
										</button>
									{/each}
								</div>
							</Dialog.Content>
						</Dialog.Root>
					{/if}
				</div>

				{#if activeSports.length === 0}
					<div
						class="text-sm text-muted-foreground px-3 py-4 border border-border rounded-lg border-dashed"
					>
						No sports selected
					</div>
				{/if}

				{#if hasBadminton}
					<div class="p-4 border border-border rounded-lg bg-background">
						<div class="mb-3 flex gap-3 items-center justify-between">
							<h3 class="text-sm text-foreground font-medium">Badminton</h3>
							<Button
								type="button"
								variant="ghost"
								size="xs"
								class="px-2"
								aria-label="Remove badminton"
								onclick={() => removeSport('badminton')}
							>
								<Trash2 class="size-4" />
							</Button>
						</div>
						<input type="hidden" name="badminton_level" value={badmintonLevel} />
						<Select
							type="single"
							items={BADMINTON_LEVELS}
							bind:value={badmintonLevel}
							placeholder="Select level…"
							label="Level"
							allowDeselect
						/>
					</div>
				{/if}

				{#if hasTennis}
					<div class="p-4 border border-border rounded-lg bg-background">
						<div class="mb-4 flex gap-3 items-center justify-between">
							<div>
								<h3 class="text-sm text-foreground font-medium">Tennis</h3>
							</div>
							<Button
								type="button"
								variant="ghost"
								size="xs"
								class="px-2"
								aria-label="Remove tennis"
								onclick={() => removeSport('tennis')}
							>
								<Trash2 class="size-4" />
							</Button>
						</div>

						<input type="hidden" name="tennis_level" value={formatTennisLevel(tennisLevel)} />
						<Select
							type="single"
							items={PROFILE_TENNIS_LEVELS}
							bind:value={tennisLevel}
							placeholder="Select level…"
							label="Level (NTRP/USTA)"
						/>
					</div>
				{/if}
			</section>

			{#if page.form?.success}
				<p class="text-sm text-success text-center">Profile updated successfully.</p>
			{/if}

			{#if page.form?.error}
				<p class="text-sm text-destructive text-center">{page.form.error}</p>
			{/if}

			<button
				type="submit"
				class="text-primary-foreground font-medium py-2 rounded-lg bg-primary w-full transition-colors hover:bg-primary/85"
			>
				Save
			</button>
		</form>
	</div>
</div>
