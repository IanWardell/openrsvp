<script lang="ts">
	import { goto } from '$app/navigation';
	import { currentUser, isLoading, isAdmin } from '$lib/stores/auth';
	import { isSuperAdmin } from '$lib/stores/auth';
	import AppShell from '$lib/components/layout/AppShell.svelte';

	$effect(() => {
		if (!$isLoading && (!$currentUser || !$isAdmin)) {
			goto('/events');
		}
	});

	let { children } = $props();
</script>

{#if $isLoading}
	<div class="flex items-center justify-center min-h-screen">
		<div class="loading-spinner"></div>
	</div>
{:else if $currentUser && $isAdmin}
	<AppShell>
		<div class="mb-7 border-b border-neutral-200">
			<nav class="flex flex-wrap gap-1" aria-label="Admin navigation">
				<a href="/admin" class="px-3 py-2 text-sm font-medium text-neutral-700 hover:text-primary">Overview</a>
				<a href="/admin/events" class="px-3 py-2 text-sm font-medium text-neutral-700 hover:text-primary">Events</a>
				<a href="/admin/users" class="px-3 py-2 text-sm font-medium text-neutral-700 hover:text-primary">Users</a>
				{#if $isSuperAdmin}
					<a href="/admin/invitations/events" class="px-3 py-2 text-sm font-medium text-neutral-700 hover:text-primary">Event Invitations</a>
					<a href="/admin/invitations/guests" class="px-3 py-2 text-sm font-medium text-neutral-700 hover:text-primary">Guest Invitations</a>
					<a href="/admin/audit" class="px-3 py-2 text-sm font-medium text-neutral-700 hover:text-primary">Audit Log</a>
					<a href="/setup" class="px-3 py-2 text-sm font-medium text-neutral-700 hover:text-primary">Settings</a>
				{/if}
			</nav>
		</div>
		{@render children()}
	</AppShell>
{/if}
