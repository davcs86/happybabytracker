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
export const KID_COOKIE = 'hbt_kid';
export const KID_COOKIE_MAX_AGE_S = 30 * 24 * 60 * 60;
// API ids are UUIDs; any other path parameter never reaches an API path.
export const ID_PATTERN = /^[A-Za-z0-9-]{1,64}$/;
// A catalogue antigen id, and a schedule line id in a path; anything else never reaches the API.
export const ANTIGEN_PATTERN = /^[a-z][a-z0-9_]*$/;
export const VACCINE_LINE_PATTERN = /^[a-z0-9][a-z0-9_-]{0,63}$/;
export const NOT_FOUND_MESSAGE = 'Not found';
export const FAILURE_MESSAGE = 'Something went wrong, try again';
export const UNREACHABLE_MESSAGE = 'The app cannot reach its server, try again soon';
