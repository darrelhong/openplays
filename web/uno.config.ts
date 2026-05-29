import { defineConfig, presetWind4 } from 'unocss';

export default defineConfig({
	presets: [presetWind4()],
	theme: {
		colors: {
			background: 'var(--color-background)',
			surface: 'var(--color-surface)',
			foreground: 'var(--color-foreground)',
			muted: {
				DEFAULT: 'var(--color-muted)',
				foreground: 'var(--color-muted-foreground)'
			},
			card: {
				DEFAULT: 'var(--color-card)',
				foreground: 'var(--color-card-foreground)'
			},
			border: 'var(--color-border)',
			input: {
				DEFAULT: 'var(--color-input)',
				border: 'var(--color-input-border)'
			},
			primary: {
				DEFAULT: 'var(--color-primary)',
				foreground: 'var(--color-primary-foreground)'
			},
			secondary: {
				DEFAULT: 'var(--color-secondary)',
				foreground: 'var(--color-secondary-foreground)'
			},
			accent: {
				DEFAULT: 'var(--color-accent)',
				foreground: 'var(--color-accent-foreground)'
			},
			destructive: {
				DEFAULT: 'var(--color-destructive)',
				foreground: 'var(--color-destructive-foreground)'
			},
			success: 'var(--color-success)',
			ring: 'var(--color-ring)'
		},
		shadow: {
			sm: '0 1px 2px var(--shadow-color)',
			DEFAULT: '0 1px 3px var(--shadow-color), 0 1px 2px var(--shadow-color)',
			md: '0 4px 6px var(--shadow-color), 0 2px 4px var(--shadow-color)',
			lg: '0 10px 15px var(--shadow-color), 0 4px 6px var(--shadow-color)',
			xl: '0 20px 25px var(--shadow-color), 0 8px 10px var(--shadow-color)'
		}
	}
});
