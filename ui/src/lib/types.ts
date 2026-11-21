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
