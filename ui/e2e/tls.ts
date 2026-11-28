import { execFileSync } from 'node:child_process';
import { existsSync, mkdirSync } from 'node:fs';
import { join, resolve } from 'node:path';

// The stub issuer's self-signed certificate for 127.0.0.1; never committed (.gitignore).
export const TLS_DIR = resolve(import.meta.dirname, '.tls');

const fresh = (cert: string) => {
	try {
		execFileSync('openssl', ['x509', '-checkend', '3600', '-noout', '-in', cert], {
			stdio: 'ignore'
		});
		return true;
	} catch {
		return false;
	}
};

export function ensureTls(): { cert: string; key: string } {
	const cert = join(TLS_DIR, 'cert.pem');
	const key = join(TLS_DIR, 'key.pem');
	if (existsSync(cert) && existsSync(key) && fresh(cert)) return { cert, key };
	mkdirSync(TLS_DIR, { recursive: true });
	execFileSync(
		'openssl',
		[
			'req',
			'-x509',
			'-newkey',
			'ec',
			'-pkeyopt',
			'ec_paramgen_curve:P-256',
			'-nodes',
			'-days',
			'7',
			'-subj',
			'/CN=127.0.0.1',
			'-addext',
			'subjectAltName=IP:127.0.0.1',
			'-keyout',
			key,
			'-out',
			cert
		],
		{ stdio: 'ignore' }
	);
	return { cert, key };
}
