<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import AdminPagination from '$lib/components/admin/AdminPagination.svelte';
	import { toast } from '$lib/stores/toast';
	import type { AdminEvent, AdminPage, AdminPagination as Pagination } from '$lib/types';

	let rows = $state<AdminEvent[]>([]);
	let search = $state('');
	let pagination = $state<Pagination>({ page: 1, pageSize: 25, total: 0, totalPages: 0 });

	async function load(page = pagination.page) {
		try {
			const result = await api.get<AdminPage<AdminEvent>>(
				`/admin/invitations/events?search=${encodeURIComponent(search)}&page=${page}&pageSize=${pagination.pageSize}`
			);
			rows = result.data;
			pagination = result.pagination;
		} catch (error: any) {
			toast.error(error.message || 'Failed to load invitations');
		}
	}

	onMount(() => load(1));
</script>

<svelte:head><title>Event Invitations — OpenRSVP Admin</title></svelte:head>

<div class="space-y-5">
	<div>
		<h1 class="text-2xl font-bold font-display">Event invitations</h1>
		<p class="text-sm text-neutral-500 mt-1">Invite designs, public availability, and aggregate responses.</p>
	</div>
	<form class="flex gap-2" onsubmit={(event) => { event.preventDefault(); load(1); }}>
		<input class="w-full max-w-md rounded-md border border-neutral-300 px-3 py-2" placeholder="Search event or organizer" bind:value={search} />
		<button class="rounded-md border px-4">Search</button>
	</form>
	<div class="grid gap-3">
		{#each rows as row}
			<article class="rounded-lg border border-neutral-200 bg-surface p-4">
				<div class="flex flex-wrap justify-between gap-3">
					<div><h2 class="font-semibold">{row.title}</h2><p class="text-sm text-neutral-500">{row.organizerEmail} · {row.templateId || 'Default design'}</p></div>
					<div class="text-sm">{row.responses} {row.responses === 1 ? 'response' : 'responses'} · {row.headcount} attending headcount</div>
				</div>
				<div class="mt-3 flex gap-3 text-sm">
					{#if row.publicUrl}<a class="text-primary" href={row.publicUrl} target="_blank" rel="noreferrer">Open public invitation</a>{:else}<span class="text-neutral-500">Not publicly available</span>{/if}
					<span>{row.effectiveSuspended ? 'Suspended' : row.status}</span>
				</div>
			</article>
		{/each}
	</div>
	<AdminPagination {...pagination} onPage={load} />
</div>
