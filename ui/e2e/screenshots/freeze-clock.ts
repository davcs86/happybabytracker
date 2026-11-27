// Preloaded with node --import into the stub and the UI server; tests never import it.
const at = Date.parse(process.env.SHOTS_NOW ?? '');
if (Number.isNaN(at)) throw new Error('freeze-clock: SHOTS_NOW is not set');

const RealDate = Date;
class FrozenDate extends RealDate {
	constructor(...args: unknown[]) {
		if (args.length === 0) super(at);
		else super(...(args as [string]));
	}
	static now() {
		return at;
	}
}
globalThis.Date = FrozenDate as DateConstructor;
