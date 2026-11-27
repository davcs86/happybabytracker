// Pages whose default state the stub API data reproduces, with the share of pixels that may differ.
// A layout tolerance, not pixel equality: intentional copy differences and 1-2 px line-box offsets remain.
export const MATCHED_PAGES: Record<string, number> = {
	'sign-in': 0.08,
	'set-password': 0.08,
	error: 0.08
};
