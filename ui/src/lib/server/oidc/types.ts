import type { SessionKind } from '../../types';

export type OidcPurpose = 'login' | 'invite' | 'link';

export type OidcStart = {
	stage: 'start';
	purpose: OidcPurpose;
	state: string;
	nonce: string;
	verifier: string;
	startedAt: number;
	next: string | null;
	binding: string | null;
	username: string | null;
	kind: SessionKind | null;
	deviceLabel: string | null;
	exp: number;
};

export type OidcPending = {
	stage: 'pending';
	purpose: 'invite' | 'link';
	binding: string;
	username: string;
	kind: SessionKind | null;
	iss: string;
	sub: string;
	label: string | null;
	deviceLabel: string | null;
	exp: number;
};

export type OidcCookie = OidcStart | OidcPending;
