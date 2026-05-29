<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '$lib/utils/cn';
	import Button from '$lib/components/ui/button.svelte';
	import * as Dialog from '$lib/components/ui/dialog/index';

	type ButtonVariant = 'default' | 'secondary' | 'outline' | 'ghost' | 'destructive';
	type ButtonSize = 'xs' | 'sm' | 'md' | 'lg';
	type TriggerContext = { props: Record<string, unknown> };

	let {
		title,
		description,
		action,
		method = 'POST',
		confirmLabel = 'Continue',
		cancelLabel = 'Cancel',
		confirmVariant = 'default',
		cancelVariant = 'outline',
		size = 'sm',
		contentClass,
		trigger,
		fields
	}: {
		title: string;
		description?: string;
		action: string;
		method?: 'GET' | 'POST';
		confirmLabel?: string;
		cancelLabel?: string;
		confirmVariant?: ButtonVariant;
		cancelVariant?: ButtonVariant;
		size?: ButtonSize;
		contentClass?: string;
		trigger: Snippet<[TriggerContext]>;
		fields?: Snippet;
	} = $props();
</script>

<Dialog.Root>
	<Dialog.Trigger>
		{#snippet child({ props })}
			{@render trigger({ props })}
		{/snippet}
	</Dialog.Trigger>
	<Dialog.Content
		showCloseButton={false}
		class={cn('p-5 border border-border gap-4 shadow-card/30 shadow-lg sm:max-w-xs', contentClass)}
	>
		<Dialog.Header>
			<Dialog.Title>{title}</Dialog.Title>
			{#if description}
				<Dialog.Description>{description}</Dialog.Description>
			{/if}
		</Dialog.Header>
		<div class="mt-1 flex flex-col gap-2 sm:flex-row sm:justify-end">
			<Dialog.Close>
				{#snippet child({ props })}
					<Button type="button" variant={cancelVariant} {size} class="w-full sm:w-auto" {...props}>
						{cancelLabel}
					</Button>
				{/snippet}
			</Dialog.Close>
			<form {method} {action} class="w-full sm:w-auto">
				{#if fields}
					{@render fields()}
				{/if}
				<Button type="submit" variant={confirmVariant} {size} class="w-full sm:w-auto">
					{confirmLabel}
				</Button>
			</form>
		</div>
	</Dialog.Content>
</Dialog.Root>
