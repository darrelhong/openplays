<script lang="ts">
	import { page } from '$app/state';
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { onDestroy } from 'svelte';
	import type { SubmitFunction } from '@sveltejs/kit';
	import type { PageData } from './$types';
	import { FormField, TextInput, InputGroup } from '$lib/components/ui/form';
	import { Select } from '$lib/components/ui/select/index';
	import * as Dialog from '$lib/components/ui/dialog/index';
	import Button from '$lib/components/ui/button.svelte';
	import UserAvatar from '$lib/components/ui/avatar/user-avatar.svelte';
	import { BADMINTON_LEVELS } from '$lib/consts/index';
	import { PROFILE_LINK_PROVIDERS } from '$lib/utils/profile-links';
	import {
		formatTennisLevel,
		PROFILE_TENNIS_LEVELS,
		PROFILE_SPORT_OPTIONS,
		type ProfileSport
	} from '$lib/utils/sports-profile';
	import Plus from '@lucide/svelte/icons/plus';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Camera from '@lucide/svelte/icons/camera';

	let { data }: { data: PageData } = $props();

	// After successful update, use the returned user; otherwise use load data
	let user = $derived(page.form?.user ?? data.user);
	let displayName = $state(initialUser().display_name);
	let username = $state(initialUser().username ?? '');
	let activeSports = $state<ProfileSport[]>([...initialSportsProfileForm().activeSports]);
	let badmintonLevel = $state(initialSportsProfileForm().badmintonLevel);
	let tennisLevel = $state(formatTennisLevel(initialSportsProfileForm().tennisLevel));
	let bio = $state(initialProfileForm().bio);
	let profileLinks = $state({ ...initialProfileForm().profileLinksForm });
	let addSportOpen = $state(false);
	let avatarInput: HTMLInputElement;
	let avatarPreview = $state<string>();
	let avatarClientError = $state('');
	let avatarBusy = $state<'upload' | 'remove' | null>(null);
	const maxAvatarBytes = 5 * 1024 * 1024;

	let hasBadminton = $derived(activeSports.includes('badminton'));
	let hasTennis = $derived(activeSports.includes('tennis'));
	let availableSports = $derived(
		PROFILE_SPORT_OPTIONS.filter((sport) => !activeSports.includes(sport.value))
	);
	let displayedAvatar = $derived(avatarPreview ?? user.photo_url);
	let avatarError = $derived(
		avatarClientError || (avatarBusy === null ? page.form?.avatarError : '') || ''
	);
	let avatarSuccess = $derived(avatarBusy === null ? page.form?.avatarSuccess : '');

	const enhanceAvatar = (action: 'upload' | 'remove'): SubmitFunction => {
		return () => {
			avatarBusy = action;
			avatarClientError = '';
			return async ({ result, update }) => {
				try {
					if (
						result.type === 'failure' &&
						result.data &&
						typeof result.data.avatarError === 'string'
					) {
						avatarClientError = result.data.avatarError;
					}
					// Successful updates already invalidate all load data by default.
					await update();
				} finally {
					avatarBusy = null;
					resetAvatarSelection();
				}
			};
		};
	};

	function clearAvatarPreview() {
		if (avatarPreview) {
			URL.revokeObjectURL(avatarPreview);
			avatarPreview = undefined;
		}
	}

	function resetAvatarSelection() {
		clearAvatarPreview();
		if (avatarInput) avatarInput.value = '';
	}

	function selectAvatar(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		avatarClientError = '';
		if (!['image/jpeg', 'image/png'].includes(file.type)) {
			avatarClientError = 'Profile photo must be JPEG or PNG';
			input.value = '';
			return;
		}
		if (file.size > maxAvatarBytes) {
			avatarClientError = 'Profile photo must be 5 MB or smaller';
			input.value = '';
			return;
		}
		clearAvatarPreview();
		avatarPreview = URL.createObjectURL(file);
		input.form?.requestSubmit();
	}

	onDestroy(clearAvatarPreview);

	function initialSportsProfileForm() {
		return data.sportsProfileForm;
	}

	function initialUser() {
		return data.user;
	}

	function initialProfileForm() {
		return (
			page.form?.profileForm ?? {
				bio: data.bio,
				profileLinksForm: data.profileLinksForm
			}
		);
	}

	$effect(() => {
		const updatedUser = page.form?.user;
		if (updatedUser) {
			displayName = updatedUser.display_name;
			username = updatedUser.username ?? username;
		}

		const profileForm = page.form?.profileForm;
		if (profileForm) {
			bio = profileForm.bio;
			profileLinks = { ...profileForm.profileLinksForm };
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
		<div class="mb-6 flex gap-3 items-center justify-between">
			<h1 class="text-xl text-foreground font-bold">Edit Profile</h1>
			{#if user.username}
				<Button href={resolve(`/${user.username}`)} variant="outline" size="sm">
					View profile
				</Button>
			{/if}
		</div>

		<section class="mb-6 pb-6 border-b border-border flex gap-4 items-center">
			<UserAvatar
				src={displayedAvatar}
				nameForFallback={user.username ?? user.display_name}
				className="size-20 text-xl"
			/>
			<div class="flex flex-1 flex-col gap-2 items-start">
				<div class="flex flex-wrap gap-2">
					<form
						method="POST"
						action="?/avatar"
						enctype="multipart/form-data"
						use:enhance={enhanceAvatar('upload')}
					>
						<input
							bind:this={avatarInput}
							type="file"
							name="avatar"
							accept="image/jpeg,image/png"
							class="sr-only"
							onchange={selectAvatar}
						/>
						<Button
							type="button"
							variant="outline"
							size="sm"
							class="gap-1.5"
							disabled={avatarBusy !== null}
							onclick={() => avatarInput.click()}
						>
							<Camera class="size-4" />
							{avatarBusy === 'upload' ? 'Uploading…' : 'Change photo'}
						</Button>
					</form>
					{#if user.has_custom_avatar}
						<form method="POST" action="?/removeAvatar" use:enhance={enhanceAvatar('remove')}>
							<Button type="submit" variant="ghost" size="sm" disabled={avatarBusy !== null}>
								{avatarBusy === 'remove' ? 'Removing…' : 'Remove'}
							</Button>
						</form>
					{/if}
				</div>
				<p class="text-xs text-muted-foreground">JPEG or PNG, up to 5 MB.</p>
				{#if avatarError}
					<p class="text-sm text-destructive" role="alert">{avatarError}</p>
				{:else if avatarSuccess}
					<p class="text-sm text-success" role="status">
						{avatarSuccess}
					</p>
				{/if}
			</div>
		</section>

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

			<FormField label="Bio" id="bio">
				<textarea
					id="bio"
					name="bio"
					maxlength={500}
					rows="4"
					bind:value={bio}
					placeholder="Tell other players a little about yourself"
					class="text-sm text-foreground px-3 py-2 border border-input-border rounded-lg bg-input min-h-24 w-full resize-y placeholder:text-muted-foreground focus:outline-none focus:border-ring"
				></textarea>
				<p class="text-xs text-muted-foreground mt-1 text-right">{[...bio].length}/500</p>
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

			<section class="pt-4 border-t border-border flex flex-col gap-3">
				<div>
					<h2 class="text-sm text-foreground font-medium">Social media links</h2>
				</div>

				{#each PROFILE_LINK_PROVIDERS as provider (provider.key)}
					<FormField label={provider.label} id={`profile_link_${provider.key}`}>
						<TextInput
							id={`profile_link_${provider.key}`}
							name={`profile_link_${provider.key}`}
							bind:value={profileLinks[provider.key]}
							placeholder={provider.placeholder}
							maxlength={provider.maxLength}
							pattern={provider.pattern}
							inputmode={provider.inputMode}
							autocapitalize="none"
							autocomplete="off"
							spellcheck="false"
						/>
					</FormField>
				{/each}
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
