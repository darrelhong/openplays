import { defineConfig, presetWind4 } from 'unocss';

export default defineConfig({
	presets: [presetWind4()],
	theme: {
		colors: {
			background: 'var(--color-background)',
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
			destructive: 'var(--color-destructive)',
			success: 'var(--color-success)',
			ring: 'var(--color-ring)'
		}
	}
});
