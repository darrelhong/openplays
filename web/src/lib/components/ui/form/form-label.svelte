<script lang="ts">
	import type { HTMLLabelAttributes } from 'svelte/elements';

	type Props = HTMLLabelAttributes & {
		required?: boolean;
	};

	let { required = false, children, class: className, ...rest }: Props = $props();
</script>

<label class="text-sm text-muted mb-1 block {className ?? ''}" {...rest}>
	{@render children?.()}{#if required}<span class="text-destructive/70 ml-0.5">*</span>{/if}
</label>
