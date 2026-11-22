import { TRACKER_KINDS } from '../lib/constants.ts';

export const match = (param: string): boolean =>
	(TRACKER_KINDS as readonly string[]).includes(param);
