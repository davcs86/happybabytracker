import type { Api } from '#lib/server/api.js';
import type { DrawerState } from '#lib/drawer.js';
import type { Session } from '#lib/types.js';

declare global {
	namespace App {
		interface Locals {
			session: Session | null;
			api: Api;
		}
		interface PageState {
			drawer?: DrawerState;
		}
	}
}

export {};
