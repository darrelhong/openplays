<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { PUBLIC_GOOGLE_CLIENT_ID } from '$env/static/public';

	let formEl: HTMLFormElement;
	let credentialInput: HTMLInputElement;
	let signInContainer: HTMLDivElement;

	interface GoogleAccounts {
		id: {
			initialize: (config: {
				client_id: string;
				callback: (response: { credential: string }) => void;
				ux_mode: string;
			}) => void;
			renderButton: (
				element: HTMLElement,
				config: { theme?: 'filled_black'; size?: 'small' | 'large' }
			) => void;
		};
	}

	function getGoogle(): { accounts: GoogleAccounts } | undefined {
		return (window as unknown as { google?: { accounts: GoogleAccounts } }).google;
	}

	onMount(() => {
		function initGoogleSignIn() {
			const google = getGoogle();
			if (!google) {
				setTimeout(initGoogleSignIn, 100);
				return;
			}

			google.accounts.id.initialize({
				client_id: PUBLIC_GOOGLE_CLIENT_ID,
				callback: handleCredentialResponse,
				ux_mode: 'popup'
			});

			google.accounts.id.renderButton(signInContainer, {});
		}

		function handleCredentialResponse(response: { credential: string }) {
			credentialInput.value = response.credential;
			formEl.requestSubmit();
		}

		initGoogleSignIn();
	});

	let error = $derived(page.form?.error as string | undefined);
</script>

<svelte:head>
	<script src="https://accounts.google.com/gsi/client" async defer></script>
</svelte:head>

<div class="flex-1 grid place-items-center">
	<div class="mx-4 p-6 pb-7 border border-border rounded-xl bg-card max-w-sm w-full">
		<div class="flex flex-col gap-6 items-center">
			<div class="mb-4 text-center flex flex-col gap-2 items-center">
				<h1 class="text-2xl text-foreground font-bold">Welcome to OpenPlays</h1>
				<p class="text-sm text-muted">Sign up or log in to create and manage your games.</p>
			</div>

			{#if error}
				<p class="text-sm text-destructive">{error}</p>
			{/if}

			<div class="text-center h-10">
				<div id="g_id_signin" bind:this={signInContainer} class="min-w-0 w-auto"></div>
			</div>

			<form bind:this={formEl} method="POST" action="?/google" class="hidden">
				<input bind:this={credentialInput} type="hidden" name="credential" />
			</form>
		</div>
	</div>
</div>
