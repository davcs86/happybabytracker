export const SESSION_COOKIE = 'hbt_session';
export const PARENT_SESSION_MAX_AGE_S = 30 * 24 * 60 * 60;
export const ADMIN_SESSION_MAX_AGE_S = 12 * 60 * 60;
export const API_TIMEOUT_MS = 5000;
// Under the API's 30 s WriteTimeout, so the API's own answer arrives first.
export const EXPORT_TIMEOUT_MS = 25000;
export const EXPORT_DEFAULT_DAYS = 30;
// The only answer streamed to the browser: an attachment named by its two dates.
export const EXPORT_DISPOSITION_PATTERN =
	/^attachment; filename="happybabytracker-\d{4}-\d{2}-\d{2}-\d{4}-\d{2}-\d{2}\.pdf"$/;
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
// A milestone catalog item id (the API's idPattern; the 48 is MilestoneCatalogSlugMaxLen); anything else never
// reaches the API.
export const MILESTONE_ITEM_PATTERN = /^[0-9]{1,2}m-(se|lc|cg|mp)-[a-z0-9-]{1,48}$/;
export const NOT_FOUND_MESSAGE = 'Not found';
export const FAILURE_MESSAGE = 'Something went wrong, try again';
export const UNREACHABLE_MESSAGE = 'The app cannot reach its server, try again soon';
// Holds this browser's push device id (an id, not a secret) so a form can name the device without JavaScript.
export const PUSH_DEVICE_COOKIE = 'hbt_push_device';
// Renewed on each visit that re-sends the subscription; a lapsed cookie only stops the automatic re-send.
export const PUSH_DEVICE_COOKIE_MAX_AGE_S = 365 * 24 * 60 * 60;
// Holds the display theme only: light, dark or auto.
export const THEME_COOKIE = 'hbt_theme';
export const THEME_COOKIE_MAX_AGE_S = 365 * 24 * 60 * 60;
// A same-origin path with its query: one leading slash, no backslash, no whitespace.
export const RETURN_PATH_PATTERN = /^\/(?![/\\])[^\s\\]*$/;
