export const SESSION_COOKIE = 'hbt_session';
export const PARENT_SESSION_MAX_AGE_S = 30 * 24 * 60 * 60;
export const ADMIN_SESSION_MAX_AGE_S = 12 * 60 * 60;
export const API_TIMEOUT_MS = 5000;
export const MIN_INTERNAL_API_TOKEN = 32;
export const MAX_PORT = 65535;
// One-time tokens are 43 base64url characters; anything else never reaches the API path.
export const TOKEN_PATTERN = /^[A-Za-z0-9_-]{1,128}$/;
export const ADMIN_HOME = '/admin/parents';
export const PARENT_HOME = '/';
export const UNAVAILABLE_MESSAGE = 'The app cannot reach its server right now';
