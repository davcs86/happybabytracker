import type {
	AgeBand,
	BottleContent,
	DiaperKind,
	ModuleId,
	NursingSide,
	PumpingSide,
	TimelineType,
	TrackerKind,
	VolumeUnit
} from './types';

// Client-safe: the browser checks these too, so they cannot live under lib/server.
export const MIN_PASSWORD_LEN = 12;
export const MAX_PASSWORD_LEN = 256;
// The API's limits (api/internal/constants), for the forms' hints and attributes.
export const USERNAME_MIN_LEN = 3;
export const USERNAME_MAX_LEN = 32;
export const DISPLAY_NAME_MAX_LEN = 64;
export const KID_NAME_MAX_LEN = 64;
export const KID_MIN_BIRTH_DATE = '1900-01-01';
export const GESTATION_WEEKS = { min: 22, max: 44 } as const;
export const GESTATION_DAYS = { min: 0, max: 6 } as const;
export const DEFAULT_TIME_ZONE = 'UTC';
export const INVITE_VALIDITY = '72 hours';
export const RESET_VALIDITY = '1 hour';

export const TRACKER_KINDS: readonly TrackerKind[] = [
	'bottles',
	'nursing',
	'pumping',
	'sleep',
	'diapers',
	'meals'
];
// The one place the timeline's singular type names meet the route names that paths and TrackerKind use.
export const TIMELINE_TYPE: Record<TrackerKind, TimelineType> = {
	bottles: 'bottle',
	nursing: 'nursing',
	pumping: 'pumping',
	sleep: 'sleep',
	diapers: 'diaper',
	meals: 'meal'
};
export const VOLUME_UNITS = ['ml', 'floz'] as const satisfies readonly VolumeUnit[];
export const BOTTLE_CONTENTS = [
	'breast_milk',
	'formula',
	'cow_milk',
	'other'
] as const satisfies readonly BottleContent[];
export const DIAPER_KINDS = [
	'wet',
	'dirty',
	'both',
	'dry'
] as const satisfies readonly DiaperKind[];
export const NURSING_SIDES = ['left', 'right'] as const satisfies readonly NursingSide[];
export const PUMPING_SIDES = ['left', 'right', 'both'] as const satisfies readonly PumpingSide[];
export const MODULES = [
	'bottles',
	'nursing',
	'pumping',
	'diapers',
	'sleep',
	'feeding',
	'milestones',
	'growth',
	'vaccination',
	'sickness',
	'medication'
] as const satisfies readonly ModuleId[];
export const AGE_BANDS = ['0-11', '12-23', '24+'] as const satisfies readonly AgeBand[];
export const SECONDS_PER_MINUTE = 60;
export const HUNDREDTHS_PER_FLOZ = 100;
