import { invalidateAll } from '$app/navigation';
import { page } from '$app/state';
import type { ActionResult, SubmitFunction } from '@sveltejs/kit';

// Chat actions redirect back to the conversation they were submitted from.
// Letting enhance apply that as a goto() scrolls the window and drops focus,
// so refresh data in place when the redirect targets the current page.
export async function applyResultInPlace(
	result: ActionResult,
	update: (options?: { reset?: boolean; invalidateAll?: boolean }) => Promise<void>
) {
	const samePageRedirect =
		result.type === 'redirect' &&
		new URL(result.location, page.url).pathname === page.url.pathname;

	if (samePageRedirect) {
		await invalidateAll();
	} else {
		await update();
	}
}

export const refreshSubmit: SubmitFunction = ({ formElement }) => {
	return async ({ result, update }) => {
		await applyResultInPlace(result, update);
		if (result.type === 'success' || result.type === 'redirect') {
			formElement.reset();
		}
	};
};
