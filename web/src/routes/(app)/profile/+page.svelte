<script lang="ts">
	import { page } from '$app/state';
	import { enhance } from '$app/forms';
	import type { PageData } from './$types';
	import { FormField, TextInput, InputGroup } from '$lib/components/ui/form';

	let { data }: { data: PageData } = $props();

	// After successful update, use the returned user; otherwise use load data
	let user = $derived(page.form?.user ?? data.user);
</script>

<div class="mx-auto mt-8 max-w-lg">
	<div class="p-6 border border-border rounded-xl bg-card">
		<h1 class="text-xl text-foreground font-bold mb-6">Edit Profile</h1>

		{#if page.form?.success}
			<p class="text-sm text-success mb-4">Profile updated successfully.</p>
		{/if}

		{#if page.form?.error}
			<p class="text-sm text-destructive mb-4">{page.form.error}</p>
		{/if}

		<form method="POST" action="?/update" use:enhance class="flex flex-col gap-4">
			<FormField label="Email" id="email">
				<TextInput id="email" type="email" value={user.email} disabled />
			</FormField>

			<FormField label="Display Name" id="display_name">
				<TextInput
					id="display_name"
					name="display_name"
					value={user.display_name}
					placeholder="Your display name"
					required
				/>
			</FormField>

			<FormField label="Username" id="username">
				<InputGroup prefix="@">
					<TextInput
						id="username"
						name="username"
						value={user.username ?? ''}
						placeholder="username"
						class="border-none bg-transparent"
						required
					/>
				</InputGroup>
			</FormField>

			<button
				type="submit"
				class="text-primary-foreground font-medium mt-4 py-2 rounded-lg bg-primary w-full transition-colors hover:bg-primary/85"
			>
				Save
			</button>
		</form>
	</div>
</div>
