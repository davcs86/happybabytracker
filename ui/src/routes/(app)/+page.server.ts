import { formatAge } from '#lib/format.js';
import { todayInOrUtc } from '#lib/time-zone.js';
import { KID_COOKIE } from '#lib/server/constants.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ parent, cookies }) => {
	const { kids } = await parent();
	const remembered = cookies.get(KID_COOKIE);
	const currentId = kids.some((k) => k.id === remembered) ? remembered : kids[0]?.id;
	return {
		currentId,
		cards: kids.map((k) => ({ ...k, age: formatAge(k.birthDate, todayInOrUtc(k.timeZone)) }))
	};
};
