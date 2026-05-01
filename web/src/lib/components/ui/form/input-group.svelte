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
	class="border border-stone-600 rounded-lg bg-stone-900 flex items-center focus-within:border-stone-400 {className ??
		''}"
	{...rest}
>
	<span class="text-stone-500 pl-3 select-none">{prefix}</span>
	{@render children()}
</div>
