<script lang="ts">
	import { ScrollArea } from 'bits-ui';
</script>

<ScrollArea.Scrollbar
	orientation="vertical"
	class="p-px rounded-full bg-card flex w-2 select-none transition-[width,opacity] duration-200 touch-none hover:bg-accent data-[state=hidden]:opacity-0 data-[state=visible]:opacity-100 hover:w-2.5"
>
	<ScrollArea.Thumb class="rounded-full bg-muted-foreground w-full" />
</ScrollArea.Scrollbar>
