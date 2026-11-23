export type SessionKind = 'admin' | 'parent';

export type Units = {
	weight: 'kg' | 'lb';
	length: 'cm' | 'in';
	volume: 'ml' | 'floz';
	temp: 'c' | 'f';
};

export type Parent = { id: string; username: string; displayName: string; units: Units };

export type Session = { kind: SessionKind; parent?: Parent };

export type NewSession = { token: string; kind: SessionKind; expiresAt: string };

export type TokenInfo = { kind: 'invite' | 'reset'; username: string };

export type Role = 'guardian' | 'caregiver';

export type Sex = 'female' | 'male';

export type GrowthSourceId = 'who' | 'cdc';

// growthSource or vaccineSchema left out keeps the stored value (on create: the API's default, no schedule).
export type KidInput = {
	name: string;
	birthDate: string;
	sex: string;
	gestationWeeks: number | null;
	gestationDays: number | null;
	timeZone: string;
	growthSource?: string;
	vaccineSchema?: string | null;
};

export type Kid = Omit<KidInput, 'sex' | 'growthSource' | 'vaccineSchema'> & {
	id: string;
	sex: Sex;
	growthSource: GrowthSourceId;
	// An id the API no longer ships counts as no schedule (schemaChosen).
	vaccineSchema: string | null;
	role: Role;
};

export type Grant = { parentId: string; username: string; displayName: string; role: Role };

export type AdminParent = {
	id: string;
	username: string;
	displayName: string;
	disabled: boolean;
	hasPassword: boolean;
	createdAt: string;
};

export type ParentLink = { kind: 'invite' | 'reset'; url: string; expiresAt: string };

export type SessionInfo = { id: string; createdAt: string; lastSeenAt: string; current: boolean };

export type TrackerKind = 'bottles' | 'nursing' | 'pumping' | 'sleep' | 'diapers' | 'meals';

export type TimelineType = 'bottle' | 'nursing' | 'pumping' | 'sleep' | 'diaper' | 'meal';

export type VolumeUnit = Units['volume'];

// The API's own names: day totals share the shape, so formatVolume takes either.
export type Volume = { ml: number; floz_hundredths: number; typed_unit: VolumeUnit };

export type BottleContent = 'breast_milk' | 'formula' | 'cow_milk' | 'other';
export type DiaperKind = 'wet' | 'dirty' | 'both' | 'dry';
export type NursingSide = 'left' | 'right';
export type PumpingSide = NursingSide | 'both';

type Noted = { id: string; note: string | null };

export type Bottle = Noted & { at: string; contents: BottleContent[]; volume: Volume };
export type Diaper = Noted & { at: string; kind: DiaperKind };
export type Meal = Noted & { at: string; what: string; reaction: string | null };
export type Pumping = Noted & {
	startedAt: string;
	durationSeconds: number;
	volume: Volume;
	side: PumpingSide;
};
// A timer runs while endedAt is null.
export type Sleep = Noted & { startedAt: string; endedAt: string | null };
export type Nursing = Noted & {
	startedAt: string;
	endedAt: string | null;
	leftSeconds: number;
	rightSeconds: number;
	side: NursingSide | null;
	sideSince: string | null;
};

export type TrackerEntries = {
	bottles: Bottle;
	nursing: Nursing;
	pumping: Pumping;
	sleep: Sleep;
	diapers: Diaper;
	meals: Meal;
};
export type TimerKind = 'sleep' | 'nursing';

// The JSON body of a create or update, in the API's names.
export type EntryBody = Record<string, unknown>;

export type ListQuery = { from?: string; to?: string; limit?: number; cursor?: string };
export type Page<T> = { items: T[]; nextCursor: string | null };

export type TimelineQuery = ListQuery & { types?: TimelineType[] };
export type TimelineItem = {
	[K in TrackerKind]: { kind: K; at: string; entry: TrackerEntries[K] };
}[TrackerKind];

export type ModuleId =
	| 'bottles'
	| 'nursing'
	| 'pumping'
	| 'diapers'
	| 'sleep'
	| 'feeding'
	| 'milestones'
	| 'growth'
	| 'vaccination'
	| 'sickness'
	| 'medication';
export type ModuleState = {
	module: ModuleId;
	enabled: boolean;
	default: boolean;
	overridden: boolean;
};

export type AgeBand = '0-11' | '12-23' | '24+';

export type VolumeTotal = { count: number; ml: number; floz_hundredths: number };
export type DaySummary = {
	date: string;
	bottles: VolumeTotal;
	nursing: { count: number; seconds: number };
	pumping: VolumeTotal;
	diapers: { total: number; wet: number; dirty: number; both: number; dry: number };
	sleep: { seconds: number };
	meals: { count: number };
};

export type Home = {
	ageMonths: number;
	// The due date while corrected age applies, else null.
	correctedBirthDate: string | null;
	band: AgeBand;
	modules: ModuleState[];
	running: { sleep: Sleep | null; nursing: Nursing | null };
	today: DaySummary;
	due: HomeDue[];
};

export type Measure = 'weight' | 'length' | 'head' | 'bmi';
export type WeightUnit = Units['weight'];
export type LengthUnit = Units['length'];
export type GrowthScoreReason = 'not_covered' | 'before_term';
export type GrowthScore =
	{ scored: true; z: number; percentile: number } | { scored: false; reason: GrowthScoreReason };
// Both columns; typedUnit names the exact one.
export type GrowthWeight = { grams: number; ozHundredths: number; typedUnit: WeightUnit };
export type GrowthLength = { mm: number; inHundredths: number; typedUnit: LengthUnit };
export type GrowthRow = {
	id: string;
	measuredOn: string;
	weight: GrowthWeight | null;
	length: GrowthLength | null;
	head: GrowthLength | null;
	note: string | null;
	ageDays: number;
	// The scoring age while corrected age applies, negative before the expected term; else null.
	correctedDays: number | null;
	scores: {
		weight: GrowthScore | null;
		length: GrowthScore | null;
		head: GrowthScore | null;
		bmi: (GrowthScore & { value: number }) | null;
	};
};
// gap marks the first point after a table seam: the line must not be joined across it.
export type CurvePoint = {
	ageMonths: number;
	metric: number;
	imperial: number | null;
	gap: boolean;
};
export type CurveLine = { percentile: number; points: CurvePoint[] };
export type Curves = {
	source: GrowthSourceId;
	measure: Measure;
	sex: Sex;
	covered: boolean;
	reason: string | null;
	fromMonths: number;
	toMonths: number;
	lines: CurveLine[];
	correctedAgeAxis: boolean;
};
export type CurvesQuery = { measure: Measure; fromMonths?: number; toMonths?: number };
export type GrowthSourceInfo = {
	id: GrowthSourceId;
	name: string;
	attribution: string;
	licence: string;
	version: string;
	originUrl: string;
	retrieved: string;
	coverage: { measure: Measure; fromMonths: number; toMonths: number }[];
};

export type VaccineStatus = 'given' | 'due' | 'upcoming' | 'overdue' | 'dismissed' | 'available';
export type VaccineKind = 'routine' | 'may_not_be_needed' | 'supplementary' | 'recurring';
export type VaccineAgeUnit = 'd' | 'w' | 'm' | 'y';
export type VaccineAge = { n: number; unit: VaccineAgeUnit };
export type VaccineDismissReason = 'not_needed' | 'given_unknown';
export type VaccineSchemaInfo = {
	id: string;
	name: string;
	authority: string;
	edition: string;
	retrieved: string;
	originUrl: string;
	licence: string;
	attribution: string;
};
// Dates are YYYY-MM-DD in the kid's zone; untilDate is the first day after the window.
export type VaccineLine = {
	id: string;
	group: string;
	antigen: string;
	series: number;
	kind: VaccineKind;
	status: VaccineStatus;
	fromDate: string;
	untilDate: string;
	givenOn: string | null;
	administration: string | null;
	dismissal: { reason: VaccineDismissReason } | null;
	note: string;
};
export type VaccineGroup = {
	from: VaccineAge;
	fromDate: string;
	status: VaccineStatus;
	lines: VaccineLine[];
};
export type VaccineRecurring = {
	antigen: string;
	lineId: string;
	doses: { administration: string; givenOn: string }[];
};
export type VaccineOther = {
	administration: string;
	antigen: string;
	givenOn: string;
	product: string | null;
};
export type Vaccines = {
	schema: (VaccineSchemaInfo & { omitted: string[] }) | null;
	today: string;
	groups: VaccineGroup[];
	also: VaccineLine[];
	recurring: VaccineRecurring[];
	other: VaccineOther[];
};
export type VaccineCatalogue = {
	antigens: { id: string; name: string }[];
	products: { id: string; name: string; antigens: string[] }[];
};
export type Administration = {
	id: string;
	givenOn: string;
	antigens: string[];
	lot: string | null;
	product: string | null;
	note: string | null;
};
// The JSON body of a create or update, in the API's names; on update null clears lot, product or note.
export type AdministrationInput = {
	given_on?: string;
	antigens?: string[];
	lot?: string | null;
	product?: string | null;
	note?: string | null;
};
export type HomeDue = {
	kind: 'vaccine';
	from: VaccineAge;
	status: 'due' | 'overdue';
	count: number;
};
