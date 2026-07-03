import { invalidateAll } from '$app/navigation';
import { page } from '$app/state';
import type { SubmitFunction } from '@sveltejs/kit';

// Chat actions redirect back to the conversation they were submitted from.
// Letting enhance apply that as a goto() scrolls the window and drops focus,
// so refresh data in place when the redirect targets the current page.
export const refreshSubmit: SubmitFunction = ({ formElement }) => {
	return async ({ result, update }) => {
		const samePageRedirect =
			result.type === 'redirect' &&
			new URL(result.location, page.url).pathname === page.url.pathname;

		if (samePageRedirect) {
			await invalidateAll();
			formElement.reset();
		} else {
			await update();
			if (result.type === 'success' || result.type === 'redirect') {
				formElement.reset();
			}
		}
	};
};
