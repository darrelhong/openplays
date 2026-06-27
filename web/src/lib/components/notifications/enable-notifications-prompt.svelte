<script lang="ts">
	import { X } from '@lucide/svelte';
	import Button from '$lib/components/ui/button.svelte';
	import * as Popover from '$lib/components/ui/popover/index';
	import type { PermissionState } from './types';

	let {
		open,
		permission,
		enabling,
		onEnable,
		onDismiss
	}: {
		open: boolean;
		permission: PermissionState;
		enabling: boolean;
		onEnable: () => void | Promise<void>;
		onDismiss: () => void;
	} = $props();
</script>

<Popover.Root {open}>
	<Popover.Trigger
		class="opacity-0 size-px pointer-events-none right-4 top-14 fixed"
		aria-hidden="true"
		tabindex={-1}
	/>

	<Popover.Content
		class="p-3 w-96"
		side="bottom"
		align="end"
		interactOutsideBehavior="ignore"
		escapeKeydownBehavior="ignore"
	>
		<div class="flex gap-2 items-start">
			<div class="flex-1 min-w-0">
				<h2 class="text-sm font-medium">Enable push notifications</h2>
				{#if permission === 'denied'}
					<p class="text-xs text-muted leading-snug mt-0.5">
						Notifications are blocked in this browser. Enable them in browser settings to get game
						updates.
					</p>
				{:else}
					<p class="text-xs text-muted leading-snug mt-0.5">
						Get updates when players join, confirm, or leave your games.
					</p>
					<Button type="button" size="xs" class="mt-2" disabled={enabling} onclick={onEnable}>
						Enable
					</Button>
				{/if}
			</div>
			<Button
				type="button"
				size="xs"
				variant="ghost"
				class="p-0 shrink-0 w-6"
				aria-label="Close"
				onclick={onDismiss}
			>
				<X class="size-4" />
			</Button>
		</div>
	</Popover.Content>
</Popover.Root>
