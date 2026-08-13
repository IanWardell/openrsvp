<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import AdminPagination from '$lib/components/admin/AdminPagination.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Modal from '$lib/components/ui/Modal.svelte';
	import { toast } from '$lib/stores/toast';
	import type { AdminGuest, AdminPage, AdminPagination as Pagination } from '$lib/types';

	let rows = $state<AdminGuest[]>([]);
	let search = $state('');
	let revealed = $state(false);
	let submitting = $state(false);
	let pagination = $state<Pagination>({ page: 1, pageSize: 25, total: 0, totalPages: 0 });
	let reasonOpen = $state(false);
	let revealTarget = $state<AdminGuest | null>(null);
	let revealReason = $state('');
	let resultsOpen = $state(false);
	let participation = $state<AdminGuest[]>([]);

	async function load(page = pagination.page) {
		revealed = false;
		try {
			const result = await api.get<AdminPage<AdminGuest>>(
				`/admin/invitations/guests?search=${encodeURIComponent(search)}&page=${page}&pageSize=${pagination.pageSize}`
			);
			rows = result.data;
			pagination = result.pagination;
		} catch (error: any) {
			toast.error(error.message || 'Failed to load guests');
		}
	}

	function requestReveal(target: AdminGuest | null) {
		revealTarget = target;
		revealReason = '';
		reasonOpen = true;
	}

	async function reveal() {
		if (!revealReason.trim()) return;
		submitting = true;
		try {
			if (revealTarget) {
				const result = await api.post<{ data: AdminGuest[] }>(`/admin/invitations/guests/${revealTarget.id}/reveal`, { reason: revealReason.trim() });
				participation = result.data;
				resultsOpen = true;
			} else {
				const result = await api.post<AdminPage<AdminGuest>>(
					`/admin/invitations/guests/reveal-page?search=${encodeURIComponent(search)}&page=${pagination.page}&pageSize=${pagination.pageSize}`,
					{ reason: revealReason.trim() }
				);
				rows = result.data;
				pagination = result.pagination;
				revealed = true;
			}
			reasonOpen = false;
		} catch (error: any) {
			toast.error(error.message || 'Reveal failed');
		} finally {
			submitting = false;
		}
	}

	onMount(() => load(1));
</script>

<svelte:head><title>Guest Invitations — OpenRSVP Admin</title></svelte:head>

<div class="space-y-5">
	<div><h1 class="text-2xl font-bold font-display">Guest invitations</h1><p class="text-sm text-neutral-500 mt-1">Search platform participation. Contact data is masked until an audited reveal.</p></div>
	<form class="flex flex-wrap gap-2" onsubmit={(event) => { event.preventDefault(); load(1); }}>
		<input class="min-w-72 rounded-md border border-neutral-300 px-3 py-2" placeholder="Search name, email, or phone" bind:value={search} />
		<Button variant="outline" type="submit">Search</Button>
		<Button type="button" onclick={() => requestReveal(null)}>Reveal current page</Button>
	</form>
	{#if revealed}<div class="rounded-md border border-warning bg-warning-light p-3 text-sm">Sensitive data is revealed for this page only and will be masked on navigation or search.</div>{/if}
	<div class="overflow-x-auto rounded-lg border bg-surface">
		<table class="min-w-full text-sm">
			<thead><tr class="text-left text-neutral-500"><th class="p-3">Guest</th><th class="p-3">Contact</th><th class="p-3">Event</th><th class="p-3">RSVP</th><th class="p-3">Source</th><th class="p-3"></th></tr></thead>
			<tbody>
				{#each rows as row}
					<tr class="border-t">
						<td class="p-3">{row.name}</td>
						<td class="p-3 text-xs text-neutral-500">{#if row.email}<div><span class="sr-only">Email: </span>{row.email}</div>{/if}{#if row.phone}<div><span class="sr-only">Phone: </span>{row.phone}</div>{/if}{#if !row.email && !row.phone}—{/if}</td>
						<td class="p-3">{row.eventTitle}</td><td class="p-3">{row.rsvpStatus} · +{row.plusOnes}</td><td class="p-3">{row.importSource || 'Self-submitted'}</td><td class="p-3"><button class="text-primary" onclick={() => requestReveal(row)}>Reveal participation</button></td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	<AdminPagination {...pagination} onPage={load} />
</div>

<Modal bind:open={reasonOpen} title={revealTarget ? 'Reveal guest participation' : 'Reveal current guest page'}>
	<div class="space-y-4">
		<div class="rounded-md border border-warning bg-warning-light p-3 text-sm">This action exposes personal information and will be recorded in the audit log.</div>
		<Input name="revealReason" label="Reason for access" bind:value={revealReason} required />
	</div>
	{#snippet actions()}
		<Button variant="outline" onclick={() => (reasonOpen = false)} disabled={submitting}>Cancel</Button>
		<Button onclick={reveal} loading={submitting} disabled={!revealReason.trim()}>Reveal</Button>
	{/snippet}
</Modal>

<Modal bind:open={resultsOpen} title="Matching guest participation">
	<div class="space-y-3">
		<p class="text-sm text-neutral-600">Matches may be based on contact information or name. Review possible name-only matches carefully.</p>
		{#each participation as item}
			<div class="rounded-md border border-neutral-200 p-3 text-sm">
				<div class="font-medium">{item.name} — {item.eventTitle}</div>
				{#if item.email}<div class="text-neutral-600">Email: {item.email}</div>{/if}
				{#if item.phone}<div class="text-neutral-600">Phone: {item.phone}</div>{/if}
				<div class="text-neutral-600">RSVP: {item.rsvpStatus} · +{item.plusOnes}</div>
			</div>
		{/each}
	</div>
	{#snippet actions()}<Button onclick={() => (resultsOpen = false)}>Close and re-mask</Button>{/snippet}
</Modal>
