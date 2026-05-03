<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	type Props = HTMLAttributes<HTMLDivElement> & {
		label: string;
		id: string;
		required?: boolean;
		children: Snippet;
	};

	let { label, id, required = false, children, class: className, ...rest }: Props = $props();
</script>

<div class={className} {...rest}>
	<label for={id} class="text-sm text-muted mb-1 block">
		{label}{#if required}<span class="text-destructive/70 ml-0.5">*</span>{/if}
	</label>
	{@render children()}
</div>
