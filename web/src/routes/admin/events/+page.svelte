<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import AdminPagination from '$lib/components/admin/AdminPagination.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Modal from '$lib/components/ui/Modal.svelte';
	import { isSuperAdmin } from '$lib/stores/auth';
	import { toast } from '$lib/stores/toast';
	import type { AdminEvent, AdminPage, AdminPagination as Pagination } from '$lib/types';

	type EventAction = 'suspend' | 'restore' | 'delete';
	let events = $state<AdminEvent[]>([]);
	let search = $state('');
	let status = $state('');
	let loading = $state(true);
	let submitting = $state(false);
	let pagination = $state<Pagination>({ page: 1, pageSize: 25, total: 0, totalPages: 0 });
	let actionOpen = $state(false);
	let actionKind = $state<EventAction>('suspend');
	let actionTarget = $state<AdminEvent | null>(null);
	let reason = $state('');
	let confirmation = $state('');

	async function load(page = pagination.page) {
		loading = true;
		try {
			const result = await api.get<AdminPage<AdminEvent>>(`/admin/events?search=${encodeURIComponent(search)}&status=${encodeURIComponent(status)}&page=${page}&pageSize=${pagination.pageSize}`);
			events = result.data;
			pagination = result.pagination;
		} catch (error: any) {
			toast.error(error.message || 'Failed to load events');
		} finally {
			loading = false;
		}
	}

	function openAction(kind: EventAction, event: AdminEvent) {
		actionKind = kind;
		actionTarget = event;
		reason = '';
		confirmation = '';
		actionOpen = true;
	}

	function actionReady() {
		if (!actionTarget || !reason.trim()) return false;
		return actionKind !== 'delete' || confirmation === actionTarget.title;
	}

	async function submitAction() {
		if (!actionTarget || !actionReady()) return;
		submitting = true;
		try {
			if (actionKind === 'delete') {
				await api.delete(`/admin/events/${actionTarget.id}`, { confirmation, reason: reason.trim() });
			} else if (actionKind === 'suspend') {
				await api.post(`/admin/events/${actionTarget.id}/suspend`, { reason: reason.trim() });
			} else {
				await api.post(`/admin/events/${actionTarget.id}/restore`, { reason: reason.trim() });
			}
			toast.success(actionKind === 'delete' ? 'Event permanently deleted' : 'Event updated');
			actionOpen = false;
			await load();
		} catch (error: any) {
			toast.error(error.message || 'Action failed');
		} finally {
			submitting = false;
		}
	}

	onMount(() => load(1));
</script>

<svelte:head><title>Events — OpenRSVP Admin</title></svelte:head>

<div class="space-y-5">
	<div><h1 class="text-2xl font-bold font-display">All events</h1><p class="text-sm text-neutral-500 mt-1">Review, suspend, restore, or remove platform events.</p></div>
	<form class="flex flex-wrap gap-2" onsubmit={(event) => { event.preventDefault(); load(1); }}>
		<input class="min-w-64 rounded-md border border-neutral-300 px-3 py-2" placeholder="Search event or organizer" bind:value={search} />
		<select class="rounded-md border border-neutral-300 px-3 py-2" bind:value={status}><option value="">All statuses</option><option>draft</option><option>published</option><option>cancelled</option><option>archived</option></select>
		<Button variant="outline" type="submit">Search</Button>
	</form>
	<div class="overflow-x-auto rounded-lg border border-neutral-200 bg-surface">
		<table class="min-w-full text-sm">
			<thead class="bg-neutral-50 text-left text-neutral-500"><tr><th class="p-3">Event</th><th class="p-3">Organizer</th><th class="p-3">Date</th><th class="p-3">Status</th><th class="p-3">Responses</th><th class="p-3">Attending</th><th class="p-3">Actions</th></tr></thead>
			<tbody>
				{#if loading}<tr><td colspan="7" class="p-8 text-center">Loading…</td></tr>{/if}
				{#each events as event}
					<tr class="border-t border-neutral-200">
						<td class="p-3 font-medium">{event.title}</td>
						<td class="p-3">{event.organizerName || event.organizerEmail}<div class="text-xs text-neutral-500">{event.organizerEmail}</div></td>
						<td class="p-3">{new Date(event.eventDate).toLocaleString()}</td>
						<td class="p-3">{event.effectiveSuspended ? 'Suspended' : event.status}{#if event.ownerSuspended}<div class="text-xs text-error">Owner suspended</div>{/if}</td>
						<td class="p-3">{event.responses}</td>
						<td class="p-3">{event.headcount}</td>
						<td class="p-3"><div class="flex gap-2"><button class="text-primary" onclick={() => openAction(event.suspendedAt ? 'restore' : 'suspend', event)}>{event.suspendedAt ? 'Restore' : 'Suspend'}</button>{#if $isSuperAdmin}<button class="text-error" onclick={() => openAction('delete', event)}>Delete</button>{/if}</div></td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	<AdminPagination {...pagination} onPage={load} />
</div>

<Modal bind:open={actionOpen} title={actionKind === 'delete' ? 'Delete event permanently' : actionKind === 'suspend' ? 'Suspend event' : 'Restore event'}>
	{#if actionTarget}
		<div class="space-y-4">
			<p class="text-sm text-neutral-600">Event: <strong>{actionTarget.title}</strong></p>
			{#if actionKind === 'delete'}
				<div class="rounded-md border border-error bg-error-light p-3 text-sm text-error">This permanently deletes the event and associated guest data. It cannot be undone.</div>
				<Input name="eventConfirmation" label={`Type ${actionTarget.title} to confirm`} bind:value={confirmation} />
			{/if}
			<Input name="eventActionReason" label="Reason" bind:value={reason} required />
		</div>
	{/if}
	{#snippet actions()}
		<Button variant="outline" onclick={() => (actionOpen = false)} disabled={submitting}>Cancel</Button>
		<Button variant={actionKind === 'delete' ? 'danger' : 'primary'} onclick={submitAction} loading={submitting} disabled={!actionReady()}>{actionKind === 'delete' ? 'Delete permanently' : 'Confirm'}</Button>
	{/snippet}
</Modal>
