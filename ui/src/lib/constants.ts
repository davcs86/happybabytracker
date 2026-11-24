import type {
	AgeBand,
	BottleContent,
	DiaperKind,
	GrowthScoreReason,
	GrowthSourceId,
	LengthUnit,
	Measure,
	MilestoneDomain,
	MilestonePosition,
	ModuleId,
	NursingSide,
	PumpingSide,
	Severity,
	SymptomName,
	TempMethod,
	TimelineType,
	TimerKind,
	TrackerKind,
	Units,
	VaccineAgeUnit,
	VaccineDismissReason,
	VaccineKind,
	VaccineStatus,
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
// Sickness, medication and reminders (constants.go): the names are the Go constants'.
// MedicationTextMaxLen: a course's name and dose, a dose's amount.
export const MEDICATION_TEXT_MAX_LEN = 80;
// SymptomOtherMaxLen.
export const SYMPTOM_OTHER_MAX_LEN = 80;
// MinTempCHundredths and MaxTempCHundredths: what the API accepts, whichever unit was typed.
export const MIN_TEMP_C_HUNDREDTHS = 3000;
export const MAX_TEMP_C_HUNDREDTHS = 4500;
// PushLabelMaxLen.
export const PUSH_LABEL_MAX_LEN = 40;
// The page's sentence and the subscribe route's answer while the server has no VAPID key.
export const PUSH_OFF_MESSAGE = 'Web Push is not configured on this server.';
// MaxScheduleTimes.
export const MAX_SCHEDULE_TIMES = 12;
// MinEveryHours and MaxEveryHours.
export const MIN_EVERY_HOURS = 1;
export const MAX_EVERY_HOURS = 168;
// MinReminderOverdueMinutes and MaxReminderOverdueMinutes.
export const MIN_REMINDER_OVERDUE_MINUTES = 5;
export const MAX_REMINDER_OVERDUE_MINUTES = 1440;
// Growth (constants.go and growth.Sources): grams for weight, millimetres for length and head.
export const MEASURES = ['weight', 'length', 'head', 'bmi'] as const satisfies readonly Measure[];
export const GROWTH_SOURCES = ['who', 'cdc'] as const satisfies readonly GrowthSourceId[];
export const GROWTH_RANGES = {
	weight: { min: 200, max: 250000 },
	length: { min: 150, max: 2500 },
	head: { min: 150, max: 800 }
} as const;
export const OZ_PER_LB = 16;
// The two sentences the weigh-in form and the estimate carry; the density mirrors MilkDensityHundredths.
export const SCALE_CAVEAT = 'A single-feed reading from a home scale can be off by several grams.';
export const MILK_ESTIMATE_BASIS = `The volume assumes about 1.03 g per ml, an approximation in the range commonly used (1.0\u20131.03 g/ml); not measured. ${SCALE_CAVEAT}`;
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
export const TEMP_UNITS = ['c', 'f'] as const satisfies readonly Units['temp'][];
export const TEMP_UNIT_LABEL: Record<Units['temp'], string> = { c: '°C', f: '°F' };
export const GROWTH_SCORE_REASONS = [
	'not_covered',
	'before_term'
] as const satisfies readonly GrowthScoreReason[];
export const DEFAULT_TIME_ZONE = 'UTC';
// The API's default source for a new kid.
export const DEFAULT_GROWTH_SOURCE: GrowthSourceId = 'who';
// Vaccination (constants.go): lot, product name and note lengths in characters.
export const VACCINE_LOT_MAX_LEN = 40;
export const VACCINE_PRODUCT_MAX_LEN = 80;
export const NOTE_MAX_LEN = 500;
export const VACCINE_STATUSES = [
	'given',
	'due',
	'upcoming',
	'overdue',
	'dismissed',
	'available'
] as const satisfies readonly VaccineStatus[];
export const VACCINE_KINDS = [
	'routine',
	'may_not_be_needed',
	'supplementary',
	'recurring'
] as const satisfies readonly VaccineKind[];
export const VACCINE_AGE_UNITS = ['d', 'w', 'm', 'y'] as const satisfies readonly VaccineAgeUnit[];
export const VACCINE_DISMISS_REASONS = [
	'not_needed',
	'given_unknown'
] as const satisfies readonly VaccineDismissReason[];
export const VACCINE_STATUS_LABEL: Record<VaccineStatus, string> = {
	given: 'Given',
	due: 'Due',
	upcoming: 'Upcoming',
	overdue: 'Overdue',
	dismissed: 'Dismissed',
	available: 'Available'
};
export const VACCINE_UNIT_LABEL: Record<VaccineAgeUnit, { one: string }> = {
	d: { one: 'day' },
	w: { one: 'week' },
	m: { one: 'month' },
	y: { one: 'year' }
};
// Milestones (constants.go): MilestoneTextMaxLen, a custom milestone's text in characters after trimming.
export const MILESTONE_TEXT_MAX = 200;
// MilestoneCatalogAges: the checklist ages, in months.
export const MILESTONE_AGES = [2, 4, 6, 9, 12, 15, 18, 24, 30, 36, 48, 60] as const;
export const MILESTONE_DOMAINS = [
	'social_emotional',
	'language_communication',
	'cognitive',
	'movement_physical'
] as const satisfies readonly MilestoneDomain[];
export const MILESTONE_POSITIONS = [
	'earlier',
	'current',
	'next',
	'later'
] as const satisfies readonly MilestonePosition[];
// The home shows at most this many due entries.
export const HOME_DUE_MAX = 3;
export const INVITE_VALIDITY = '72 hours';
export const RESET_VALIDITY = '1 hour';

export const TRACKER_KINDS: readonly TrackerKind[] = [
	'bottles',
	'nursing',
	'pumping',
	'sleep',
	'diapers',
	'meals',
	'symptoms'
];
export const TIMER_KINDS = ['sleep', 'nursing'] as const satisfies readonly TimerKind[];
// The one place the timeline's singular type names meet the route names that paths and TrackerKind use.
export const TIMELINE_TYPE: Record<TrackerKind, TimelineType> = {
	bottles: 'bottle',
	nursing: 'nursing',
	pumping: 'pumping',
	sleep: 'sleep',
	diapers: 'diaper',
	meals: 'meal',
	symptoms: 'symptom'
};
export const TRACKER_TITLE: Record<TrackerKind, { log: string; edit: string }> = {
	bottles: { log: 'Log a bottle', edit: 'Edit bottle' },
	nursing: { log: 'Log nursing', edit: 'Edit nursing' },
	pumping: { log: 'Log pumping', edit: 'Edit pumping' },
	sleep: { log: 'Log sleep', edit: 'Edit sleep' },
	diapers: { log: 'Log a diaper', edit: 'Edit diaper' },
	meals: { log: 'Log a meal', edit: 'Edit meal' },
	symptoms: { log: 'Log a symptom', edit: 'Edit symptom' }
};
export const TRACKER_LABEL: Record<TrackerKind, string> = {
	bottles: 'Bottle',
	nursing: 'Nursing',
	pumping: 'Pumping',
	sleep: 'Sleep',
	diapers: 'Diaper',
	meals: 'Solids',
	symptoms: 'Symptom'
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
export const SYMPTOMS = [
	'congestion',
	'cough',
	'fever',
	'vomiting',
	'diarrhea',
	'constipation',
	'rash',
	'ear_pain',
	'sore_throat',
	'poor_appetite',
	'unusually_sleepy',
	'other'
] as const satisfies readonly SymptomName[];
export const SYMPTOM_LABEL: Record<SymptomName, string> = {
	congestion: 'Congestion',
	cough: 'Cough',
	fever: 'Fever',
	vomiting: 'Vomiting',
	diarrhea: 'Diarrhea',
	constipation: 'Constipation',
	rash: 'Rash',
	ear_pain: 'Ear pain',
	sore_throat: 'Sore throat',
	poor_appetite: 'Poor appetite',
	unusually_sleepy: 'Unusually sleepy',
	other: 'Other'
};
export const SEVERITIES = ['mild', 'moderate', 'severe'] as const satisfies readonly Severity[];
export const TEMP_METHODS = [
	'oral',
	'rectal',
	'axillary',
	'ear',
	'forehead',
	'other'
] as const satisfies readonly TempMethod[];
export const TEMP_METHOD_LABEL: Record<TempMethod, string> = {
	oral: 'Oral',
	rectal: 'Rectal',
	axillary: 'Armpit',
	ear: 'Ear',
	forehead: 'Forehead',
	other: 'Other'
};
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
	feeding: 'meals',
	sickness: 'symptoms'
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
export const HUNDREDTHS_PER_DEGREE = 100;
