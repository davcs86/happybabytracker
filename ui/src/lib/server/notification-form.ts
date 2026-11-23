import { PUSH_LABEL_MAX_LEN } from '../constants';
import { ID_PATTERN } from './constants';

export type DeviceField = 'label' | 'kid';
export type DeviceFormValues = { label: string; kids: string[] };

// Never a typed value in a message.
export const DEVICE_FIELD_MESSAGES: Record<DeviceField, string> = {
	label: `Enter a name of 1 to ${PUSH_LABEL_MAX_LEN} characters`,
	kid: 'Choose from your children'
};

// Value-free answers to the API's push and ntfy refusals.
export const NTFY_OFF_MESSAGE = 'ntfy is not configured on this server.';
export const DEVICE_LIMIT_MESSAGE =
	'You have the most devices allowed. Remove one, then try again.';
export const SUBSCRIPTION_REFUSED_MESSAGE =
	"The server refused this browser's notification address. Try again later.";

// The label and the kids switched on (one `kid` field each); the API checks which of them the parent holds.
export function readDeviceForm(form: FormData): {
	values: DeviceFormValues;
	input: DeviceFormValues | null;
	field: DeviceField | null;
} {
	const kids = form.getAll('kid');
	const typed = String(form.get('label') ?? '');
	const values = {
		label: typed,
		kids: [...new Set(kids.filter((k): k is string => typeof k === 'string'))]
	};
	const label = typed.trim();
	const length = [...label].length;
	if (length < 1 || length > PUSH_LABEL_MAX_LEN) return { values, input: null, field: 'label' };
	if (kids.some((k) => typeof k !== 'string' || !ID_PATTERN.test(k))) {
		return { values, input: null, field: 'kid' };
	}
	return { values, input: { label, kids: values.kids }, field: null };
}
