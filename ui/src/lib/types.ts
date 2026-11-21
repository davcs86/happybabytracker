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
