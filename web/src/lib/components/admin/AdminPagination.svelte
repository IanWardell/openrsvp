<script lang="ts">
	interface Props {
		page: number;
		totalPages: number;
		total: number;
		onPage: (page: number) => void;
	}

	let { page, totalPages, total, onPage }: Props = $props();
</script>

{#if total > 0}
	<div class="flex items-center justify-between gap-4 text-sm text-neutral-500">
		<span>{total} total</span>
		<div class="flex items-center gap-3">
			<button
				type="button"
				class="rounded-md border border-neutral-300 px-3 py-1.5 disabled:opacity-40"
				disabled={page <= 1}
				onclick={() => onPage(page - 1)}>Previous</button
			>
			<span>Page {page} of {Math.max(totalPages, 1)}</span>
			<button
				type="button"
				class="rounded-md border border-neutral-300 px-3 py-1.5 disabled:opacity-40"
				disabled={page >= totalPages}
				onclick={() => onPage(page + 1)}>Next</button
			>
		</div>
	</div>
{/if}
