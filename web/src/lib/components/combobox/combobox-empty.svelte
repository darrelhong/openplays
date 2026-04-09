<script lang="ts">
	import { cn } from '$lib/utils/cn';
	import type { HTMLAttributes } from 'svelte/elements';

	let { class: className, ...restProps }: HTMLAttributes<HTMLSpanElement> = $props();
</script>

<span
	data-slot="combobox-empty"
	class={cn('text-sm text-stone-500 px-2 py-1.5 block', className)}
	{...restProps}
>
	No results found.
</span>
