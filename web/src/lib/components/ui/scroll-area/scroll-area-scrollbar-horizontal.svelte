<script lang="ts">
	import { ScrollArea } from 'bits-ui';
</script>

<ScrollArea.Scrollbar
	orientation="horizontal"
	class="p-px rounded-full bg-stone-800 flex h-2 select-none transition-[height,opacity] duration-200 touch-none hover:bg-stone-700 data-[state=hidden]:opacity-0 data-[state=visible]:opacity-100 hover:h-2.5"
>
	<ScrollArea.Thumb class="rounded-full bg-stone-500 h-full" />
</ScrollArea.Scrollbar>
