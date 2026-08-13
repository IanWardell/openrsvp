<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import { currentUser, isSuperAdmin } from '$lib/stores/auth';
	import AdminPagination from '$lib/components/admin/AdminPagination.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Modal from '$lib/components/ui/Modal.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import { toast } from '$lib/stores/toast';
	import type { AdminPage, AdminPagination as Pagination, AdminUser, Organizer } from '$lib/types';

	type UserAction = 'suspend' | 'restore' | 'magic-link' | 'revoke' | 'role' | 'delete';
	const allRoles: Array<{ value: Organizer['role']; label: string }> = [
		{ value: 'organizer', label: 'Organizer' },
		{ value: 'admin', label: 'Admin' },
		{ value: 'super_admin', label: 'Super admin' }
	];

	let users = $state<AdminUser[]>([]);
	let search = $state('');
	let loading = $state(true);
	let submitting = $state(false);
	let pagination = $state<Pagination>({ page: 1, pageSize: 25, total: 0, totalPages: 0 });

	let inviteOpen = $state(false);
	let inviteEmail = $state('');
	let inviteName = $state('');
	let actionOpen = $state(false);
	let actionKind = $state<UserAction>('suspend');
	let actionTarget = $state<AdminUser | null>(null);
	let actionReason = $state('');
	let actionConfirmation = $state('');
	let actionRole = $state<Organizer['role']>('organizer');
	let roleOptions = $state(allRoles);

	async function load(page = pagination.page) {
		loading = true;
		try {
			const result = await api.get<AdminPage<AdminUser>>(
				`/admin/users?search=${encodeURIComponent(search)}&page=${page}&pageSize=${pagination.pageSize}`
			);
			users = result.data;
			pagination = result.pagination;
		} catch (error: any) {
			toast.error(error.message || 'Failed to load users');
		} finally {
			loading = false;
		}
	}

	function roleRank(role: Organizer['role']) {
		return role === 'super_admin' ? 2 : role === 'admin' ? 1 : 0;
	}

	function canModerate(user: AdminUser) {
		if (user.id === $currentUser?.id) return false;
		if (user.role === 'super_admin' && (!$isSuperAdmin || user.roleManagedByEnvironment)) return false;
		return true;
	}

	function canChangeRole(user: AdminUser) {
		return $isSuperAdmin && user.id !== $currentUser?.id && !(user.roleManagedByEnvironment && user.minimumRole === 'super_admin');
	}

	function canDelete(user: AdminUser) {
		return $isSuperAdmin && user.id !== $currentUser?.id && !user.roleManagedByEnvironment;
	}

	function protectedReason(user: AdminUser) {
		if (user.id === $currentUser?.id) return 'Current account';
		if (user.roleManagedByEnvironment && user.role === 'super_admin') return 'Deployment protected';
		if (!$isSuperAdmin && user.role === 'super_admin') return 'Super-admin protected';
		return '';
	}

	function openAction(kind: UserAction, user: AdminUser) {
		actionKind = kind;
		actionTarget = user;
		actionReason = '';
		actionConfirmation = '';
		actionRole = user.storedRole;
		roleOptions = allRoles.filter((option) => roleRank(option.value) >= roleRank(user.minimumRole));
		actionOpen = true;
	}

	function actionTitle() {
		const titles: Record<UserAction, string> = {
			suspend: 'Suspend organizer',
			restore: 'Restore organizer',
			'magic-link': 'Send sign-in link',
			revoke: 'Revoke all sessions',
			role: 'Change organizer role',
			delete: 'Delete organizer permanently'
		};
		return titles[actionKind];
	}

	function actionReady() {
		if (!actionTarget) return false;
		if (actionKind === 'suspend' || actionKind === 'restore' || actionKind === 'role') return actionReason.trim().length > 0;
		if (actionKind === 'delete') return actionReason.trim().length > 0 && actionConfirmation === actionTarget.email;
		return true;
	}

	async function submitAction() {
		if (!actionTarget || !actionReady()) return;
		submitting = true;
		try {
			switch (actionKind) {
				case 'suspend':
					await api.post(`/admin/users/${actionTarget.id}/suspend`, { reason: actionReason.trim() });
					break;
				case 'restore':
					await api.post(`/admin/users/${actionTarget.id}/restore`, { reason: actionReason.trim() });
					break;
				case 'magic-link':
					await api.post(`/admin/users/${actionTarget.id}/magic-link`);
					break;
				case 'revoke':
					await api.post(`/admin/users/${actionTarget.id}/sessions/revoke`);
					break;
				case 'role':
					await api.patch(`/admin/users/${actionTarget.id}/role`, { role: actionRole, reason: actionReason.trim() });
					break;
				case 'delete':
					await api.delete(`/admin/users/${actionTarget.id}`, { confirmation: actionConfirmation, reason: actionReason.trim() });
					break;
			}
			toast.success(actionKind === 'magic-link' ? 'Magic link sent' : 'Organizer updated');
			actionOpen = false;
			await load();
		} catch (error: any) {
			toast.error(error.message || 'Action failed');
		} finally {
			submitting = false;
		}
	}

	async function invite() {
		if (!inviteEmail.trim()) return;
		submitting = true;
		try {
			await api.post('/admin/users', { email: inviteEmail.trim(), name: inviteName.trim() });
			toast.success('Organizer invited');
			inviteOpen = false;
			inviteEmail = '';
			inviteName = '';
			await load(1);
		} catch (error: any) {
			toast.error(error.message || 'Invitation failed');
		} finally {
			submitting = false;
		}
	}

	onMount(() => load(1));
</script>

<svelte:head><title>Users — OpenRSVP Admin</title></svelte:head>

<div class="space-y-5">
	<div class="flex flex-wrap items-end justify-between gap-4">
		<div><h1 class="text-2xl font-bold font-display">Users</h1><p class="text-sm text-neutral-500 mt-1">Platform accounts and moderation state.</p></div>
		{#if $isSuperAdmin}<Button onclick={() => (inviteOpen = true)}>Invite organizer</Button>{/if}
	</div>
	<form class="flex gap-2" onsubmit={(event) => { event.preventDefault(); load(1); }}>
		<input class="w-full max-w-md rounded-md border border-neutral-300 px-3 py-2" placeholder="Search name or email" bind:value={search} />
		<Button variant="outline" type="submit">Search</Button>
	</form>
	<div class="overflow-x-auto rounded-lg border border-neutral-200 bg-surface">
		<table class="min-w-full text-sm">
			<thead class="bg-neutral-50 text-left text-neutral-500"><tr><th class="p-3">User</th><th class="p-3">Role</th><th class="p-3">Events</th><th class="p-3">Status</th><th class="p-3">Actions</th></tr></thead>
			<tbody>
				{#if loading}<tr><td colspan="5" class="p-8 text-center">Loading…</td></tr>{:else if users.length === 0}<tr><td colspan="5" class="p-8 text-center text-neutral-500">No users found.</td></tr>{/if}
				{#each users as user}
					<tr class="border-t border-neutral-200">
						<td class="p-3"><div class="font-medium">{user.name || 'Unnamed organizer'}</div><div class="text-neutral-500">{user.email}</div></td>
						<td class="p-3">{user.role.replace('_', ' ')}{#if user.roleManagedByEnvironment}<div class="text-xs text-primary">Deployment managed</div>{/if}</td>
						<td class="p-3">{user.ownedEvents} owned · {user.cohostedEvents} cohosted</td>
						<td class="p-3">{user.suspendedAt ? 'Suspended' : 'Active'}</td>
						<td class="p-3">
							<div class="flex flex-wrap gap-2">
								{#if canModerate(user)}<button class="text-primary" onclick={() => openAction(user.suspendedAt ? 'restore' : 'suspend', user)}>{user.suspendedAt ? 'Restore' : 'Suspend'}</button>{:else}<span class="text-xs text-neutral-400">{protectedReason(user)}</span>{/if}
								{#if $isSuperAdmin}
									<button class="text-primary" onclick={() => openAction('magic-link', user)}>Magic link</button>
									<button class="text-primary" onclick={() => openAction('revoke', user)}>Revoke sessions</button>
									{#if canChangeRole(user)}<button class="text-primary" onclick={() => openAction('role', user)}>Role</button>{/if}
									{#if canDelete(user)}<button class="text-error" onclick={() => openAction('delete', user)}>Delete</button>{/if}
								{/if}
							</div>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
	<AdminPagination {...pagination} onPage={load} />
</div>

<Modal bind:open={inviteOpen} title="Invite organizer">
	<div class="space-y-4">
		<p class="text-sm text-neutral-600">Create the organizer account and send its first magic sign-in link.</p>
		<Input name="inviteEmail" type="email" label="Email address" bind:value={inviteEmail} required />
		<Input name="inviteName" label="Name (optional)" bind:value={inviteName} />
	</div>
	{#snippet actions()}
		<Button variant="outline" onclick={() => (inviteOpen = false)} disabled={submitting}>Cancel</Button>
		<Button onclick={invite} loading={submitting} disabled={!inviteEmail.trim()}>Invite organizer</Button>
	{/snippet}
</Modal>

<Modal bind:open={actionOpen} title={actionTitle()}>
	{#if actionTarget}
		<div class="space-y-4">
			<p class="text-sm text-neutral-600">Account: <strong>{actionTarget.email}</strong></p>
			{#if actionKind === 'magic-link'}
				<p class="text-sm text-neutral-600">A new one-time sign-in link will be emailed to this organizer.</p>
			{:else if actionKind === 'revoke'}
				<p class="text-sm text-neutral-600">This organizer will be signed out on every device.</p>
			{:else if actionKind === 'role'}
				<Select name="actionRole" label="New role" bind:value={actionRole} options={roleOptions} />
			{:else if actionKind === 'delete'}
				<div class="rounded-md border border-error bg-error-light p-3 text-sm text-error">This permanently deletes the organizer and owned data. It cannot be undone.</div>
				<Input name="actionConfirmation" label={`Type ${actionTarget.email} to confirm`} bind:value={actionConfirmation} />
			{/if}
			{#if actionKind === 'suspend' || actionKind === 'restore' || actionKind === 'role' || actionKind === 'delete'}
				<Input name="actionReason" label="Reason" bind:value={actionReason} required />
			{/if}
		</div>
	{/if}
	{#snippet actions()}
		<Button variant="outline" onclick={() => (actionOpen = false)} disabled={submitting}>Cancel</Button>
		<Button variant={actionKind === 'delete' ? 'danger' : 'primary'} onclick={submitAction} loading={submitting} disabled={!actionReady()}>{actionKind === 'delete' ? 'Delete permanently' : 'Confirm'}</Button>
	{/snippet}
</Modal>
