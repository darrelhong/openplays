import { browser } from '$app/environment';

export type Theme = 'light' | 'dark' | 'system';

const STORAGE_KEY = 'theme';

function getStoredTheme(): Theme {
	if (!browser) return 'system';
	const stored = localStorage.getItem(STORAGE_KEY);
	if (stored === 'light' || stored === 'dark') return stored;
	return 'system';
}

function getEffectiveTheme(theme: Theme): 'light' | 'dark' {
	if (theme !== 'system') return theme;
	if (!browser) return 'dark';
	return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function applyTheme(theme: Theme) {
	if (!browser) return;
	const effective = getEffectiveTheme(theme);
	document.documentElement.classList.toggle('dark', effective === 'dark');
}

let current: Theme = $state(getStoredTheme());

export function getTheme(): Theme {
	return current;
}

export function setTheme(theme: Theme) {
	current = theme;
	if (browser) {
		if (theme === 'system') {
			localStorage.removeItem(STORAGE_KEY);
		} else {
			localStorage.setItem(STORAGE_KEY, theme);
		}
		applyTheme(theme);
	}
}

// Listen for OS theme changes when set to 'system'
if (browser) {
	window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
		if (current === 'system') {
			applyTheme('system');
		}
	});
}
