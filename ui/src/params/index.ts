import { defineParams } from '@sveltejs/kit/params';
import { match } from './tracker.ts';

export const params = defineParams({
	tracker: (param) => (match(param) ? param : undefined)
});
