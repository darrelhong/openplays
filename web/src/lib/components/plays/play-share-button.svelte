<script lang="ts">
	import Share2 from '@lucide/svelte/icons/share-2';
	import Button from '$lib/components/ui/button.svelte';

	let { playId, title }: { playId: string; title: string } = $props();

	type Feedback = 'idle' | 'shared' | 'copied' | 'error';
	let feedback = $state<Feedback>('idle');
	const label = $derived(
		feedback === 'shared'
			? 'Shared'
			: feedback === 'copied'
				? 'Link copied'
				: feedback === 'error'
					? 'Unable to copy link'
					: 'Share'
	);

	function playURL() {
		return new URL(`/play/${playId}`, window.location.origin).toString();
	}

	async function copyURL(url: string) {
		if (!navigator.clipboard?.writeText) throw new Error('clipboard unavailable');
		await navigator.clipboard.writeText(url);
		feedback = 'copied';
	}

	async function sharePlay() {
		const url = playURL();
		feedback = 'idle';

		if (navigator.share) {
			try {
				await navigator.share({ title, url });
				feedback = 'shared';
				return;
			} catch (error) {
				if (error instanceof DOMException && error.name === 'AbortError') return;
			}
		}

		try {
			await copyURL(url);
		} catch {
			feedback = 'error';
		}
	}
</script>

<Button
	type="button"
	size="sm"
	variant="outline"
	title="Share this game"
	aria-label={label}
	onclick={sharePlay}
	class="gap-1.5"
>
	<Share2 class="size-4" aria-hidden="true" />
	{label}
</Button>
