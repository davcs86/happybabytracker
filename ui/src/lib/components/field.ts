// Native <select> styled like the Input; the chevron is two currentColor gradients, so it works without JavaScript.
export const SELECT_CLASS =
	'min-h-12 w-full min-w-0 appearance-none rounded-sm border border-input bg-background bg-[image:linear-gradient(to_bottom_left,transparent_calc(50%_-_1px),currentColor_calc(50%_-_1px),currentColor_calc(50%_+_1px),transparent_calc(50%_+_1px)),linear-gradient(to_bottom_right,transparent_calc(50%_-_1px),currentColor_calc(50%_-_1px),currentColor_calc(50%_+_1px),transparent_calc(50%_+_1px))] bg-[position:calc(100%_-_18px)_center,calc(100%_-_12px)_center] bg-size-[6px_6px] bg-no-repeat py-2 pr-9 pl-3 text-base leading-[1.4] text-foreground shadow-xs outline-none disabled:pointer-events-none disabled:opacity-50';

export const ERROR_CLASS = 'text-[12.5px] leading-snug font-semibold text-destructive';

// Keeps the fields after an enhanced submit: the reloaded page data is what they show.
export const keepValues = () => {
	return async ({ update }: { update: (o?: { reset?: boolean }) => Promise<void> }) =>
		update({ reset: false });
};

export const END_RUNNING_HINT = 'Set the end to end the timer';

// Native radio and checkbox: 20 px, drawn by the browser in the primary colour.
export const CHOICE_CLASS = 'size-5 shrink-0 accent-primary';
