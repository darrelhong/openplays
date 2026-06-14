<script lang="ts">
	import { resolve } from '$app/paths';
	import { goto } from '$app/navigation';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index';
	import UserAvatar from '$lib/components/ui/avatar/user-avatar.svelte';

	type UserMenuUser = {
		photo_url?: string;
		username?: string;
		display_name: string;
	};

	let { user }: { user: UserMenuUser } = $props();

	let logoutFormEl = $state<HTMLFormElement | null>(null);

	function submitLogout() {
		logoutFormEl?.requestSubmit();
	}

	function goToProfile() {
		goto(resolve('/profile'));
	}

	function goToMyGames() {
		goto(resolve('/my-games'));
	}

	function goToFavourites() {
		goto(resolve('/favourites'));
	}

	function goToCreate() {
		goto(resolve('/create'));
	}
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<button
				{...props}
				type="button"
				class="text-foreground p-1 rounded-md flex gap-2 items-center hover:text-foreground/80"
				aria-label="User menu"
			>
				<UserAvatar src={user.photo_url} nameForFallback={user.username ?? user.display_name} />
				<span class="text-sm">{user.username ?? user.display_name}</span>
			</button>
		{/snippet}
	</DropdownMenu.Trigger>

	<DropdownMenu.Portal>
		<DropdownMenu.Content sideOffset={8}>
			<DropdownMenu.Item onSelect={goToMyGames}>My Games</DropdownMenu.Item>

			<DropdownMenu.Item onSelect={goToFavourites}>Favourites</DropdownMenu.Item>

			<DropdownMenu.Item onSelect={goToProfile}>Profile</DropdownMenu.Item>

			<DropdownMenu.Item onSelect={goToCreate}>Create Game</DropdownMenu.Item>

			<DropdownMenu.Separator />

			<DropdownMenu.Item onSelect={submitLogout}>Logout</DropdownMenu.Item>
		</DropdownMenu.Content>
	</DropdownMenu.Portal>
</DropdownMenu.Root>

<form bind:this={logoutFormEl} method="POST" action="/logout" class="hidden"></form>
