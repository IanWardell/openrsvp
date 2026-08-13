<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import { toast } from '$lib/stores/toast';
	import { getTimezoneOptions } from '$lib/utils/timezones';
	import type { ApiResponse } from '$lib/types';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';

	interface InstanceSettings {
		instanceName: string;
		defaultTimezone: string;
		allowSignups: boolean;
		supportEmail: string;
		notifyNewOrganizer: boolean;
		notifyNewAdmin: boolean;
		notifyNewSuperAdmin: boolean;
	}

	const browserTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
	const timezoneOptions = getTimezoneOptions(browserTimezone);
	const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

	let loading = $state(true);
	let saving = $state(false);
	let instanceName = $state('');
	let defaultTimezone = $state(browserTimezone);
	let allowSignups = $state(true);
	let supportEmail = $state('');
	let notifyNewOrganizer = $state(false);
	let notifyNewAdmin = $state(false);
	let notifyNewSuperAdmin = $state(false);
	let errors: Record<string, string> = $state({});

	onMount(async () => {
		try {
			const response = await api.get<ApiResponse<InstanceSettings>>('/setup/config');
			instanceName = response.data.instanceName;
			defaultTimezone = response.data.defaultTimezone || browserTimezone;
			allowSignups = response.data.allowSignups;
			supportEmail = response.data.supportEmail;
			notifyNewOrganizer = response.data.notifyNewOrganizer;
			notifyNewAdmin = response.data.notifyNewAdmin;
			notifyNewSuperAdmin = response.data.notifyNewSuperAdmin;
		} catch (error: any) {
			toast.error(error.message || 'Failed to load instance settings');
		} finally {
			loading = false;
		}
	});

	function validate() {
		errors = {};
		if (!instanceName.trim()) errors.instanceName = 'Instance name is required';
		if (supportEmail.trim() && !emailPattern.test(supportEmail.trim())) {
			errors.supportEmail = 'Enter a valid email address';
		}
		return Object.keys(errors).length === 0;
	}

	async function save() {
		if (!validate()) return;
		saving = true;
		try {
			await api.post('/setup/config', {
				instanceName: instanceName.trim(),
				defaultTimezone,
				allowSignups,
				supportEmail: supportEmail.trim(),
				notifyNewOrganizer,
				notifyNewAdmin,
				notifyNewSuperAdmin
			});
			toast.success('Instance settings saved');
		} catch (error: any) {
			toast.error(error.message || 'Failed to save instance settings');
		} finally {
			saving = false;
		}
	}
</script>

<svelte:head><title>Settings — OpenRSVP Admin</title></svelte:head>

<div class="max-w-3xl space-y-6">
	<div>
		<h1 class="text-2xl font-bold font-display">Instance Settings</h1>
		<p class="mt-1 text-sm text-neutral-500">Manage organizer access and super-admin notifications without rerunning setup.</p>
	</div>

	{#if loading}
		<div class="flex justify-center py-20"><Spinner /></div>
	{:else}
		<Card>
			<div class="space-y-6">
				<Input label="Instance name" name="instanceName" bind:value={instanceName} error={errors.instanceName || ''} required />
				<div>
					<Select label="Default timezone" name="defaultTimezone" bind:value={defaultTimezone} options={timezoneOptions} />
					<p class="mt-1 text-sm text-neutral-500">Used as the default for newly created events.</p>
				</div>
				<Input label="Support email (optional)" name="supportEmail" type="email" bind:value={supportEmail} error={errors.supportEmail || ''} />
			</div>
		</Card>

		<Card>
			<div class="space-y-5">
				<div>
					<h2 class="font-display text-lg font-semibold">Organizer access</h2>
					<p class="mt-1 text-sm text-neutral-500">Control whether unknown email addresses may create organizer accounts.</p>
				</div>
				<label class="flex items-start gap-3 cursor-pointer">
					<input type="checkbox" bind:checked={allowSignups} class="mt-0.5 rounded border-neutral-300 text-primary focus:ring-primary/40" />
					<span><span class="text-sm font-medium text-neutral-700">Allow public organizer signups</span><span class="mt-1 block text-xs text-neutral-500">When disabled, existing accounts and deployment-managed admins can still sign in. New organizers must be invited from Users.</span></span>
				</label>
			</div>
		</Card>

		<Card>
			<div class="space-y-5">
				<div>
					<h2 class="font-display text-lg font-semibold">Super-admin notifications</h2>
					<p class="mt-1 text-sm text-neutral-500">Email active super admins when accounts are created or promoted. These notices are off by default.</p>
				</div>
				{#each [
					{ label: 'New organizers', description: 'Notify when a new organizer account is created.', get: () => notifyNewOrganizer, set: (value: boolean) => (notifyNewOrganizer = value) },
					{ label: 'New admins', description: 'Notify when an organizer is promoted to admin.', get: () => notifyNewAdmin, set: (value: boolean) => (notifyNewAdmin = value) },
					{ label: 'New super admins', description: 'Notify when an account is promoted to super admin.', get: () => notifyNewSuperAdmin, set: (value: boolean) => (notifyNewSuperAdmin = value) }
				] as preference}
					<label class="flex items-start gap-3 cursor-pointer">
						<input type="checkbox" checked={preference.get()} onchange={(event) => preference.set(event.currentTarget.checked)} class="mt-0.5 rounded border-neutral-300 text-primary focus:ring-primary/40" />
						<span><span class="text-sm font-medium text-neutral-700">{preference.label}</span><span class="mt-1 block text-xs text-neutral-500">{preference.description}</span></span>
					</label>
				{/each}
			</div>
		</Card>

		<div class="flex justify-end">
			<Button onclick={save} loading={saving}>Save settings</Button>
		</div>
	{/if}
</div>
