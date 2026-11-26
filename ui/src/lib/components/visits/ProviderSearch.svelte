<script lang="ts">
	import Field from '#lib/components/entry/Field.svelte';
	import RadioField from '#lib/components/entry/RadioField.svelte';
	import { SELECT_CLASS } from '#lib/components/field.js';
	import { Alert } from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Separator } from '#lib/components/ui/separator/index.js';
	import { NPI_STATES, NPI_STATE_LABEL } from '#lib/constants.js';
	import type { ProviderPanel } from '#lib/server/provider-form.js';
	import type { ProviderResult } from '#lib/types.js';

	let { panel }: { panel: ProviderPanel | undefined } = $props();

	const BLANK = {
		kind: 'person',
		last_name: '',
		first_name: '',
		organization: '',
		state: '',
		city: '',
		postal_code: ''
	};
	const query = $derived(panel?.query ?? BLANK);
	const page = $derived(panel?.page);
	// Open again after a search, so its outcome is in view; closed after a pick.
	const open = $derived(!!(panel && (panel.page || panel.error || panel.field)));
	const error = (field: string) => (panel?.field === field ? (panel.message ?? null) : null);

	const KINDS = ['person', 'organization'] as const;
	const KIND_LABEL = { person: 'A doctor', organization: 'A clinic or practice' };
	const ERRORS = {
		unavailable:
			'The provider search isn’t available right now. You can type the doctor or clinic yourself.',
		rate_limited:
			'Too many searches, try again in a minute. You can type the doctor or clinic yourself.'
	};
	const ROW_FIELDS = ['npi', 'name', 'credential', 'taxonomy', 'address', 'phone'] as const;

	const options = $derived((page?.results ?? []).map((_, i) => String(i)));
	const label = (r: ProviderResult) => (r.credential ? `${r.name}, ${r.credential}` : r.name);
	const description = (r: ProviderResult) =>
		[r.taxonomy, r.address, r.phone].filter(Boolean).join(' · ');
	const labels = $derived(Object.fromEntries((page?.results ?? []).map((r, i) => [i, label(r)])));
	const descriptions = $derived(
		Object.fromEntries((page?.results ?? []).map((r, i) => [i, description(r)]))
	);

	let search = $state<HTMLButtonElement | null>(null);
	// With JavaScript, Enter in the panel searches; elsewhere in the form it saves.
	function enterSearches(node: HTMLElement) {
		const onkeydown = (e: KeyboardEvent) => {
			if (e.key !== 'Enter' || !(e.target instanceof HTMLInputElement)) return;
			e.preventDefault();
			search?.click();
		};
		node.addEventListener('keydown', onkeydown);
		return () => node.removeEventListener('keydown', onkeydown);
	}
</script>

<details class="group/npi" {open}>
	<summary class="list-item min-h-11 cursor-pointer py-2.5 text-[15px] leading-6 font-semibold">
		Find a US provider
	</summary>
	<div class="mt-2 flex flex-col gap-3.5 rounded-2xl border bg-card p-4" {@attach enterSearches}>
		<p class="text-[13.5px] leading-snug text-muted-foreground">
			Only US doctors and clinics are listed. Your search is sent to the US national provider
			registry (NPPES).
		</p>
		<RadioField
			name="npi_kind"
			legend="Search for"
			options={KINDS}
			labels={KIND_LABEL}
			values={{ npi_kind: query.kind === 'organization' ? 'organization' : 'person' }}
			error={() => null}
		/>
		<!-- CSS shows the chosen kind's fields, so this works without JavaScript. -->
		<div class="flex flex-col gap-3.5 group-has-[[value=organization]:checked]/npi:hidden">
			<Field id="npi-last_name" label="Last name" error={error('last_name')}>
				{#snippet children(control)}
					<Input {...control} name="npi_last_name" autocomplete="off" value={query.last_name} />
				{/snippet}
			</Field>
			<Field id="npi-first_name" label="First name (optional)" error={error('first_name')}>
				{#snippet children(control)}
					<Input {...control} name="npi_first_name" autocomplete="off" value={query.first_name} />
				{/snippet}
			</Field>
		</div>
		<div class="hidden flex-col gap-3.5 group-has-[[value=organization]:checked]/npi:flex">
			<Field id="npi-organization" label="Clinic or practice name" error={error('organization')}>
				{#snippet children(control)}
					<Input
						{...control}
						name="npi_organization"
						autocomplete="off"
						value={query.organization}
					/>
				{/snippet}
			</Field>
		</div>
		<Field id="npi-state" label="State" error={error('state')}>
			{#snippet children(control)}
				<select {...control} name="npi_state" class={SELECT_CLASS} value={query.state}>
					<option value="">Choose a state</option>
					{#each NPI_STATES as code (code)}
						<option value={code}>{NPI_STATE_LABEL[code]}</option>
					{/each}
				</select>
			{/snippet}
		</Field>
		<Field id="npi-city" label="City (optional)" error={error('city')}>
			{#snippet children(control)}
				<Input {...control} name="npi_city" autocomplete="off" value={query.city} />
			{/snippet}
		</Field>
		<Field id="npi-postal_code" label="ZIP code (optional)" error={error('postal_code')}>
			{#snippet children(control)}
				<Input
					{...control}
					name="npi_postal_code"
					inputmode="numeric"
					autocomplete="off"
					value={query.postal_code}
				/>
			{/snippet}
		</Field>
		<Button
			bind:ref={search}
			type="submit"
			variant="outline"
			formaction="?/find"
			class="self-start px-5">Search</Button
		>
		{#if panel?.error}
			<Alert class="[overflow-wrap:anywhere]">{ERRORS[panel.error]}</Alert>
		{:else if page}
			<Separator />
			{#if page.results.length === 0}
				<p class="text-[13.5px] leading-snug">
					No US providers match. Check the spelling, or type the doctor or clinic yourself.
				</p>
			{:else}
				<RadioField
					name="npi_choice"
					legend="Choose a provider"
					{options}
					{labels}
					{descriptions}
					values={{}}
					error={(name) => (name === 'npi_choice' ? error('choice') : null)}
				/>
				{#each page.results as r, i (i)}
					{#each ROW_FIELDS as k (k)}
						<input type="hidden" name="npi_r{i}_{k}" value={r[k] ?? ''} />
					{/each}
				{/each}
				{#if page.more}
					<input type="hidden" name="npi_more" value="1" />
					<p class="text-[13.5px] leading-snug text-muted-foreground">
						There may be more. Add a first name, a city or a ZIP code to narrow the search.
					</p>
				{/if}
				<Button type="submit" variant="outline" formaction="?/pick" class="self-start px-5"
					>Use this provider</Button
				>
			{/if}
		{/if}
	</div>
</details>
