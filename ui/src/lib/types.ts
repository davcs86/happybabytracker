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
	reminderOverdueMinutes?: number;
};

export type Kid = Omit<
	KidInput,
	'sex' | 'growthSource' | 'vaccineSchema' | 'reminderOverdueMinutes'
> & {
	id: string;
	sex: Sex;
	growthSource: GrowthSourceId;
	// An id the API no longer ships counts as no schedule (schemaChosen).
	vaccineSchema: string | null;
	reminderOverdueMinutes: number;
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

export type TrackerKind =
	'bottles' | 'nursing' | 'pumping' | 'sleep' | 'diapers' | 'meals' | 'symptoms';

export type TimelineType =
	'bottle' | 'nursing' | 'pumping' | 'sleep' | 'diaper' | 'meal' | 'symptom';

export type VolumeUnit = Units['volume'];

// The API's own names: day totals share the shape, so formatVolume takes either.
export type Volume = { ml: number; floz_hundredths: number; typed_unit: VolumeUnit };

export type BottleContent = 'breast_milk' | 'formula' | 'cow_milk' | 'other';
export type DiaperKind = 'wet' | 'dirty' | 'both' | 'dry';
export type NursingSide = 'left' | 'right';
export type PumpingSide = NursingSide | 'both';

type Noted = { id: string; note: string | null };

// One source of an advanced bottle; sources is empty for a simple bottle, else one per content.
export type BottleSource = { source: BottleContent; volume: Volume };
export type Bottle = Noted & {
	at: string;
	contents: BottleContent[];
	volume: Volume;
	sources: BottleSource[];
};
export type Diaper = Noted & { at: string; kind: DiaperKind };
export type Meal = Noted & { at: string; what: string; reaction: string | null };
// One side of an advanced session; sides is empty for a simple one, else one per side pumped.
export type PumpingSideVolume = { side: NursingSide; volume: Volume };
export type Pumping = Noted & {
	startedAt: string;
	durationSeconds: number;
	volume: Volume;
	side: PumpingSide;
	sides: PumpingSideVolume[];
};
// A timer runs while endedAt is null.
export type Sleep = Noted & { startedAt: string; endedAt: string | null };
// The API's columns: the weight gain, and its volume at the assumed density; null when not defined.
export type Milk = {
	gain: { grams: number; ozHundredths: number };
	volume: { ml: number; flozHundredths: number };
};
export type Nursing = Noted & {
	startedAt: string;
	endedAt: string | null;
	leftSeconds: number;
	rightSeconds: number;
	side: NursingSide | null;
	sideSince: string | null;
	// A weigh-in has a growth weight's columns but is not a growth row.
	weightBefore: GrowthWeight | null;
	weightAfter: GrowthWeight | null;
	milk: Milk | null;
};

export type SymptomName =
	| 'congestion'
	| 'cough'
	| 'fever'
	| 'vomiting'
	| 'diarrhea'
	| 'constipation'
	| 'rash'
	| 'ear_pain'
	| 'sore_throat'
	| 'poor_appetite'
	| 'unusually_sleepy'
	| 'other';
export type Severity = 'mild' | 'moderate' | 'severe';
export type TempMethod = 'oral' | 'rectal' | 'axillary' | 'ear' | 'forehead' | 'other';
// Both columns; typedUnit names the exact one.
export type Temperature = { cHundredths: number; fHundredths: number; typedUnit: Units['temp'] };
export type Symptom = Noted & {
	at: string;
	symptom: SymptomName;
	severity: Severity | null;
	otherText: string | null;
	temperature: Temperature | null;
	method: TempMethod | null;
};

// times are HH:MM in the kid's zone; every is a grid from the course's start.
export type Schedule =
	{ kind: 'none' } | { kind: 'times'; times: string[] } | { kind: 'every'; hours: number };
export type Medication = Noted & {
	name: string;
	dose: string;
	schedule: Schedule;
	startAt: string;
	// A kid-local date, inclusive.
	endOn: string | null;
	active: boolean;
	lastDose: { id: string; givenAt: string; amount: string | null } | null;
	nextDue: { dueAt: string } | null;
	doseCount: number;
};
export type MedicationDose = Noted & {
	medication: string;
	givenAt: string;
	amount: string | null;
};
export type DoseQuery = ListQuery & { medication?: string };

// A browser's subscription is a secret of the device: it travels only in the body of an upsert.
export type PushState = { enabled: false } | { enabled: true; publicKey: string };
// kids are the kids this device is opted in for, filtered to the parent's current grants.
export type PushDevice = { id: string; label: string; createdAt: string; kids: string[] };
export type PushSubscriptionInput = {
	endpoint: string;
	keys: { p256dh: string; auth: string };
	// Absent: the API keeps a known device's name, or names a new one "This device".
	label?: string;
};
// topic is null until the parent turns ntfy on; it is a capability, shown only to its owner.
export type NtfyState =
	| { enabled: false; hasTopic: boolean }
	| { enabled: true; server: string; topic: string | null; kids: string[] };

export type TrackerEntries = {
	bottles: Bottle;
	nursing: Nursing;
	pumping: Pumping;
	sleep: Sleep;
	diapers: Diaper;
	meals: Meal;
	symptoms: Symptom;
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
export type HomeDueVaccine = {
	kind: 'vaccine';
	from: VaccineAge;
	status: 'due' | 'overdue';
	count: number;
};
// dueAt is an instant; the API decides which entries are listed, the UI applies no window.
export type HomeDueMedication = {
	kind: 'medication';
	medication: { id: string; name: string };
	dueAt: string;
	state: 'due' | 'overdue';
};
export type HomeDue = HomeDueVaccine | HomeDueMedication;

export type MilestoneDomain =
	'social_emotional' | 'language_communication' | 'cognitive' | 'movement_physical';
export type MilestonePosition = 'earlier' | 'current' | 'next' | 'later';
// Dates are YYYY-MM-DD in the kid's zone.
export type MilestoneReached = Noted & { reachedOn: string };
export type MilestoneItem = {
	id: string;
	domain: MilestoneDomain;
	text: string;
	reached: MilestoneReached | null;
};
export type MilestoneChecklist = {
	ageMonths: number;
	position: MilestonePosition;
	items: MilestoneItem[];
};
// checklists is empty once catalogShown is false; reached then lists the reached catalog rows.
export type Milestones = {
	catalog: {
		id: string;
		name: string;
		authority: string;
		edition: string;
		retrieved: string;
		originUrl: string;
		licence: string;
		attribution: string;
		screeningNote: string;
	};
	age: { chronologicalMonths: number; correctedMonths: number | null; basisMonths: number };
	catalogShown: boolean;
	checklists: MilestoneChecklist[];
	custom: (MilestoneReached & { text: string })[];
	retired: (MilestoneReached & { item: string; text: string })[];
	reached: (MilestoneReached & {
		item: string;
		ageMonths: number;
		domain: MilestoneDomain;
		text: string;
	})[];
};
// item is null for a custom milestone; text is the catalog's for an item.
export type MilestoneEntry = MilestoneReached & { item: string | null; text: string };
