<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api/client';
	import { toast } from '$lib/stores/toast';
	import type { InviteCard, Event } from '$lib/types';
	import AppShell from '$lib/components/layout/AppShell.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import InviteCardPreview from '$lib/components/invite/InviteCardPreview.svelte';
	import { onMount } from 'svelte';

	const eventId = $derived($page.params.eventId);

	let loading = $state(true);
	let saving = $state(false);
	let saved = $state(false);
	let event: Event | null = $state(null);

	// Template selection
	let selectedTemplate = $state('balloon-party');
	let templatePickerOpen = $state(true);

	// Customization fields
	let heading = $state('You\'re Invited!');
	let body = $state('Join us for a wonderful celebration.');
	let footer = $state('We hope to see you there!');
	let primaryColor = $state('#4F46E5');
	let secondaryColor = $state('#EC4899');
	let font = $state('Inter');
	let qrEnabled = $state(true);
	let qrUrl = $state('');
	let qrCaption = $state('Scan for details');
	let qrForeground = $state('#1c1917');
	let qrBackground = $state('#ffffff');
	let photoFit = $state('cover');
	let photoPositionX = $state(50);
	let photoPositionY = $state(50);
	let photoZoom = $state(1);
	let photoLayoutX = $state(60);
	let photoLayoutY = $state(15);
	let photoSize = $state(35);
	let qrLayoutX = $state(79);
	let qrLayoutY = $state(70);
	let qrSize = $state(17);

	type TextAreaKey = 'eyebrowLeft' | 'eyebrowRight' | 'heading' | 'body' | 'eventDate' | 'eventLocation' | 'footer';
	type EditorElementKey = TextAreaKey | 'photo' | 'qr';
	type TextAreaState = {
		text: string;
		x: number;
		y: number;
		width: number;
		fontSize: number;
		fontFamily: string;
		align: 'left' | 'center' | 'right';
	};

	const textAreaFields: Array<{ key: TextAreaKey; label: string; multiline?: boolean }> = [
		{ key: 'eyebrowLeft', label: 'Top-left label' },
		{ key: 'eyebrowRight', label: 'Top-right label' },
		{ key: 'heading', label: 'Heading' },
		{ key: 'body', label: 'Body text', multiline: true },
		{ key: 'eventDate', label: 'Event date' },
		{ key: 'eventLocation', label: 'Event location' },
		{ key: 'footer', label: 'Footer text', multiline: true }
	];
	let activeElement = $state<EditorElementKey>('heading');

	let textAreas = $state<Record<TextAreaKey, TextAreaState>>({
		eyebrowLeft: { text: 'SAVE THE DATE', x: 7, y: 8, width: 34, fontSize: 8, fontFamily: 'Arial', align: 'left' },
		eyebrowRight: { text: 'FORMAL INVITATION TO FOLLOW', x: 57, y: 8, width: 36, fontSize: 8, fontFamily: 'Arial', align: 'right' },
		heading: { text: "You're Invited!", x: 6, y: 20, width: 52, fontSize: 25, fontFamily: 'Georgia', align: 'center' },
		body: { text: 'Join us for a wonderful celebration.', x: 7, y: 47, width: 50, fontSize: 10, fontFamily: 'Arial', align: 'center' },
		eventDate: { text: '', x: 7, y: 63, width: 50, fontSize: 9, fontFamily: 'Arial', align: 'left' },
		eventLocation: { text: '', x: 7, y: 73, width: 50, fontSize: 9, fontFamily: 'Arial', align: 'left' },
		footer: { text: 'We hope to see you there!', x: 7, y: 83, width: 50, fontSize: 8, fontFamily: 'Georgia', align: 'left' }
	});

	// Background image
	let backgroundImageUrl = $state('');
	let uploading = $state(false);

	const templates = [
		{
			id: 'balloon-party',
			name: 'Balloon Party',
			description: 'Colorful balloons and confetti',
			emoji: '\u{1F388}'
		},
		{
			id: 'confetti',
			name: 'Confetti',
			description: 'Colorful and celebratory',
			emoji: '\u{1F38A}'
		},
		{
			id: 'unicorn-magic',
			name: 'Unicorn Magic',
			description: 'Purple and pink dreamscape',
			emoji: '\u{1F984}'
		},
		{
			id: 'superhero',
			name: 'Superhero',
			description: 'Bold and action-packed',
			emoji: '\u{26A1}'
		},
		{
			id: 'garden-picnic',
			name: 'Garden Picnic',
			description: 'Green and floral',
			emoji: '\u{1F33F}'
		},
		{
			id: 'elegant-affair',
			name: 'Elegant Affair',
			description: 'Refined and sophisticated',
			emoji: '\u{1F48E}'
		},
		{
			id: 'clean-minimal',
			name: 'Clean Minimal',
			description: 'Simple and modern',
			emoji: '\u{25FE}'
		},
		{
			id: 'tropical-vibes',
			name: 'Tropical Vibes',
			description: 'Warm and beachy',
			emoji: '\u{1F334}'
		},
		{
			id: 'vintage-retro',
			name: 'Vintage Retro',
			description: 'Classic sepia tones',
			emoji: '\u{1F4F7}'
		},
		{
			id: 'chalkboard',
			name: 'Chalkboard',
			description: 'Dark and handwritten',
			emoji: '\u{270D}'
		},
		{
			id: 'botanical-save-the-date',
			name: 'Botanical Save the Date',
			description: 'Animated envelope, photo, and QR code',
			emoji: '\u{1F33F}'
		}
	];

	const fontOptions = [
		{ value: 'Inter', label: 'Inter (Modern)' },
		{ value: 'Georgia', label: 'Georgia (Serif)' },
		{ value: 'Courier New', label: 'Courier New (Mono)' },
		{ value: 'Comic Sans MS', label: 'Comic Sans (Fun)' },
		{ value: 'Arial', label: 'Arial (Clean)' }
	];
	const photoFitOptions = [
		{ value: 'cover', label: 'Fill circle (crop)' },
		{ value: 'contain', label: 'Show full image' }
	];
	const selectedTemplateDetails = $derived(
		templates.find((template) => template.id === selectedTemplate) || templates[0]
	);
	const activeTextArea = $derived(
		textAreaFields.some((area) => area.key === activeElement) ? activeElement as TextAreaKey : null
	);
	const activeTextAreaField = $derived(
		textAreaFields.find((area) => area.key === activeTextArea) || textAreaFields[0]
	);

	const previewHeading = $derived(selectedTemplate === 'botanical-save-the-date' ? textAreas.heading.text : heading);
	const previewBody = $derived(selectedTemplate === 'botanical-save-the-date' ? textAreas.body.text : body);
	const previewFooter = $derived(selectedTemplate === 'botanical-save-the-date' ? textAreas.footer.text : footer);
	const customTextAreas = $derived(Object.fromEntries(
		textAreaFields.map(({ key }) => [key, textAreas[key]])
	));

	const customDataJSON = $derived(JSON.stringify({
		...(backgroundImageUrl ? { backgroundImage: backgroundImageUrl } : {}),
		...(selectedTemplate === 'botanical-save-the-date'
			? {
					textAreas: customTextAreas,
					photo: {
						fit: photoFit,
						x: photoPositionX,
						y: photoPositionY,
						zoom: photoZoom,
						layoutX: photoLayoutX,
						layoutY: photoLayoutY,
						size: photoSize
					},
					qr: {
						enabled: qrEnabled,
						...(qrUrl.trim() ? { url: qrUrl.trim() } : {}),
						caption: qrCaption.trim(),
						foreground: qrForeground,
						background: qrBackground,
						x: qrLayoutX,
						y: qrLayoutY,
						size: qrSize
					}
				}
			: {})
	}));

	onMount(async () => {
		try {
			const [eventResult, inviteResult] = await Promise.all([
				api.get<{ data: Event }>(`/events/${eventId}`),
				api.get<{ data: InviteCard }>(`/invite/event/${eventId}`).catch(() => null)
			]);
			event = eventResult.data;

			if (inviteResult) {
				const invite = inviteResult.data;
				selectedTemplate = invite.templateId || 'balloon-party';
				heading = invite.heading || heading;
				body = invite.body || body;
				footer = invite.footer || footer;
				primaryColor = invite.primaryColor || primaryColor;
				secondaryColor = invite.secondaryColor || secondaryColor;
				font = invite.font || font;
				templatePickerOpen = false;
				textAreas.heading.text = invite.heading || textAreas.heading.text;
				textAreas.body.text = invite.body || textAreas.body.text;
				textAreas.footer.text = invite.footer || textAreas.footer.text;
				textAreas.eventDate.text = eventResult.data.eventDate ? formatDateForEditor(eventResult.data.eventDate) : '';
				textAreas.eventLocation.text = eventResult.data.location || '';

				// Load existing background image from customData
				try {
					const cd = typeof invite.customData === 'string'
						? JSON.parse(invite.customData || '{}')
						: invite.customData || {};
					if (cd.backgroundImage) {
						backgroundImageUrl = cd.backgroundImage;
					}
					if (cd.qr && typeof cd.qr === 'object') {
						qrEnabled = cd.qr.enabled !== false;
						qrUrl = typeof cd.qr.url === 'string' ? cd.qr.url : '';
						qrCaption = typeof cd.qr.caption === 'string' ? cd.qr.caption : qrCaption;
						qrForeground = typeof cd.qr.foreground === 'string' ? cd.qr.foreground : qrForeground;
						qrBackground = typeof cd.qr.background === 'string' ? cd.qr.background : qrBackground;
						qrLayoutX = typeof cd.qr.x === 'number' ? cd.qr.x : qrLayoutX;
						qrLayoutY = typeof cd.qr.y === 'number' ? cd.qr.y : qrLayoutY;
						qrSize = typeof cd.qr.size === 'number' ? cd.qr.size : qrSize;
					}
					if (cd.photo && typeof cd.photo === 'object') {
						photoFit = cd.photo.fit === 'contain' ? 'contain' : photoFit;
						photoPositionX = typeof cd.photo.x === 'number' ? cd.photo.x : photoPositionX;
						photoPositionY = typeof cd.photo.y === 'number' ? cd.photo.y : photoPositionY;
						photoZoom = typeof cd.photo.zoom === 'number' ? cd.photo.zoom : photoZoom;
						photoLayoutX = typeof cd.photo.layoutX === 'number' ? cd.photo.layoutX : photoLayoutX;
						photoLayoutY = typeof cd.photo.layoutY === 'number' ? cd.photo.layoutY : photoLayoutY;
						photoSize = typeof cd.photo.size === 'number' ? cd.photo.size : photoSize;
					}
					if (cd.textAreas && typeof cd.textAreas === 'object') {
						for (const field of textAreaFields) {
							const savedArea = cd.textAreas[field.key];
							if (!savedArea || typeof savedArea !== 'object') continue;
							const target = textAreas[field.key];
							if (typeof savedArea.text === 'string') target.text = savedArea.text;
							if (typeof savedArea.x === 'number') target.x = savedArea.x;
							if (typeof savedArea.y === 'number') target.y = savedArea.y;
							if (typeof savedArea.width === 'number') target.width = savedArea.width;
							if (typeof savedArea.fontSize === 'number') target.fontSize = savedArea.fontSize;
							if (typeof savedArea.fontFamily === 'string') target.fontFamily = savedArea.fontFamily;
							if (savedArea.align === 'left' || savedArea.align === 'center' || savedArea.align === 'right') target.align = savedArea.align;
						}
					}
				} catch {
					// ignore parse errors
				}
			}
		} catch (err: unknown) {
			const apiErr = err as { message?: string };
			toast.error(apiErr.message || 'Failed to load invite data');
		} finally {
			loading = false;
		}
	});

	async function handleSave() {
		saving = true;
		try {
			await api.put(`/invite/event/${eventId}`, {
				templateId: selectedTemplate,
				heading: previewHeading,
				body: previewBody,
				footer: previewFooter,
				primaryColor,
				secondaryColor,
				font,
				customData: customDataJSON
			});
			saved = true;
			toast.success('Invite design saved!');
		} catch (err: unknown) {
			const apiErr = err as { message?: string };
			toast.error(apiErr.message || 'Failed to save invite design');
		} finally {
			saving = false;
		}
	}

	function formatDateForEditor(dateStr: string): string {
		if (!dateStr) return '';
		const date = new Date(dateStr);
		return Number.isNaN(date.getTime()) ? dateStr : date.toLocaleDateString('en-US', {
			weekday: 'long', year: 'numeric', month: 'long', day: 'numeric'
		});
	}

	function selectTemplate(templateId: string) {
		if (templateId === 'botanical-save-the-date') {
			textAreas.heading.text = heading || textAreas.heading.text;
			textAreas.body.text = body || textAreas.body.text;
			textAreas.footer.text = footer || textAreas.footer.text;
			textAreas.eventDate.text ||= event?.eventDate ? formatDateForEditor(event.eventDate) : '';
			textAreas.eventLocation.text ||= event?.location || '';
		} else if (selectedTemplate === 'botanical-save-the-date') {
			heading = textAreas.heading.text;
			body = textAreas.body.text;
			footer = textAreas.footer.text;
		}
		selectedTemplate = templateId;
		templatePickerOpen = false;
	}

	function selectEditorElement(element: EditorElementKey) {
		activeElement = element;
	}

	function moveEditorElement(element: EditorElementKey, x: number, y: number) {
		if (element === 'photo') {
			photoLayoutX = x;
			photoLayoutY = y;
		} else if (element === 'qr') {
			qrLayoutX = x;
			qrLayoutY = y;
		} else {
			textAreas[element].x = x;
			textAreas[element].y = y;
		}
	}

	async function handleImageUpload(e: globalThis.Event & { currentTarget: HTMLInputElement }) {
		const file = (e.currentTarget as HTMLInputElement).files?.[0];
		if (!file) return;

		if (file.size > 2 * 1024 * 1024) {
			toast.error('Image must be under 2MB');
			return;
		}

		uploading = true;
		try {
			const result = await api.upload<{ data: { url: string } }>(`/invite/event/${eventId}/image`, file);
			backgroundImageUrl = result.data.url;
			toast.success('Background image uploaded!');
		} catch (err: unknown) {
			const apiErr = err as { message?: string };
			toast.error(apiErr.message || 'Failed to upload image');
		} finally {
			uploading = false;
		}
	}

	let publishing = $state(false);

	async function publishAndGo(destination: 'share' | 'dashboard') {
		if (event && event.status === 'draft') {
			publishing = true;
			try {
				const result = await api.post<{ data: Event }>(`/events/${eventId}/publish`);
				event = result.data;
				toast.success('Event published!');
			} catch (err: unknown) {
				const apiErr = err as { message?: string };
				toast.error(apiErr.message || 'Failed to publish event');
				publishing = false;
				return;
			}
			publishing = false;
		}
		if (destination === 'share') {
			goto(`/events/${eventId}/share`);
		} else {
			goto(`/events/${eventId}`);
		}
	}

	function removeBackgroundImage() {
		backgroundImageUrl = '';
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		const file = e.dataTransfer?.files?.[0];
		if (!file || !file.type.startsWith('image/')) return;
		// Trigger the same upload flow
		const dt = new DataTransfer();
		dt.items.add(file);
		const input = document.getElementById('bg-image-input') as HTMLInputElement;
		if (input) {
			input.files = dt.files;
			input.dispatchEvent(new window.Event('change', { bubbles: true }));
		}
	}
</script>

<svelte:head>
	<title>Invite Designer -- OpenRSVP</title>
</svelte:head>

<AppShell>
	<div class="mb-6">
		<a href="/events/{eventId}" class="text-sm text-primary hover:text-primary-hover">&larr; Back to event</a>
		<h1 class="mt-2 text-2xl font-bold font-display text-neutral-900">Invite Designer</h1>
		{#if event}
			<p class="text-sm text-neutral-500">{event.title}</p>
		{/if}
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-16">
			<Spinner size="lg" class="text-primary" />
		</div>
	{:else}
		<div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
			<!-- Left: Template picker + customization -->
			<div class="space-y-6">
				<!-- Template picker -->
				<Card>
					{#snippet header()}
						<button
							type="button"
							class="flex w-full items-center justify-between gap-4 text-left"
							onclick={() => (templatePickerOpen = !templatePickerOpen)}
							aria-expanded={templatePickerOpen}
						>
							<span>
								<span class="block text-lg font-semibold font-display text-neutral-900">
									{templatePickerOpen ? 'Choose a Template' : 'Template'}
								</span>
								{#if !templatePickerOpen}
									<span class="mt-1 flex items-center gap-2 text-sm text-neutral-600">
										<span aria-hidden="true">{selectedTemplateDetails.emoji}</span>
										{selectedTemplateDetails.name}
									</span>
								{/if}
							</span>
							<span class="inline-flex items-center gap-2 text-xs font-semibold text-primary">
								{templatePickerOpen ? 'Hide templates' : 'Change template'}
								<svg class="h-4 w-4 transition-transform {templatePickerOpen ? 'rotate-180' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2" aria-hidden="true">
									<path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
								</svg>
							</span>
						</button>
					{/snippet}

					{#if templatePickerOpen}
						<div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
							{#each templates as template}
								<button
									type="button"
									class="relative rounded-lg border-2 p-3 text-center transition-all {selectedTemplate === template.id
										? 'border-primary bg-primary-lighter ring-2 ring-primary-light'
										: 'border-neutral-200 hover:border-primary-light hover:bg-neutral-50'}"
									onclick={() => selectTemplate(template.id)}
								>
									<div class="text-2xl mb-1">{template.emoji}</div>
									<p class="text-xs font-medium text-neutral-900">{template.name}</p>
									<p class="text-xs text-neutral-500 mt-0.5">{template.description}</p>
								</button>
							{/each}
						</div>
					{:else}
						<div class="flex items-center gap-3 rounded-lg bg-neutral-50 px-4 py-3">
							<span class="text-2xl" aria-hidden="true">{selectedTemplateDetails.emoji}</span>
							<div class="min-w-0">
								<p class="text-sm font-semibold text-neutral-900">{selectedTemplateDetails.name}</p>
								<p class="text-xs text-neutral-500">{selectedTemplateDetails.description}</p>
							</div>
						</div>
					{/if}
				</Card>

				<!-- Customization form -->
				<Card>
					{#snippet header()}
						<h2 class="text-lg font-semibold font-display text-neutral-900">Customize</h2>
					{/snippet}

					<div class="space-y-4">
						{#if selectedTemplate === 'botanical-save-the-date'}
							<div class="rounded-lg border border-neutral-200 bg-neutral-50 p-4 space-y-4">
								<div>
									<p class="text-sm font-semibold text-neutral-900">Selected element</p>
									<p class="text-xs text-neutral-500">Click an object on the preview or choose it here. Drag selected objects directly on the card.</p>
								</div>
								<div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
									{#each textAreaFields as area}
										<button
											type="button"
										class="rounded-md border px-3 py-2 text-left text-xs font-medium transition-colors {activeElement === area.key
											? 'border-primary bg-primary-lighter text-primary'
											: 'border-neutral-200 bg-surface text-neutral-600 hover:border-primary-light'}"
										onclick={() => selectEditorElement(area.key)}
										aria-pressed={activeElement === area.key}
										>
										{area.label}
									</button>
									{/each}
									{#each [{ key: 'photo', label: 'Photo' }, { key: 'qr', label: 'QR code' }] as object}
										<button
											type="button"
											class="rounded-md border px-3 py-2 text-left text-xs font-medium transition-colors {activeElement === object.key
												? 'border-primary bg-primary-lighter text-primary'
												: 'border-neutral-200 bg-surface text-neutral-600 hover:border-primary-light'}"
											onclick={() => selectEditorElement(object.key as EditorElementKey)}
											aria-pressed={activeElement === object.key}
										>{object.label}</button>
									{/each}
								</div>

								{#if activeTextArea}
									<div class="rounded-lg border border-neutral-200 bg-surface p-4 space-y-4">
									{#if activeTextAreaField.multiline}
										<Textarea label={activeTextAreaField.label} name={`text-${activeTextArea}`} bind:value={textAreas[activeTextArea].text} rows={2} />
									{:else}
										<Input label={activeTextAreaField.label} name={`text-${activeTextArea}`} bind:value={textAreas[activeTextArea].text} />
									{/if}

									<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
										<Select label="Font" name={`font-${activeTextArea}`} bind:value={textAreas[activeTextArea].fontFamily} options={fontOptions} />
										<div class="space-y-1">
											<label for={`size-${activeTextArea}`} class="block text-xs font-medium text-neutral-600">Font size: {Math.round(textAreas[activeTextArea].fontSize)}px</label>
											<input id={`size-${activeTextArea}`} type="range" min="8" max="64" step="1" bind:value={textAreas[activeTextArea].fontSize} class="w-full accent-primary" />
										</div>
									</div>

									<div class="space-y-2">
										<p class="text-xs font-medium text-neutral-600">Text alignment</p>
										<div class="grid grid-cols-3 rounded-md border border-neutral-200 bg-neutral-50 p-1">
											{#each ['left', 'center', 'right'] as alignment}
												<button
													type="button"
													class="rounded px-2 py-1.5 text-xs font-medium capitalize transition-colors {textAreas[activeTextArea].align === alignment ? 'bg-surface text-primary shadow-sm' : 'text-neutral-500 hover:text-neutral-800'}"
													onclick={() => (textAreas[activeTextArea].align = alignment as TextAreaState['align'])}
												>
													{alignment}
												</button>
											{/each}
										</div>
									</div>

									<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
										<div class="space-y-1">
											<label for={`x-${activeTextArea}`} class="block text-xs font-medium text-neutral-600">Horizontal: {Math.round(textAreas[activeTextArea].x)}%</label>
											<input id={`x-${activeTextArea}`} type="range" min="0" max="90" step="1" bind:value={textAreas[activeTextArea].x} class="w-full accent-primary" />
										</div>
										<div class="space-y-1">
											<label for={`y-${activeTextArea}`} class="block text-xs font-medium text-neutral-600">Vertical: {Math.round(textAreas[activeTextArea].y)}%</label>
											<input id={`y-${activeTextArea}`} type="range" min="0" max="92" step="1" bind:value={textAreas[activeTextArea].y} class="w-full accent-primary" />
										</div>
									</div>
									<div class="space-y-1">
										<label for={`width-${activeTextArea}`} class="block text-xs font-medium text-neutral-600">Text box width: {Math.round(textAreas[activeTextArea].width)}%</label>
										<input id={`width-${activeTextArea}`} type="range" min="10" max="90" step="1" bind:value={textAreas[activeTextArea].width} class="w-full accent-primary" />
									</div>
									</div>
								{:else if activeElement === 'photo'}
									<div class="rounded-lg border border-primary-light bg-surface p-4 space-y-3">
										<div>
											<p class="text-sm font-semibold text-neutral-900">Photo position</p>
											<p class="text-xs text-neutral-500">Drag the circular photo on the preview or fine-tune it here.</p>
										</div>
										<div class="grid grid-cols-2 gap-3">
											<div class="space-y-1"><label for="photoLayoutX" class="block text-xs font-medium text-neutral-600">Horizontal: {Math.round(photoLayoutX)}%</label><input id="photoLayoutX" type="range" min="0" max={100 - photoSize} step="1" bind:value={photoLayoutX} class="w-full accent-primary" /></div>
											<div class="space-y-1"><label for="photoLayoutY" class="block text-xs font-medium text-neutral-600">Vertical: {Math.round(photoLayoutY)}%</label><input id="photoLayoutY" type="range" min="0" max={100 - photoSize} step="1" bind:value={photoLayoutY} class="w-full accent-primary" /></div>
										</div>
										<div class="space-y-1"><label for="photoSize" class="block text-xs font-medium text-neutral-600">Size: {Math.round(photoSize)}%</label><input id="photoSize" type="range" min="18" max="50" step="1" bind:value={photoSize} class="w-full accent-primary" /></div>
									</div>
								{:else}
									<div class="rounded-lg border border-primary-light bg-surface p-4 space-y-3">
										<div><p class="text-sm font-semibold text-neutral-900">QR code position</p><p class="text-xs text-neutral-500">Drag the QR block on the preview or fine-tune it here.</p></div>
										<div class="grid grid-cols-2 gap-3">
											<div class="space-y-1"><label for="qrLayoutX" class="block text-xs font-medium text-neutral-600">Horizontal: {Math.round(qrLayoutX)}%</label><input id="qrLayoutX" type="range" min="0" max={100 - qrSize} step="1" bind:value={qrLayoutX} class="w-full accent-primary" /></div>
											<div class="space-y-1"><label for="qrLayoutY" class="block text-xs font-medium text-neutral-600">Vertical: {Math.round(qrLayoutY)}%</label><input id="qrLayoutY" type="range" min="0" max={100 - qrSize} step="1" bind:value={qrLayoutY} class="w-full accent-primary" /></div>
										</div>
										<div class="space-y-1"><label for="qrSize" class="block text-xs font-medium text-neutral-600">Size: {Math.round(qrSize)}%</label><input id="qrSize" type="range" min="10" max="28" step="1" bind:value={qrSize} class="w-full accent-primary" /></div>
									</div>
								{/if}
							</div>
						{:else}
							<Input label="Heading" name="heading" bind:value={heading} placeholder="You're Invited!" />

							<Textarea
								label="Body Text"
								name="body"
								bind:value={body}
								placeholder="Join us for a wonderful celebration..."
								rows={3}
							/>

							<Input label="Footer Text" name="footer" bind:value={footer} placeholder="We hope to see you there!" />
						{/if}

						<div class="grid grid-cols-2 gap-4">
							<div class="space-y-1">
								<label for="primaryColor" class="block text-sm font-medium text-neutral-700">Primary Color</label>
								<input
									id="primaryColor"
									type="color"
									bind:value={primaryColor}
									class="h-10 w-full rounded-lg border border-neutral-300 cursor-pointer"
								/>
							</div>
							<div class="space-y-1">
								<label for="secondaryColor" class="block text-sm font-medium text-neutral-700">Secondary Color</label>
								<input
									id="secondaryColor"
									type="color"
									bind:value={secondaryColor}
									class="h-10 w-full rounded-lg border border-neutral-300 cursor-pointer"
								/>
							</div>
						</div>

						<Select label="Font" name="font" bind:value={font} options={fontOptions} />

						{#if selectedTemplate === 'botanical-save-the-date' && activeElement === 'qr'}
							<div class="rounded-lg border border-neutral-200 bg-neutral-50 p-4 space-y-3">
								<div class="flex items-center justify-between gap-3">
									<div>
										<p class="text-sm font-medium text-neutral-700">QR code</p>
										<p class="text-xs text-neutral-500">By default it opens this event’s invitation.</p>
									</div>
									<label class="inline-flex items-center gap-2 text-sm text-neutral-700">
										<input type="checkbox" bind:checked={qrEnabled} class="h-4 w-4 rounded border-neutral-300 text-primary focus:ring-primary" />
										Show QR
									</label>
								</div>
								<Input label="Custom QR URL (optional)" name="qrUrl" bind:value={qrUrl} placeholder="https://example.com/details" />
								<Input label="QR caption" name="qrCaption" bind:value={qrCaption} placeholder="Scan for details" />
								<div class="grid grid-cols-2 gap-3">
									<div class="space-y-1">
										<label for="qrForeground" class="block text-xs font-medium text-neutral-600">QR ink</label>
										<input id="qrForeground" type="color" bind:value={qrForeground} class="h-9 w-full rounded border border-neutral-300 cursor-pointer" />
									</div>
									<div class="space-y-1">
										<label for="qrBackground" class="block text-xs font-medium text-neutral-600">QR paper</label>
										<input id="qrBackground" type="color" bind:value={qrBackground} class="h-9 w-full rounded border border-neutral-300 cursor-pointer" />
									</div>
								</div>
							</div>
						{/if}

						<!-- Background Image Upload -->
						{#if selectedTemplate !== 'botanical-save-the-date' || activeElement === 'photo'}
						<div class="space-y-2">
							<label for="bg-image-input" class="block text-sm font-medium text-neutral-700">
								{selectedTemplate === 'botanical-save-the-date' ? 'Feature photo' : 'Background Image'}
							</label>
							{#if selectedTemplate === 'botanical-save-the-date'}
								<p class="text-xs text-neutral-500">Shown as the circular artwork on the invitation card.</p>
							{/if}
							{#if backgroundImageUrl}
								<div class="relative rounded-lg border border-neutral-200 overflow-hidden">
									<img src={backgroundImageUrl} alt="Background preview" class="w-full h-24 object-cover" />
									<button
										type="button"
										onclick={removeBackgroundImage}
										class="absolute top-1 right-1 rounded-full bg-surface/90 p-1 text-neutral-500 hover:text-error shadow-sm transition-colors"
										aria-label="Remove background image"
									>
										<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
											<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
										</svg>
									</button>
								</div>
							{:else}
								<!-- svelte-ignore a11y_no_static_element_interactions -->
								<div
									class="rounded-lg border-2 border-dashed border-neutral-300 p-4 text-center hover:border-primary-light transition-colors cursor-pointer"
									ondragover={(e) => e.preventDefault()}
									ondrop={handleDrop}
								>
									<input
										id="bg-image-input"
										type="file"
										accept="image/jpeg,image/png,image/webp"
										class="hidden"
										onchange={handleImageUpload}
									/>
									<label for="bg-image-input" class="cursor-pointer">
										{#if uploading}
											<div class="flex items-center justify-center gap-2 text-primary">
												<Spinner size="sm" />
												<span class="text-sm">Uploading...</span>
											</div>
										{:else}
											<svg class="w-8 h-8 mx-auto text-neutral-400 mb-1" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
												<path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909M3.75 21h16.5A2.25 2.25 0 0022.5 18.75V5.25A2.25 2.25 0 0020.25 3H3.75A2.25 2.25 0 001.5 5.25v13.5A2.25 2.25 0 003.75 21z" />
											</svg>
											<p class="text-xs text-neutral-500">Drop an image or click to upload</p>
											<p class="text-xs text-neutral-400 mt-0.5">JPEG, PNG, or WebP (max 2MB)</p>
										{/if}
									</label>
								</div>
							{/if}
						</div>
						{/if}

						{#if selectedTemplate === 'botanical-save-the-date' && activeElement === 'photo' && backgroundImageUrl}
							<div class="rounded-lg border border-neutral-200 bg-neutral-50 p-4 space-y-3">
								<div>
									<p class="text-sm font-medium text-neutral-700">Circular photo crop</p>
									<p class="text-xs text-neutral-500">Choose whether to fill the circle and adjust the focal point and zoom.</p>
								</div>
								<Select label="Crop style" name="photoFit" bind:value={photoFit} options={photoFitOptions} />
								<div class="grid grid-cols-2 gap-3">
									<div class="space-y-1">
										<label for="photoPositionX" class="block text-xs font-medium text-neutral-600">Focal point X: {Math.round(photoPositionX)}%</label>
										<input id="photoPositionX" type="range" min="0" max="100" step="1" bind:value={photoPositionX} class="w-full accent-primary" />
									</div>
									<div class="space-y-1">
										<label for="photoPositionY" class="block text-xs font-medium text-neutral-600">Focal point Y: {Math.round(photoPositionY)}%</label>
										<input id="photoPositionY" type="range" min="0" max="100" step="1" bind:value={photoPositionY} class="w-full accent-primary" />
									</div>
								</div>
								<div class="space-y-1">
									<label for="photoZoom" class="block text-xs font-medium text-neutral-600">Zoom: {Number(photoZoom).toFixed(2)}x</label>
									<input id="photoZoom" type="range" min="1" max="2" step="0.05" bind:value={photoZoom} class="w-full accent-primary" />
								</div>
							</div>
						{/if}
					</div>
				</Card>

				<Button onclick={handleSave} loading={saving} class="w-full">Save Invite Design</Button>

				{#if saved}
					<Card class="mt-4">
						<div class="text-center space-y-3">
							<div class="flex items-center justify-center gap-2 text-success">
								<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
								</svg>
								<span class="text-sm font-medium">Design saved</span>
							</div>
							<p class="text-xs text-neutral-500">What would you like to do next?</p>
							<div class="flex flex-col gap-2">
								{#if event && event.status === 'draft'}
									<Button onclick={() => publishAndGo('share')} loading={publishing} variant="primary" size="sm" class="w-full">Publish & Share</Button>
									<Button onclick={() => publishAndGo('dashboard')} variant="outline" size="sm" class="w-full">Publish & View Dashboard</Button>
								{:else}
									<Button href="/events/{eventId}/share" variant="primary" size="sm" class="w-full">Share & Get QR Code</Button>
									<Button href="/events/{eventId}" variant="outline" size="sm" class="w-full">View Event Dashboard</Button>
								{/if}
							</div>
						</div>
					</Card>
				{/if}
			</div>

			<!-- Right: Live preview -->
			<div>
				<div class="sticky top-8">
					<h2 class="text-lg font-semibold font-display text-neutral-900 mb-4">Preview</h2>
					<InviteCardPreview
						templateId={selectedTemplate}
						heading={previewHeading}
						body={previewBody}
						footer={previewFooter}
						{primaryColor}
						{secondaryColor}
						{font}
						eventTitle={event?.title || ''}
						eventDate={event?.eventDate || ''}
						eventLocation={event?.location || ''}
						customData={customDataJSON}
						qrTargetUrl={$page.url.origin + '/i/' + (event?.shareToken || '')}
						timezone={event?.timezone}
						editable={selectedTemplate === 'botanical-save-the-date'}
						selectedElement={activeElement}
						onselectelement={selectEditorElement}
						onmoveelement={moveEditorElement}
					/>
				</div>
			</div>
		</div>
	{/if}
</AppShell>
