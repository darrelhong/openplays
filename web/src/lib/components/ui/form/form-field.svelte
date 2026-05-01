<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	type Props = HTMLAttributes<HTMLDivElement> & {
		label: string;
		id: string;
		children: Snippet;
	};

	let { label, id, children, class: className, ...rest }: Props = $props();
</script>

<div class={className} {...rest}>
	<label for={id} class="text-sm text-muted mb-1 block">{label}</label>
	{@render children()}
</div>
