import { createHash, createHmac, generateKeyPairSync, randomBytes, sign } from 'node:crypto';

export type IssuerUser = { sub: string; preferredUsername?: string };

export type IssuerBehaviour = {
	user: IssuerUser;
	reauthUser?: IssuerUser;
	deny?: boolean;
	idToken?: Partial<{
		iss: string;
		aud: string;
		expOffsetS: number;
		alg: string;
		nonce: string;
		authTime: number | null;
		badSignature: boolean;
	}>;
	userinfoFails?: boolean;
	jwksFails?: boolean;
	down?: boolean;
};

export type Issuer = {
	handle(req: Request): Promise<Response>;
	configure(b: Partial<IssuerBehaviour>): void;
	reset(): void;
	authorizeLog: Array<{ prompt: string | null; maxAge: string | null }>;
};

const DEFAULT_USER: IssuerUser = { sub: 'stub-subject-0001', preferredUsername: 'stub-user' };
const KID = 'stub-key-1';
const EC_KID = 'stub-key-ec';

type Grant = {
	redirectUri: string;
	challenge: string;
	nonce: string | null;
	user: IssuerUser;
	authTime: number;
};

const b64u = (v: string | Buffer) => Buffer.from(v).toString('base64url');
const nowS = () => Math.floor(Date.now() / 1000);

function json(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'content-type': 'application/json', 'cache-control': 'no-store' }
	});
}

export function createIssuer(opts: {
	issuer: string;
	clientId: string;
	clientSecret: string;
}): Issuer {
	const signing = generateKeyPairSync('rsa', { modulusLength: 2048 });
	const impostor = generateKeyPairSync('rsa', { modulusLength: 2048 });
	// A published ES256 key, so a client that does not pin RS256 would accept an ES256 token.
	const ec = generateKeyPairSync('ec', { namedCurve: 'P-256' });
	const ecJwk = {
		...ec.publicKey.export({ format: 'jwk' }),
		kid: EC_KID,
		alg: 'ES256',
		use: 'sig'
	};
	const jwk = {
		...signing.publicKey.export({ format: 'jwk' }),
		kid: KID,
		alg: 'RS256',
		use: 'sig'
	};
	const base = opts.issuer.replace(/\/$/, '');
	const basePath = new URL(base).pathname.replace(/\/$/, '');

	let behaviour: IssuerBehaviour = { user: DEFAULT_USER };
	const codes = new Map<string, Grant>();
	const tokens = new Map<string, IssuerUser>();
	const authorizeLog: Issuer['authorizeLog'] = [];

	function idToken(grant: Grant): string {
		const t = behaviour.idToken ?? {};
		const alg = t.alg ?? 'RS256';
		const now = nowS();
		const claims: Record<string, unknown> = {
			iss: t.iss ?? opts.issuer,
			aud: t.aud ?? opts.clientId,
			sub: grant.user.sub,
			iat: now,
			exp: now + (t.expOffsetS ?? 300)
		};
		const nonce = t.nonce ?? grant.nonce;
		if (nonce !== null) claims.nonce = nonce;
		const authTime = t.authTime === undefined ? grant.authTime : t.authTime;
		if (authTime !== null) claims.auth_time = authTime;
		if (grant.user.preferredUsername !== undefined) {
			claims.preferred_username = grant.user.preferredUsername;
		}
		const kid = alg === 'ES256' ? EC_KID : KID;
		const input = `${b64u(JSON.stringify({ alg, typ: 'JWT', kid }))}.${b64u(JSON.stringify(claims))}`;
		if (alg === 'none') return `${input}.`;
		if (alg === 'HS256') {
			return `${input}.${createHmac('sha256', opts.clientSecret).update(input).digest('base64url')}`;
		}
		if (alg === 'ES256') {
			const sig = sign('sha256', Buffer.from(input), {
				key: ec.privateKey,
				dsaEncoding: 'ieee-p1363'
			});
			return `${input}.${sig.toString('base64url')}`;
		}
		const key = t.badSignature ? impostor.privateKey : signing.privateKey;
		return `${input}.${sign('sha256', Buffer.from(input), key).toString('base64url')}`;
	}

	function authorize(url: URL): Response {
		const q = url.searchParams;
		const redirectUri = q.get('redirect_uri') ?? '';
		authorizeLog.push({ prompt: q.get('prompt'), maxAge: q.get('max_age') });
		if (q.get('client_id') !== opts.clientId || !URL.canParse(redirectUri)) {
			return json({ error: 'invalid_request' }, 400);
		}
		const back = new URL(redirectUri);
		const state = q.get('state');
		if (state !== null) back.searchParams.set('state', state);
		const challenge = q.get('code_challenge');
		if (behaviour.deny) {
			back.searchParams.set('error', 'access_denied');
		} else if (
			q.get('response_type') !== 'code' ||
			q.get('code_challenge_method') !== 'S256' ||
			!challenge
		) {
			back.searchParams.set('error', 'invalid_request');
		} else {
			const reauth = q.get('prompt') === 'login' && behaviour.reauthUser;
			const code = randomBytes(32).toString('base64url');
			codes.set(code, {
				redirectUri,
				challenge,
				nonce: q.get('nonce'),
				user: reauth || behaviour.user,
				authTime: nowS()
			});
			back.searchParams.set('code', code);
		}
		return new Response(null, { status: 302, headers: { location: back.href } });
	}

	function clientOk(req: Request): boolean {
		const [scheme, value] = (req.headers.get('authorization') ?? '').split(' ');
		if (scheme !== 'Basic' || !value) return false;
		const [id, secret] = Buffer.from(value, 'base64').toString('utf8').split(':');
		return (
			decodeURIComponent(id ?? '') === opts.clientId &&
			decodeURIComponent(secret ?? '') === opts.clientSecret
		);
	}

	async function token(req: Request): Promise<Response> {
		if (!clientOk(req)) return json({ error: 'invalid_client' }, 401);
		const form = new URLSearchParams(await req.text());
		const code = form.get('code') ?? '';
		const grant = codes.get(code);
		codes.delete(code);
		const verifier = form.get('code_verifier') ?? '';
		if (
			form.get('grant_type') !== 'authorization_code' ||
			!grant ||
			form.get('redirect_uri') !== grant.redirectUri ||
			createHash('sha256').update(verifier).digest('base64url') !== grant.challenge
		) {
			return json({ error: 'invalid_grant' }, 400);
		}
		const accessToken = randomBytes(32).toString('base64url');
		tokens.set(accessToken, grant.user);
		return json({
			access_token: accessToken,
			token_type: 'Bearer',
			expires_in: 300,
			id_token: idToken(grant)
		});
	}

	function userinfo(req: Request): Response {
		if (behaviour.userinfoFails) return json({ error: 'server_error' }, 503);
		const [scheme, value] = (req.headers.get('authorization') ?? '').split(' ');
		const user = scheme === 'Bearer' ? tokens.get(value ?? '') : undefined;
		if (!user) return json({ error: 'invalid_token' }, 401);
		return json({
			sub: user.sub,
			...(user.preferredUsername === undefined
				? {}
				: { preferred_username: user.preferredUsername })
		});
	}

	async function handle(req: Request): Promise<Response> {
		if (behaviour.down) return json({ error: 'temporarily_unavailable' }, 503);
		const url = new URL(req.url);
		const path = url.pathname.startsWith(basePath) ? url.pathname.slice(basePath.length) : '';
		switch (`${req.method} ${path}`) {
			case 'GET /.well-known/openid-configuration':
				return json({
					issuer: opts.issuer,
					authorization_endpoint: `${base}/authorize`,
					token_endpoint: `${base}/token`,
					userinfo_endpoint: `${base}/userinfo`,
					jwks_uri: `${base}/jwks.json`,
					response_types_supported: ['code'],
					subject_types_supported: ['public'],
					id_token_signing_alg_values_supported: ['RS256', 'HS256', 'ES256'],
					token_endpoint_auth_methods_supported: ['client_secret_basic'],
					code_challenge_methods_supported: ['S256']
				});
			case 'GET /jwks.json':
				return behaviour.jwksFails
					? json({ error: 'unavailable' }, 503)
					: json({ keys: [jwk, ecJwk] });
			case 'GET /authorize':
				return authorize(url);
			case 'POST /token':
				return token(req);
			case 'GET /userinfo':
				return userinfo(req);
			default:
				return json({ error: 'not_found' }, 404);
		}
	}

	return {
		handle,
		configure(b) {
			behaviour = { ...behaviour, ...b };
		},
		reset() {
			behaviour = { user: DEFAULT_USER };
			codes.clear();
			tokens.clear();
			authorizeLog.length = 0;
		},
		authorizeLog
	};
}
