import { Slider as SliderPrimitive } from 'bits-ui';
import Slider from './slider.svelte';
import TennisSlider from './tennis-slider.svelte';

const Root = SliderPrimitive.Root;
const Range = SliderPrimitive.Range;
const Thumb = SliderPrimitive.Thumb;
const Tick = SliderPrimitive.Tick;
const TickLabel = SliderPrimitive.TickLabel;
const ThumbLabel = SliderPrimitive.ThumbLabel;

export {
	// Batteries-included generic slider
	Slider,
	Slider as default,
	// Primitives for custom composition
	Root,
	Range,
	Thumb,
	Tick,
	TickLabel,
	ThumbLabel,
	// Sport-specific wrappers
	TennisSlider,
	// Aliases
	Root as SliderRoot,
	Range as SliderRange,
	Thumb as SliderThumb,
	Tick as SliderTick,
	TickLabel as SliderTickLabel,
	ThumbLabel as SliderThumbLabel
};
