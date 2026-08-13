<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import AdminPagination from '$lib/components/admin/AdminPagination.svelte';
	import { toast } from '$lib/stores/toast';
	import type { AdminAuditEntry, AdminPage, AdminPagination as Pagination } from '$lib/types';

	let rows = $state<AdminAuditEntry[]>([]);
	let pagination = $state<Pagination>({ page: 1, pageSize: 25, total: 0, totalPages: 0 });

	async function load(page = pagination.page) {
		try {
			const result = await api.get<AdminPage<AdminAuditEntry>>(`/admin/audit-log?page=${page}&pageSize=${pagination.pageSize}`);
			rows = result.data;
			pagination = result.pagination;
		} catch (error: any) {
			toast.error(error.message || 'Failed to load audit log');
		}
	}

	onMount(() => load(1));
</script>

<svelte:head><title>Audit Log — OpenRSVP Admin</title></svelte:head>

<div class="space-y-5">
	<div><h1 class="text-2xl font-bold font-display">Audit log</h1><p class="text-sm text-neutral-500 mt-1">Security-sensitive administrative activity.</p></div>
	<div class="overflow-x-auto rounded-lg border bg-surface">
		<table class="min-w-full text-sm"><thead><tr class="text-left text-neutral-500"><th class="p-3">When</th><th class="p-3">Action</th><th class="p-3">Actor</th><th class="p-3">Target</th><th class="p-3">Reason</th></tr></thead><tbody>
				{#each rows as row}<tr class="border-t"><td class="p-3">{new Date(row.createdAt).toLocaleString()}</td><td class="p-3 font-medium">{row.action}</td><td class="p-3">{row.actorEmail || row.actorId}<div class="text-xs text-neutral-500">{row.actorRole} · {row.actorId}</div></td><td class="p-3">{row.targetType}: {row.targetLabel || row.targetId}{#if row.targetLabel}<div class="text-xs text-neutral-500">{row.targetId}</div>{/if}</td><td class="p-3">{row.reason || '—'}</td></tr>{/each}
		</tbody></table>
	</div>
	<AdminPagination {...pagination} onPage={load} />
</div>
