<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import { toast } from '$lib/stores/toast';
	import { allowSignups, appConfigLoaded, loadAppConfig } from '$lib/stores/config';
	import { isValidEmail } from '$lib/utils/validation';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';

	let email = $state('');
	let loading = $state(false);
	let sent = $state(false);
	let emailError = $state('');

	onMount(() => {
		// The policy is live-editable by a super admin, so refresh it whenever
		// this entry point is visited.
		loadAppConfig(true);
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		emailError = '';

		if (!email.trim()) {
			emailError = 'Email is required';
			return;
		}

		if (!isValidEmail(email)) {
			emailError = 'Please enter a valid email address';
			return;
		}

		loading = true;
		try {
			await api.post('/auth/magic-link', { email });
			sent = true;
		} catch (err: unknown) {
			const apiErr = err as { message?: string };
			toast.error(apiErr.message || 'Failed to send magic link. Please try again.');
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Sign In -- OpenRSVP</title>
</svelte:head>

<div class="min-h-screen flex items-center justify-center px-4">
	<div class="w-full max-w-md">
		<div class="text-center mb-8">
			<a href="/" class="text-2xl font-bold text-primary">OpenRSVP</a>
			<h1 class="font-display mt-4 text-2xl font-semibold text-neutral-900">
				{$appConfigLoaded && $allowSignups ? 'Sign in or create an account' : 'Organizer sign in'}
			</h1>
			<p class="mt-2 text-neutral-600">
				{$appConfigLoaded && !$allowSignups
					? 'Enter the email for your existing or invited organizer account'
					: 'Enter your email to receive a magic link'}
			</p>
		</div>

		<div class="bg-surface rounded-lg shadow-sm border border-neutral-200 p-8">
			{#if sent}
				<!-- Success state -->
				<div class="text-center">
					<div
						class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-success-light mb-4"
					>
						<svg
							class="h-6 w-6 text-success"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
							/>
						</svg>
					</div>
					<h2 class="text-lg font-semibold text-neutral-900 mb-2">Check your email</h2>
					<p class="text-sm text-neutral-600 mb-4">
						If an account exists for <strong>{email}</strong>, a magic link has been
						sent. Click the link in the email to sign in.
					</p>
					<p class="text-xs text-neutral-500 mb-6">
						Did not receive the email? Check your spam folder or try again.
					</p>
					<Button
						variant="outline"
						onclick={() => {
							sent = false;
							email = '';
						}}
					>
						Try a different email
					</Button>
				</div>
			{:else}
				<!-- Login form -->
				{#if $appConfigLoaded && !$allowSignups}
					<div class="mb-6 rounded-md border border-neutral-200 bg-neutral-50 px-4 py-3 text-sm text-neutral-700">
						<strong class="font-semibold text-neutral-900">Public organizer registration is closed.</strong>
						Only existing organizers and people invited by an administrator will receive a sign-in link.
					</div>
				{/if}
				<form onsubmit={handleSubmit} class="space-y-6">
					<Input
						label="Email address"
						name="email"
						type="email"
						bind:value={email}
						placeholder="you@example.com"
						error={emailError}
						required
					/>

					<Button type="submit" {loading} class="w-full">
						{loading ? 'Sending...' : 'Send Magic Link'}
					</Button>
				</form>

				<div class="mt-6 text-center">
					<a href="/" class="text-sm text-primary hover:text-primary-hover">
						Back to home
					</a>
				</div>
			{/if}
		</div>
	</div>
</div>
