// Native <select> styled like the shadcn Input, so a select works without JavaScript.
export const SELECT_CLASS =
	'h-8 w-full min-w-0 rounded-lg border border-input bg-background px-2 text-base outline-none focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 aria-invalid:border-destructive aria-invalid:ring-3 aria-invalid:ring-destructive/20 md:text-sm';

export const ERROR_CLASS = 'text-sm text-destructive';

// Keeps the fields after an enhanced submit: the reloaded page data is what they show.
export const keepValues = () => {
	return async ({ update }: { update: (o?: { reset?: boolean }) => Promise<void> }) =>
		update({ reset: false });
};

export const END_RUNNING_HINT = 'Set the end to end the timer';
