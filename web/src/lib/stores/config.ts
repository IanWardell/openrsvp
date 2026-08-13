import { writable } from 'svelte/store';
import { api } from '$lib/api/client';

export const smsEnabled = writable(false);
// Default closed until the public policy is loaded. This prevents a prerendered
// page from briefly advertising registration on a closed instance.
export const allowSignups = writable(false);
export const appConfigLoaded = writable(false);

let loaded = false;

export async function loadAppConfig(force = false) {
	if (loaded && !force) return;
	try {
		const result = await api.get<{ data: { smsEnabled: boolean; allowSignups: boolean } }>('/config');
		smsEnabled.set(result.data.smsEnabled);
		allowSignups.set(result.data.allowSignups);
		loaded = true;
	} catch {
		smsEnabled.set(false);
		allowSignups.set(false);
	} finally {
		appConfigLoaded.set(true);
	}
}
