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

			google.accounts.id.renderButton(signInContainer, {
				size: 'large'
			});
		}

		function handleCredentialResponse(response: { credential: string }) {
			credentialInput.value = response.credential;
			formEl.requestSubmit();
		}

		initGoogleSignIn();
	});

	// Facebook login disabled — requires Business Profile for app review
	// function loginWithFacebook() { ... }

	let error = $derived(
		(page.form?.error ?? page.url.searchParams.get('error')) as string | undefined
	);
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
				<div bind:this={signInContainer} id="g_id_signin" class="w-full"></div>

				<!-- Facebook login disabled — requires Business Profile for app review
				<button
					onclick={loginWithFacebook}
					class="flex w-full items-center justify-center gap-3 rounded-lg bg-[#1877F2] px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[#166FE5]"
				>
					<svg class="size-5 shrink-0" viewBox="0 0 24 24" fill="currentColor">
						<path d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z"/>
					</svg>
					Continue with Facebook
				</button>
				-->
			</div>

			<form bind:this={formEl} method="POST" action="?/google" class="hidden">
				<input bind:this={credentialInput} type="hidden" name="credential" />
			</form>
		</div>
	</div>
</div>
