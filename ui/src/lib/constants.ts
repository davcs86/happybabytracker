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
export const DEFAULT_TIME_ZONE = 'UTC';
export const INVITE_VALIDITY = '72 hours';
export const RESET_VALIDITY = '1 hour';
