import { writable, derived } from 'svelte/store';
import type { Organizer } from '$lib/types';

export const currentUser = writable<Organizer | null>(null);
export const isAuthenticated = derived(currentUser, ($user) => $user !== null);
export const isAdmin = derived(currentUser, ($user) => $user?.isAdmin === true);
export const isSuperAdmin = derived(currentUser, ($user) => $user?.isSuperAdmin === true || $user?.role === 'super_admin');
export const isLoading = writable(true);
