import type { Component } from 'svelte';
import BellIcon from '@lucide/svelte/icons/bell';
import ChartLineIcon from '@lucide/svelte/icons/chart-line';
import EllipsisIcon from '@lucide/svelte/icons/ellipsis';
import FileDownIcon from '@lucide/svelte/icons/file-down';
import FlagIcon from '@lucide/svelte/icons/flag';
import HouseIcon from '@lucide/svelte/icons/house';
import LayoutGridIcon from '@lucide/svelte/icons/layout-grid';
import ListIcon from '@lucide/svelte/icons/list';
import LogOutIcon from '@lucide/svelte/icons/log-out';
import NotebookPenIcon from '@lucide/svelte/icons/notebook-pen';
import SunMoonIcon from '@lucide/svelte/icons/sun-moon';
import SyringeIcon from '@lucide/svelte/icons/syringe';
import UserIcon from '@lucide/svelte/icons/user';
import BabyIcon from '@lucide/svelte/icons/baby';
import { THEMES, type Theme } from './constants';
import type { ModuleId, ModuleState, Role } from './types';

export type NavIcon = Component;
type Base = { label: string; icon: NavIcon; current: boolean };
export type NavLink = Base & { kind: 'link'; id: string; href: string; note: string };
export type NavAction = Base & { kind: 'theme' | 'logout'; id: string; note: string };
export type NavEntry = NavLink | NavAction;
export type PrimaryEntry = Base & { id: 'home' | 'timeline' | 'growth'; href: string };
export type NavInput = {
	pathname: string;
	kidId: string | null;
	role: Role | null;
	// null when the modules could not be read: nothing is hidden.
	modules: ModuleState[] | null;
	theme: Theme;
};

export const MORE_ICON = EllipsisIcon;
export const BELL_ICON = BellIcon;

const THEME_LABEL: Record<Theme, string> = { light: 'Light', dark: 'Dark', auto: 'Auto' };
export const themeLabel = (t: Theme) => THEME_LABEL[t];
export { THEMES };

const KID = String.raw`^/kids/(?!new(?:/|$))[^/]+`;
const section = (name: string) => new RegExp(`${KID}/${name}(/|$)`);
const HOME = new RegExp(`^/$|${KID}(/log/[^/]+)?$`);
const TIMELINE = new RegExp(`^/timeline$|${KID}/timeline(/|$)|${KID}/log/[^/]+/[^/]+$`);
const PROFILE = new RegExp(`${KID}/(profile|access|settings)(/|$)`);

// Each entry names the module that must be on for the child, if any.
const MODULE_OF: Partial<Record<string, ModuleId>> = {
	vaccines: 'vaccination',
	medication: 'medication',
	milestones: 'milestones'
};

export function navModel(i: NavInput): { primary: PrimaryEntry[]; more: NavEntry[] } {
	const at = (re: RegExp) => re.test(i.pathname);
	const primary: PrimaryEntry[] = [
		{ id: 'home', label: 'Home', href: '/', icon: HouseIcon, current: at(HOME) },
		{ id: 'timeline', label: 'Timeline', href: '/timeline', icon: ListIcon, current: at(TIMELINE) },
		{
			id: 'growth',
			label: 'Growth',
			href: '/growth',
			icon: ChartLineIcon,
			current: at(section('growth'))
		}
	];
	const off = new Set((i.modules ?? []).filter((m) => !m.enabled).map((m) => m.module));
	const guardian = i.role === 'guardian';
	const kid = i.kidId ? `/kids/${i.kidId}` : null;
	const link = (
		id: string,
		label: string,
		icon: NavIcon,
		path: string | null,
		note: string,
		current: RegExp
	): NavLink[] =>
		path === null || off.has(MODULE_OF[id] as ModuleId)
			? []
			: [{ kind: 'link', id, label, icon, href: path, note, current: at(current) }];
	const more: NavEntry[] = [
		...link(
			'vaccines',
			'Vaccines',
			SyringeIcon,
			kid && `${kid}/vaccines`,
			'Open',
			section('vaccines')
		),
		...link(
			'medication',
			'Medication',
			NotebookPenIcon,
			kid && `${kid}/medication`,
			'Open',
			section('medication')
		),
		...link(
			'milestones',
			'Milestones',
			FlagIcon,
			kid && `${kid}/milestones`,
			'Open',
			section('milestones')
		),
		...link(
			'export',
			'Export PDF',
			FileDownIcon,
			kid && `${kid}/export`,
			'Open',
			section('export')
		),
		...(guardian
			? [
					...link(
						'modules',
						'Modules',
						LayoutGridIcon,
						kid && `${kid}/modules`,
						'Guardian',
						section('modules')
					),
					...link(
						'profile',
						'Child profile',
						BabyIcon,
						kid && `${kid}/profile`,
						'Guardian',
						PROFILE
					)
				]
			: []),
		...link('account', 'My profile', UserIcon, '/account', 'Open', /^\/account$/),
		{
			kind: 'theme',
			id: 'theme',
			label: 'Change theme',
			icon: SunMoonIcon,
			note: themeLabel(i.theme),
			current: false
		},
		{
			kind: 'logout',
			id: 'logout',
			label: 'Sign out',
			icon: LogOutIcon,
			note: 'This device',
			current: false
		}
	];
	return { primary, more };
}
