<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	type Props = HTMLAttributes<HTMLDivElement> & {
		prefix: string;
		children: Snippet;
	};

	let { prefix, children, class: className, ...rest }: Props = $props();
</script>

<div
	class="border border-input-border rounded-lg bg-input flex items-center focus-within:border-ring {className ??
		''}"
	{...rest}
>
	<span class="text-muted-foreground pl-3 select-none">{prefix}</span>
	{@render children()}
</div>
