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

export type KidInput = {
	name: string;
	birthDate: string;
	sex: string;
	gestationWeeks: number | null;
	gestationDays: number | null;
	timeZone: string;
};

export type Kid = Omit<KidInput, 'sex'> & { id: string; sex: Sex; role: Role };

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
	band: AgeBand;
	modules: ModuleState[];
	running: { sleep: Sleep | null; nursing: Nursing | null };
	today: DaySummary;
};
