<script lang="ts">
	import { Checkbox } from '$lib/components/ui/checkbox/index';

	let {
		requireWaitlist = $bindable(false),
		unlisted = $bindable(false)
	}: {
		requireWaitlist?: boolean;
		unlisted?: boolean;
	} = $props();
</script>

<!-- Shared play settings for the create and edit forms. Must render inside the
     <form>: the hidden inputs bridge the checkbox state into the POST body. -->
<input type="hidden" name="visibility" value={unlisted ? 'unlisted' : 'public'} />
<input type="hidden" name="require_waitlist" value={requireWaitlist ? 'true' : 'false'} />

<Checkbox
	bind:checked={requireWaitlist}
	class="items-start"
	rootClass="mt-0.5"
	labelClass="grid gap-0.5"
>
	<span>Require waitlist</span>
	<span class="text-muted">
		Players must go through waitlist regardless of skill level. Host can then approve players from
		the waitlist.
	</span>
</Checkbox>

<Checkbox bind:checked={unlisted} class="items-start" rootClass="mt-0.5" labelClass="grid gap-0.5">
	<span>Set visibility as unlisted</span>
	<span class="text-muted">
		Will not appear in public searches but anyone with the link can view
	</span>
</Checkbox>
