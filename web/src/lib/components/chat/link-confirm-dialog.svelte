<script lang="ts">
	import Button from '$lib/components/ui/button.svelte';
	import * as Dialog from '$lib/components/ui/dialog/index';
	import { linkConfirm } from './link-confirm.svelte';

	const open = $derived(linkConfirm.url != null);

	function onOpenChange(value: boolean) {
		if (!value) {
			linkConfirm.url = null;
		}
	}

	function openLink() {
		if (linkConfirm.url) {
			window.open(linkConfirm.url, '_blank', 'noopener,noreferrer');
		}
		linkConfirm.url = null;
	}
</script>

<Dialog.Root {open} {onOpenChange}>
	<Dialog.Content
		showCloseButton={false}
		class="p-5 border border-border gap-4 shadow-card/30 shadow-lg sm:max-w-md"
	>
		<Dialog.Header>
			<Dialog.Title>Open external link?</Dialog.Title>
			<Dialog.Description>
				This link was shared in chat. Only open links from people you trust.
			</Dialog.Description>
		</Dialog.Header>
		<p class="text-sm text-foreground px-3 py-2 rounded-md bg-secondary break-all">
			{linkConfirm.url}
		</p>
		<div class="mt-1 flex flex-col gap-2 sm:flex-row sm:justify-end">
			<Dialog.Close>
				{#snippet child({ props })}
					<Button type="button" variant="outline" size="sm" class="w-full sm:w-auto" {...props}>
						Cancel
					</Button>
				{/snippet}
			</Dialog.Close>
			<Button type="button" size="sm" class="w-full sm:w-auto" onclick={openLink}>Open link</Button>
		</div>
	</Dialog.Content>
</Dialog.Root>
