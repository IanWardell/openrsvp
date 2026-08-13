<script lang="ts">
	import { goto } from '$app/navigation';
	import { isLoading, isSuperAdmin } from '$lib/stores/auth';

	let { children } = $props();

	$effect(() => {
		if (!$isLoading && !$isSuperAdmin) {
			goto('/admin');
		}
	});
</script>

{#if !$isLoading && $isSuperAdmin}
	{@render children()}
{/if}
