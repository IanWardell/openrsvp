<script lang="ts">
	import '../app.css';
	import { currentUser, isLoading } from '$lib/stores/auth';
	import { api } from '$lib/api/client';
	import { onMount } from 'svelte';
	import Toast from '$lib/components/ui/Toast.svelte';

	onMount(async () => {
		const userAtStart = $currentUser;
		try {
			const user = await api.get<import('$lib/types').Organizer>('/auth/me');
			// A magic-link verification can complete while this bootstrap request
			// is in flight. Never overwrite that newer authenticated state.
			if ($currentUser !== userAtStart && $currentUser !== null) return;
			$currentUser = user;

			// Auto-save browser timezone to profile if not set yet.
			if (!user.timezone) {
				const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
				if (tz) {
					api.patch<import('$lib/types').Organizer>('/auth/me', { timezone: tz })
						.then((updated) => { $currentUser = updated; })
						.catch(() => {});
				}
			}
		} catch {
			if ($currentUser === userAtStart) {
				$currentUser = null;
			}
		} finally {
			$isLoading = false;
		}
	});

	let { children } = $props();
</script>

<div class="min-h-screen bg-neutral-50">
	{@render children()}
	<Toast />
</div>
