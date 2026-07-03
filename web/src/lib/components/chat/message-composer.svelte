<script lang="ts">
	import { enhance } from '$app/forms';
	import { Send } from '@lucide/svelte';
	import type { SubmitFunction } from '@sveltejs/kit';
	import Button from '$lib/components/ui/button.svelte';
	import { addPendingMessage, removePendingMessage } from './pending-messages.svelte';
	import { applyResultInPlace } from './refresh-submit';

	let { error, conversationId }: { error?: string; conversationId: string } = $props();

	// Optimistic send: show the message and reset the composer immediately,
	// then reconcile once the refreshed data (or a failure) comes back
	const handleSubmit: SubmitFunction = ({ formElement, formData, cancel }) => {
		const body = String(formData.get('body') ?? '').trim();
		if (!body) {
			cancel();
			return;
		}
		const localId = addPendingMessage(conversationId, body);
		formElement.reset();

		return async ({ result, update }) => {
			await applyResultInPlace(result, update);
			removePendingMessage(localId);
			if (result.type === 'failure' || result.type === 'error') {
				// Restore the unsent text so it can be retried
				const textarea = formElement.elements.namedItem('body');
				if (textarea instanceof HTMLTextAreaElement && !textarea.value) {
					textarea.value = body;
				}
			}
		};
	};

	function handleKeydown(event: KeyboardEvent & { currentTarget: HTMLTextAreaElement }) {
		if (event.key === 'Enter' && !event.shiftKey && !event.isComposing) {
			event.preventDefault();
			if (event.currentTarget.value.trim()) {
				event.currentTarget.form?.requestSubmit();
			}
		}
	}
</script>

<form method="POST" action="?/send" use:enhance={handleSubmit} class="mx-auto max-w-3xl w-full">
	{#if error}
		<p class="text-sm text-destructive mb-2">{error}</p>
	{/if}
	<div class="flex gap-2 items-end">
		<div class="px-3 py-2 border border-border rounded-xl bg-card flex flex-1 shadow-sm items-end">
			<textarea
				name="body"
				rows="2"
				maxlength="2000"
				onkeydown={handleKeydown}
				class="text-sm text-foreground bg-transparent max-h-36 min-h-10 w-full resize-y placeholder:text-muted-foreground focus:outline-none"
				placeholder="Message"
			></textarea>
		</div>
		<Button
			type="submit"
			class="p-0 rounded-full shrink-0 h-12 w-12 shadow-sm"
			aria-label="Send message"
		>
			<Send class="size-5" />
		</Button>
	</div>
</form>
