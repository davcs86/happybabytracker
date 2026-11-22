import type {
	AgeBand,
	BottleContent,
	DiaperKind,
	GrowthScoreReason,
	GrowthSourceId,
	LengthUnit,
	Measure,
	ModuleId,
	NursingSide,
	PumpingSide,
	TimelineType,
	TimerKind,
	TrackerKind,
	VolumeUnit,
	WeightUnit
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
// Longest sleep, nursing or pumping session the API accepts (MaxSessionDuration).
export const MAX_SESSION_HOURS = 24;
// The API's volume range per tracker (Min/MaxBottleVolumeML, Min/MaxPumpingVolumeML) and MaxMealTextLen.
export const VOLUME_ML_RANGE = {
	bottles: { min: 1, max: 1000 },
	pumping: { min: 1, max: 2000 }
} as const;
export const MEAL_TEXT_MAX_LEN = 200;
// Growth (constants.go and growth.Sources): grams for weight, millimetres for length and head.
export const MEASURES = ['weight', 'length', 'head', 'bmi'] as const satisfies readonly Measure[];
export const GROWTH_SOURCES = ['who', 'cdc'] as const satisfies readonly GrowthSourceId[];
export const GROWTH_RANGES = {
	weight: { min: 200, max: 250000 },
	length: { min: 150, max: 2500 },
	head: { min: 150, max: 800 }
} as const;
export const OZ_PER_LB = 16;
export const GRAMS_PER_KG = 1000;
export const MM_PER_CM = 10;
export const HUNDREDTHS_PER_OZ = 100;
export const HUNDREDTHS_PER_INCH = 100;
// The API's DaysPerMonth, for ages in days shown in months.
export const DAYS_PER_MONTH = 30.4375;
// Percentiles beyond these are shown as "<0.1" and ">99.9".
export const PERCENTILE_SHOWN = { min: 0.1, max: 99.9 } as const;
export const PERCENTILE_DECIMALS = 1;
export const Z_SCORE_DECIMALS = 2;
// Decimals of a chart value (curve or kid's point) in the chart's data table.
export const CHART_VALUE_DECIMALS = 1;
export const WEIGHT_UNITS = ['kg', 'lb'] as const satisfies readonly WeightUnit[];
export const LENGTH_UNITS = ['cm', 'in'] as const satisfies readonly LengthUnit[];
export const GROWTH_SCORE_REASONS = [
	'not_covered',
	'before_term'
] as const satisfies readonly GrowthScoreReason[];
export const DEFAULT_TIME_ZONE = 'UTC';
// The API's default source for a new kid.
export const DEFAULT_GROWTH_SOURCE: GrowthSourceId = 'who';
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
export const TIMER_KINDS = ['sleep', 'nursing'] as const satisfies readonly TimerKind[];
// The one place the timeline's singular type names meet the route names that paths and TrackerKind use.
export const TIMELINE_TYPE: Record<TrackerKind, TimelineType> = {
	bottles: 'bottle',
	nursing: 'nursing',
	pumping: 'pumping',
	sleep: 'sleep',
	diapers: 'diaper',
	meals: 'meal'
};
export const TRACKER_TITLE: Record<TrackerKind, { log: string; edit: string }> = {
	bottles: { log: 'Log a bottle', edit: 'Edit bottle' },
	nursing: { log: 'Log nursing', edit: 'Edit nursing' },
	pumping: { log: 'Log pumping', edit: 'Edit pumping' },
	sleep: { log: 'Log sleep', edit: 'Edit sleep' },
	diapers: { log: 'Log a diaper', edit: 'Edit diaper' },
	meals: { log: 'Log a meal', edit: 'Edit meal' }
};
export const TRACKER_LABEL: Record<TrackerKind, string> = {
	bottles: 'Bottle',
	nursing: 'Nursing',
	pumping: 'Pumping',
	sleep: 'Sleep',
	diapers: 'Diaper',
	meals: 'Solids'
};
// Longest range the summary endpoint answers (SummaryMaxDays), so a longer timeline range is refused here.
export const SUMMARY_MAX_DAYS = 93;
export const VOLUME_UNITS = ['ml', 'floz'] as const satisfies readonly VolumeUnit[];
export const BOTTLE_CONTENTS = [
	'breast_milk',
	'formula',
	'cow_milk',
	'other'
] as const satisfies readonly BottleContent[];
export const BOTTLE_CONTENT_LABEL: Record<BottleContent, string> = {
	breast_milk: 'Breast milk',
	formula: 'Formula',
	cow_milk: "Cow's milk",
	other: 'Other'
};
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
export const MODULE_LABEL: Record<ModuleId, string> = {
	bottles: 'Bottle',
	nursing: 'Nursing',
	pumping: 'Pumping',
	diapers: 'Diaper',
	sleep: 'Sleep',
	feeding: 'Feeding',
	milestones: 'Milestones',
	growth: 'Growth',
	vaccination: 'Vaccination',
	sickness: 'Sickness',
	medication: 'Medication'
};
// Only modules whose tracker exists in this build have an entry point.
export const TRACKER_ROUTE: Partial<Record<ModuleId, TrackerKind>> = {
	bottles: 'bottles',
	nursing: 'nursing',
	pumping: 'pumping',
	diapers: 'diapers',
	sleep: 'sleep',
	feeding: 'meals'
};
// Spec 7.3: the quick-log buttons lead with these per age band; the other modules follow in MODULES order.
export const QUICK_LOG_FIRST: Record<AgeBand, readonly ModuleId[]> = {
	'0-11': ['bottles', 'nursing', 'diapers', 'sleep'],
	'12-23': ['feeding', 'bottles', 'sleep', 'diapers'],
	'24+': ['sickness', 'medication', 'growth', 'vaccination']
};
export const AGE_BANDS = ['0-11', '12-23', '24+'] as const satisfies readonly AgeBand[];
export const SECONDS_PER_MINUTE = 60;
export const SECONDS_PER_HOUR = 3600;
export const HUNDREDTHS_PER_FLOZ = 100;
