<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
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
	let otpSent = $derived(page.form?.otpSent as boolean | undefined);
	let otpEmail = $derived(page.form?.email as string | undefined);
</script>

<svelte:head>
	<script src="https://accounts.google.com/gsi/client" async defer></script>
</svelte:head>

<div class="flex-1 grid place-items-center">
	<div class="mx-4 p-6 pb-7 border border-border rounded-xl bg-card max-w-sm w-full">
		<div class="flex flex-col gap-6 items-center">
			<div class="mb-4 text-center flex flex-col gap-2 items-center">
				<h1 class="text-2xl text-foreground font-bold">Welcome back</h1>
				<p class="text-sm text-muted">Sign in to keep track of your games.</p>
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

			<div class="text-xs text-muted flex gap-3 w-full items-center">
				<div class="bg-border flex-1 h-px"></div>
				<span>or use email</span>
				<div class="bg-border flex-1 h-px"></div>
			</div>

			{#if otpSent}
				<div class="text-sm text-muted p-4 text-center border border-border rounded-lg bg-muted/30">
					<p class="text-foreground font-medium">Check your inbox</p>
					<p class="mt-1">
						We sent a 6-digit code to <span class="text-foreground font-medium">{otpEmail}</span>.
						Enter it below to finish signing in.
					</p>
				</div>

				<form method="POST" action="?/email-otp-verify" class="flex flex-col gap-3 w-full">
					<input type="hidden" name="email" value={otpEmail} />
					<label for="email-otp-code" class="text-sm text-foreground font-medium">Your code</label>
					<input
						id="email-otp-code"
						name="code"
						type="text"
						inputmode="numeric"
						autocomplete="one-time-code"
						pattern="[0-9]*"
						maxlength="6"
						minlength="6"
						placeholder="123456"
						title="Enter the 6-digit code from your email"
						required
						oninput={(event) => {
							const input = event.currentTarget as HTMLInputElement;
							input.value = input.value.replace(/\D/g, '').slice(0, 6);
						}}
						class="text-lg text-foreground tracking-[0.3em] px-3 py-2.5 text-center outline-none border border-border rounded-lg bg-background w-full placeholder:text-muted focus:border-primary focus:ring-2 focus:ring-primary/20"
					/>
					<button
						type="submit"
						class="text-sm text-primary-foreground font-medium px-4 py-2.5 rounded-lg bg-primary w-full transition-colors hover:bg-primary/90"
					>
						Sign in
					</button>
					<a href={resolve('/login')} class="text-sm text-muted text-center hover:text-foreground"
						>Use a different email</a
					>
				</form>
			{:else}
				<form method="POST" action="?/email-otp" class="flex flex-col gap-3 w-full">
					<label for="email-otp-email" class="sr-only">Email address</label>
					<input
						id="email-otp-email"
						name="email"
						type="email"
						autocomplete="email"
						placeholder="you@example.com"
						required
						class="text-sm text-foreground px-3 py-2.5 outline-none border border-border rounded-lg bg-background w-full placeholder:text-muted focus:border-primary focus:ring-2 focus:ring-primary/20"
					/>
					<button
						type="submit"
						class="text-sm text-primary-foreground font-medium px-4 py-2.5 rounded-lg bg-primary w-full transition-colors hover:bg-primary/90"
					>
						Continue with email
					</button>
				</form>
			{/if}

			<form bind:this={formEl} method="POST" action="?/google" class="hidden">
				<input bind:this={credentialInput} type="hidden" name="credential" />
			</form>
		</div>
	</div>
</div>
