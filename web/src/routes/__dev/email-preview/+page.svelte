<script lang="ts">
	let { data } = $props();
</script>

<svelte:head>
	<title>Email preview - OpenPlays</title>
</svelte:head>

<section class="mx-auto px-4 py-10 flex flex-col max-w-2xl min-h-[70vh] w-full justify-center">
	<div class="space-y-6">
		<div class="space-y-2">
			<p class="text-sm text-emerald-700 tracking-wide font-medium uppercase">Local development</p>
			<h1 class="text-3xl text-zinc-950 font-semibold">Email preview</h1>
			<p class="text-sm text-zinc-600">
				This is the exact HTML sent for email sign-in codes. Check Mailpit for the final delivered
				rendering.
			</p>
		</div>

		<iframe
			title="Email OTP preview"
			sandbox=""
			srcdoc={data.html}
			class="p-6 border border-zinc-200 rounded-lg bg-white min-h-72 w-full shadow-sm"
		></iframe>
	</div>
</section>
