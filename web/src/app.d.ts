// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		interface Locals {
			user: {
				id: string;
				email: string;
				username?: string;
				display_name: string;
				photo_url?: string;
				status: string;
				sports_profile?: string;
				contact_info?: string;
				created_at: string;
				updated_at: string;
			} | null;
		}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}
}

export {};
