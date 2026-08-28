<script lang="ts">
	import QRCode from 'qrcode';

	interface Props {
		heading: string;
		body: string;
		footer: string;
		primaryColor: string;
		secondaryColor: string;
		font: string;
		eventTitle: string;
		eventDate: string;
		eventLocation: string;
		customData?: string | Record<string, unknown>;
		qrTargetUrl?: string;
		timezone?: string;
		editable?: boolean;
		selectedElement?: EditorElementKey | string;
		onselectelement?: (element: EditorElementKey) => void;
		onmoveelement?: (element: EditorElementKey, x: number, y: number) => void;
	}

	type TextAreaKey =
		| 'eyebrowLeft'
		| 'eyebrowRight'
		| 'heading'
		| 'body'
		| 'eventDate'
		| 'eventLocation'
		| 'footer';
	type EditorElementKey = TextAreaKey | 'photo' | 'qr';

	type ResolvedTextArea = {
		text: string;
		x: number;
		y: number;
		width: number;
		fontSize: number;
		fontFamily: string;
		align: 'left' | 'center' | 'right';
	};

	let {
		heading,
		body,
		footer,
		primaryColor,
		secondaryColor,
		font,
		eventTitle,
		eventDate,
		eventLocation,
		customData = '{}',
		qrTargetUrl = '',
		timezone,
		editable = false,
		selectedElement = '',
		onselectelement,
		onmoveelement
	}: Props = $props();

	let showingReveal = $state(false);
	let playing = $state(false);
	let qrDataUrl = $state('');
	let qrGeneration = 0;

	const parsedCustomData = $derived.by(() => {
		if (typeof customData === 'object' && customData !== null) return customData;
		try {
			return JSON.parse(customData || '{}') as Record<string, unknown>;
		} catch {
			return {};
		}
	});

	const qrConfig = $derived(
		(parsedCustomData.qr as Record<string, unknown> | undefined) || {}
	);
	function safeImageUrl(raw: unknown): string {
		if (typeof raw !== 'string' || !raw.trim() || /[()'"<>\\]/.test(raw)) return '';
		const value = raw.trim();
		if (value.startsWith('/')) return value;
		try {
			const parsed = new URL(value);
			return parsed.protocol === 'http:' || parsed.protocol === 'https:' ? value : '';
		} catch {
			return '';
		}
	}
	const photoUrl = $derived(
		safeImageUrl(parsedCustomData.backgroundImage)
	);
	const cardBackgroundUrl = $derived(safeImageUrl(parsedCustomData.cardBackgroundImage));
	const cardBackgroundConfig = $derived(
		(parsedCustomData.cardBackground as Record<string, unknown> | undefined) || {}
	);
	const envelopeFlapColor = $derived(
		typeof parsedCustomData.envelopeFlapColor === 'string'
			? parsedCustomData.envelopeFlapColor
			: '#aebda9'
	);
	const photoConfig = $derived(
		(parsedCustomData.photo as Record<string, unknown> | undefined) || {}
	);
	const rawTextAreas = $derived(
		(parsedCustomData.textAreas as Record<string, unknown> | undefined) || {}
	);
	const qrEnabled = $derived(qrConfig.enabled !== false);
	const qrCaption = $derived(
		typeof qrConfig.caption === 'string' && qrConfig.caption.trim()
			? qrConfig.caption
			: 'Scan for details'
	);
	const qrUrl = $derived(
		typeof qrConfig.url === 'string' && qrConfig.url.trim() ? qrConfig.url.trim() : qrTargetUrl
	);
	const safeQrUrl = $derived.by(() => {
		try {
			const parsed = new URL(qrUrl);
			return parsed.protocol === 'http:' || parsed.protocol === 'https:' ? qrUrl : '';
		} catch {
			return '';
		}
	});
	const qrForeground = $derived(
		typeof qrConfig.foreground === 'string' ? qrConfig.foreground : '#1c1917'
	);
	const qrBackground = $derived(
		typeof qrConfig.background === 'string' ? qrConfig.background : '#ffffff'
	);
	const photoFit = $derived(photoConfig.fit === 'contain' ? 'contain' : 'cover');
	const photoPositionX = $derived(clampNumber(photoConfig.x, 0, 100, 50));
	const photoPositionY = $derived(clampNumber(photoConfig.y, 0, 100, 50));
	const photoZoom = $derived(clampNumber(photoConfig.zoom, 1, 2, 1));
	const cardBackgroundFit = $derived(cardBackgroundConfig.fit === 'contain' ? 'contain' : 'cover');
	const cardBackgroundX = $derived(clampNumber(cardBackgroundConfig.x, 0, 100, 50));
	const cardBackgroundY = $derived(clampNumber(cardBackgroundConfig.y, 0, 100, 50));
	const cardBackgroundZoom = $derived(clampNumber(cardBackgroundConfig.zoom, 1, 2, 1));
	const photoLayoutX = $derived(clampNumber(photoConfig.layoutX, 0, 82, 60));
	const photoLayoutY = $derived(clampNumber(photoConfig.layoutY, 0, 82, 15));
	const photoSize = $derived(clampNumber(photoConfig.size, 18, 50, 35));
	const qrLayoutX = $derived(clampNumber(qrConfig.x, 0, 90, 79));
	const qrLayoutY = $derived(clampNumber(qrConfig.y, 0, 90, 70));
	const qrSize = $derived(clampNumber(qrConfig.size, 10, 28, 17));

	const textAreas = $derived.by(() => {
		const defaults: Record<TextAreaKey, ResolvedTextArea> = {
			eyebrowLeft: { text: 'SAVE THE DATE', x: 7, y: 8, width: 34, fontSize: 8, fontFamily: 'Arial', align: 'left' },
			eyebrowRight: { text: 'FORMAL INVITATION TO FOLLOW', x: 57, y: 8, width: 36, fontSize: 8, fontFamily: 'Arial', align: 'right' },
			heading: { text: heading || eventTitle || 'Your event title', x: 6, y: 20, width: 52, fontSize: 25, fontFamily: font || 'Georgia', align: 'center' },
			body: { text: body || '', x: 7, y: 47, width: 50, fontSize: 10, fontFamily: 'Arial', align: 'center' },
			eventDate: { text: eventDate ? formatDate(eventDate) : '', x: 7, y: 63, width: 50, fontSize: 9, fontFamily: 'Arial', align: 'left' },
			eventLocation: { text: eventLocation || '', x: 7, y: 73, width: 50, fontSize: 9, fontFamily: 'Arial', align: 'left' },
			footer: { text: footer || '', x: 7, y: 83, width: 50, fontSize: 8, fontFamily: 'Georgia', align: 'left' }
		};

		return Object.fromEntries(
			(Object.keys(defaults) as TextAreaKey[]).map((key) => {
				const raw = (rawTextAreas[key] as Record<string, unknown> | undefined) || {};
				const fallback = defaults[key];
				return [key, {
					text: typeof raw.text === 'string' ? raw.text : fallback.text,
					x: clampNumber(raw.x, 0, 90, fallback.x),
					y: clampNumber(raw.y, 0, 92, fallback.y),
					width: clampNumber(raw.width, 10, 90, fallback.width),
					fontSize: clampNumber(raw.fontSize, 8, 64, fallback.fontSize),
					fontFamily: safeFont(typeof raw.fontFamily === 'string' ? raw.fontFamily : fallback.fontFamily, fallback.fontFamily),
					align: raw.align === 'left' || raw.align === 'right' || raw.align === 'center' ? raw.align : fallback.align
				} satisfies ResolvedTextArea];
			})
		) as Record<TextAreaKey, ResolvedTextArea>;
	});

	function clampNumber(value: unknown, min: number, max: number, fallback: number): number {
		const numeric = typeof value === 'number' ? value : Number(value);
		return Number.isFinite(numeric) ? Math.min(max, Math.max(min, numeric)) : fallback;
	}

	function safeFont(value: string, fallback: string): string {
		const allowed = ['Inter', 'Georgia', 'Courier New', 'Comic Sans MS', 'Arial'];
		return allowed.includes(value) ? value : fallback;
	}

	function formatDate(dateStr: string): string {
		if (!dateStr) return '';
		try {
			const options: Intl.DateTimeFormatOptions = {
				weekday: 'long',
				year: 'numeric',
				month: 'long',
				day: 'numeric'
			};
			if (timezone) options.timeZone = timezone;
			return new Date(dateStr).toLocaleDateString('en-US', options);
		} catch {
			return dateStr;
		}
	}

	function areaStyle(key: TextAreaKey): string {
		const area = textAreas[key];
		return `left: ${area.x}%; top: ${area.y}%; width: ${area.width}%; --area-size: ${area.fontSize}px; --area-font: ${area.fontFamily}; text-align: ${area.align};`;
	}

	function selectElement(element: EditorElementKey) {
		if (editable && !showingReveal) onselectelement?.(element);
	}

	function startElementDrag(event: PointerEvent, element: EditorElementKey) {
		if (!editable || showingReveal || !onmoveelement) return;
		event.preventDefault();
		event.stopPropagation();
		selectElement(element);

		const target = event.currentTarget as HTMLElement;
		const card = target.closest('.botanical-card') as HTMLElement | null;
		if (!card) return;
		const cardRect = card.getBoundingClientRect();
		const targetRect = target.getBoundingClientRect();
		const offsetX = event.clientX - targetRect.left;
		const offsetY = event.clientY - targetRect.top;
		const maxX = Math.max(0, 100 - (targetRect.width / cardRect.width) * 100);
		const maxY = Math.max(0, 100 - (targetRect.height / cardRect.height) * 100);
		target.setPointerCapture(event.pointerId);

		const move = (moveEvent: PointerEvent) => {
			const x = clampNumber(((moveEvent.clientX - cardRect.left - offsetX) / cardRect.width) * 100, 0, maxX, 0);
			const y = clampNumber(((moveEvent.clientY - cardRect.top - offsetY) / cardRect.height) * 100, 0, maxY, 0);
			onmoveelement(element, Math.round(x * 10) / 10, Math.round(y * 10) / 10);
		};
		const end = () => {
			target.removeEventListener('pointermove', move);
			target.removeEventListener('pointerup', end);
			target.removeEventListener('pointercancel', end);
		};
		target.addEventListener('pointermove', move);
		target.addEventListener('pointerup', end);
		target.addEventListener('pointercancel', end);
	}

	async function generateQr(url: string, foreground: string, background: string, enabled: boolean) {
		const generation = ++qrGeneration;
		if (!url || !enabled) {
			qrDataUrl = '';
			return;
		}
		try {
			const dataUrl = await QRCode.toDataURL(url, {
				width: 220,
				margin: 2,
				errorCorrectionLevel: 'M',
				color: { dark: foreground, light: background }
			});
			if (generation === qrGeneration) qrDataUrl = dataUrl;
		} catch {
			qrDataUrl = '';
		}
	}

	function startReveal() {
		playing = false;
		requestAnimationFrame(() => {
			requestAnimationFrame(() => (playing = true));
		});
	}

	function playReveal() {
		showingReveal = true;
		startReveal();
	}

	function returnToCardPreview() {
		playing = false;
		showingReveal = false;
	}

	$effect(() => {
		void generateQr(safeQrUrl, qrForeground, qrBackground, qrEnabled);
	});

</script>

<div
	class="botanical-experience"
	style="--primary: {primaryColor || '#6f8d7b'}; --secondary: {secondaryColor || '#d7c2a4'}; --card-font: {font || 'Georgia, serif'};"
>
	{#snippet invitationCard()}
		<article class="botanical-card" aria-label="Invitation card preview">
			{#if cardBackgroundUrl}
				<div class="card-background" aria-hidden="true">
					<img src={cardBackgroundUrl} alt="" style="--background-fit: {cardBackgroundFit}; --background-x: {cardBackgroundX}%; --background-y: {cardBackgroundY}%; --background-zoom: {cardBackgroundZoom};" />
				</div>
				<div class="card-background-overlay" aria-hidden="true"></div>
			{/if}
			<div class="leaf-cluster leaf-cluster-left" aria-hidden="true">❧</div>
			<div class="leaf-cluster leaf-cluster-right" aria-hidden="true">❧</div>
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div class="card-text-area card-text-eyebrow" class:editor-element={editable && !showingReveal} class:selected={selectedElement === 'eyebrowLeft'} style={areaStyle('eyebrowLeft')} data-editor-element="eyebrowLeft" onpointerdown={(event) => startElementDrag(event, 'eyebrowLeft')}>{textAreas.eyebrowLeft.text}</div>
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div class="card-text-area card-text-eyebrow" class:editor-element={editable && !showingReveal} class:selected={selectedElement === 'eyebrowRight'} style={areaStyle('eyebrowRight')} data-editor-element="eyebrowRight" onpointerdown={(event) => startElementDrag(event, 'eyebrowRight')}>{textAreas.eyebrowRight.text}</div>
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<h1 class="card-text-area card-text-heading" class:editor-element={editable && !showingReveal} class:selected={selectedElement === 'heading'} style={areaStyle('heading')} data-editor-element="heading" onpointerdown={(event) => startElementDrag(event, 'heading')}>{textAreas.heading.text}</h1>
			{#if textAreas.body.text}
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<p class="card-text-area card-text-body" class:editor-element={editable && !showingReveal} class:selected={selectedElement === 'body'} style={areaStyle('body')} data-editor-element="body" onpointerdown={(event) => startElementDrag(event, 'body')}>{textAreas.body.text}</p>
			{/if}
			{#if textAreas.eventDate.text}
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<p class="card-text-area card-text-date" class:editor-element={editable && !showingReveal} class:selected={selectedElement === 'eventDate'} style={areaStyle('eventDate')} data-editor-element="eventDate" onpointerdown={(event) => startElementDrag(event, 'eventDate')}>{textAreas.eventDate.text}</p>
			{/if}
			{#if textAreas.eventLocation.text}
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<p class="card-text-area card-text-location" class:editor-element={editable && !showingReveal} class:selected={selectedElement === 'eventLocation'} style={areaStyle('eventLocation')} data-editor-element="eventLocation" onpointerdown={(event) => startElementDrag(event, 'eventLocation')}>{textAreas.eventLocation.text}</p>
			{/if}
			{#if textAreas.footer.text}
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<p class="card-text-area card-text-footer" class:editor-element={editable && !showingReveal} class:selected={selectedElement === 'footer'} style={areaStyle('footer')} data-editor-element="footer" onpointerdown={(event) => startElementDrag(event, 'footer')}>{textAreas.footer.text}</p>
			{/if}
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div
				class="photo-frame"
				class:editor-element={editable && !showingReveal}
				class:selected={selectedElement === 'photo'}
				style="left: {photoLayoutX}%; top: {photoLayoutY}%; width: {photoSize}%; --photo-fit: {photoFit}; --photo-x: {photoPositionX}%; --photo-y: {photoPositionY}%; --photo-zoom: {photoZoom};"
				data-editor-element="photo"
				onpointerdown={(event) => startElementDrag(event, 'photo')}
			>
				{#if photoUrl}
					<img src={photoUrl} alt="Invitation artwork" />
				{:else}
					<span>Photo</span>
				{/if}
			</div>
			{#if qrEnabled && safeQrUrl}
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div class="qr-block" class:editor-element={editable && !showingReveal} class:selected={selectedElement === 'qr'} style="left: {qrLayoutX}%; top: {qrLayoutY}%; width: {qrSize}%;" data-editor-element="qr" onpointerdown={(event) => startElementDrag(event, 'qr')}>
					{#if qrDataUrl}
						<img src={qrDataUrl} alt="QR code for invitation details" />
					{:else}
						<div class="qr-placeholder" aria-label="Generating QR code"></div>
					{/if}
					<span>{qrCaption}</span>
				</div>
			{/if}
		</article>
	{/snippet}

	{#if showingReveal}
		<div
		class="botanical-stage"
		class:playing
	>
		<div class="woodgrain" aria-hidden="true"></div>
		<div class="envelope-shell" aria-hidden="true">
			<img class="envelope-back-art" src="/invite/botanical/envelope-back.svg" alt="" />
			<div class="envelope-flap-color" style="background-color: {envelopeFlapColor};"></div>
			{@render invitationCard()}
			<img class="envelope-front-art" src="/invite/botanical/envelope-front.svg" alt="" />
			<div class="envelope-paper-grain"></div>
		</div>
		<div class="revealed-card" aria-hidden="true">
			{@render invitationCard()}
		</div>
		<button type="button" class="replay-button" aria-label="Replay invitation reveal" onclick={startReveal}>
			<span aria-hidden="true">▷</span>
			<span class="sr-only">Replay</span>
		</button>
		<button type="button" class="return-button" onclick={returnToCardPreview}>Back to card preview</button>
		</div>
	{:else}
		<div class="card-preview-stage">
			{@render invitationCard()}
		</div>
		<button type="button" class="play-reveal-button" onclick={playReveal}>
			<span aria-hidden="true">▷</span>
			Play envelope animation
		</button>
	{/if}
</div>

<style>
	.botanical-experience {
		width: 100%;
	}

	.botanical-stage {
		position: relative;
		min-height: 38rem;
		isolation: isolate;
		overflow: hidden;
		background: #fefbf6;
		border-radius: 1rem;
	}

	.card-preview-stage {
		position: relative;
		display: grid;
		min-height: 33rem;
		place-items: center;
		overflow: hidden;
		border-radius: 1rem;
		background:
			repeating-linear-gradient(3deg, rgba(120, 105, 87, 0.05) 0 1px, transparent 1px 9px),
			linear-gradient(135deg, #f5f4ef, #ffffff 45%, #f2f0ea);
	}

	.woodgrain {
		position: absolute;
		inset: 0;
		background:
			repeating-linear-gradient(3deg, rgba(120, 105, 87, 0.05) 0 1px, transparent 1px 9px),
			linear-gradient(135deg, #f5f4ef, #ffffff 45%, #f2f0ea);
	}

	.envelope-shell {
		position: absolute;
		left: 50%;
		top: 12%;
		width: min(43.5%, 24rem);
		aspect-ratio: 712.5 / 1000;
		transform: translate(-50%, -28%) scale(1.2);
		transform-origin: center;
		isolation: isolate;
		z-index: 1;
	}

	.envelope-back-art,
	.envelope-front-art,
	.envelope-paper-grain {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
	}

	.envelope-back-art {
		filter: drop-shadow(0 1.25rem 1.5rem rgba(68, 62, 52, 0.18));
		object-fit: contain;
		z-index: 1;
	}

	.envelope-flap-color {
		position: absolute;
		inset: 0;
		clip-path: polygon(0 48.8%, 46.5% 5.8%, 50% 4.8%, 53.5% 5.8%, 100% 48.8%, 93% 50%, 62% 69%, 60% 73.5%, 40% 73.5%, 38% 69%, 7% 50%);
		background-image:
			linear-gradient(100deg, rgba(255, 255, 255, 0.18), transparent 35%),
			url('/invite/botanical/envelope-paper.png');
		background-blend-mode: soft-light, multiply;
		background-size: auto, 13rem;
		box-shadow: inset 0 -0.35rem 0.8rem rgba(0, 0, 0, 0.08);
		z-index: 2;
	}

	.envelope-front-art {
		clip-path: polygon(0 48.8%, 50% 73.5%, 100% 48.8%, 100% 100%, 0 100%);
		object-fit: contain;
		z-index: 4;
	}

	.envelope-paper-grain {
		background: url('/invite/botanical/envelope-paper.png') center / 13rem repeat;
		clip-path: polygon(0 48.8%, 50% 73.5%, 100% 48.8%, 100% 100%, 0 100%);
		mix-blend-mode: multiply;
		opacity: 0.14;
		z-index: 5;
	}

	.botanical-card {
		position: absolute;
		left: 50%;
		top: 50%;
		width: 110%;
		aspect-ratio: 1.4;
		padding: 1.25rem 1.35rem;
		transform: translate(-50%, -48%) scale(0.72);
		transform-origin: center;
		background: #fffefb;
		box-shadow: 0 0.75rem 1.75rem rgba(64, 58, 48, 0.18);
		overflow: hidden;
		z-index: 3;
		opacity: 1;
		isolation: isolate;
		backface-visibility: hidden;
		font-family: var(--card-font);
		color: #263229;
	}

	.card-background,
	.card-background-overlay {
		position: absolute;
		inset: 0;
		z-index: 0;
	}

	.card-background { overflow: hidden; }

	.card-background img {
		width: 100%;
		height: 100%;
		object-fit: var(--background-fit);
		object-position: var(--background-x) var(--background-y);
		transform: scale(var(--background-zoom));
	}

	.card-background-overlay {
		background: rgba(255, 254, 251, 0.28);
	}

	.card-preview-stage .botanical-card {
		position: relative;
		left: auto;
		top: auto;
		width: min(86%, 32rem);
		transform: none;
		z-index: 1;
	}

	.revealed-card {
		position: absolute;
		inset: 0;
		visibility: hidden;
		z-index: 6;
	}

	.revealed-card .botanical-card {
		left: 50%;
		top: 16%;
		width: min(60%, 32rem);
		transform: translate(-50%, 0) scale(0.9);
	}

	.envelope-shell .botanical-card {
		width: 135%;
	}

	.playing .envelope-shell {
		animation: envelope-descend 2.45s 0.08s cubic-bezier(0.65, 0, 0.35, 1) both;
	}

	.playing .envelope-shell .botanical-card {
		opacity: 1;
		will-change: transform, clip-path;
		animation: card-rise 2.45s 0.08s cubic-bezier(0.65, 0, 0.35, 1) both;
	}

	.playing .revealed-card {
		animation: reveal-final-card 2.45s 0.08s steps(1, end) both;
	}

	.playing .revealed-card .botanical-card {
		animation: settle-final-card 2.45s 0.08s cubic-bezier(0.22, 1, 0.36, 1) both;
	}

	.card-text-area {
		position: absolute;
		z-index: 2;
		margin: 0;
		font-family: var(--area-font);
		font-size: clamp(0.45rem, var(--area-size), 3rem);
		line-height: 1.18;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	.card-text-eyebrow {
		letter-spacing: 0.15em;
		line-height: 1.25;
		text-transform: uppercase;
	}

	.card-text-heading {
		font-weight: 500;
		line-height: 1.05;
		color: var(--primary);
		text-wrap: balance;
	}

	.card-text-body { color: #263229; }
	.card-text-date { font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; }
	.card-text-location { color: #59665b; }
	.card-text-footer { color: #68736c; font-style: italic; }

	.photo-frame {
		position: absolute;
		left: 60%;
		top: 15%;
		width: 35%;
		aspect-ratio: 1;
		border: 0.25rem solid var(--secondary);
		border-radius: 50%;
		background: linear-gradient(145deg, #e8eee7, #c5d4c4);
		box-shadow: inset 0 0 0 0.22rem #fffefb;
		overflow: hidden;
		z-index: 2;
	}

	.photo-frame img {
		width: 100%;
		height: 100%;
		object-fit: var(--photo-fit);
		object-position: var(--photo-x) var(--photo-y);
		transform: scale(var(--photo-zoom));
		transform-origin: center;
	}

	.photo-frame span {
		display: grid;
		place-items: center;
		height: 100%;
		font-family: ui-sans-serif, system-ui, sans-serif;
		font-size: 0.7rem;
		color: #526256;
	}

	.leaf-cluster {
		position: absolute;
		font-size: clamp(4rem, 13vw, 10rem);
		line-height: 0.7;
		color: color-mix(in srgb, var(--primary) 44%, white);
		opacity: 0.45;
		transform: rotate(-25deg);
		z-index: 1;
	}

	.leaf-cluster-left { left: -2.2rem; bottom: -0.5rem; }
	.leaf-cluster-right { right: -2.8rem; top: -1.3rem; transform: rotate(155deg); }

	.qr-block {
		position: absolute;
		left: 79%;
		top: 70%;
		z-index: 3;
		width: 17%;
		text-align: center;
		font-family: ui-sans-serif, system-ui, sans-serif;
		font-size: clamp(0.4rem, 0.9vw, 0.58rem);
		line-height: 1.1;
	}

	.editor-element {
		cursor: grab;
		touch-action: none;
		transition: outline-color 120ms ease, box-shadow 120ms ease;
	}

	.editor-element:hover {
		outline: 1px dashed color-mix(in srgb, var(--primary) 70%, white);
		outline-offset: 0.2rem;
	}

	.editor-element.selected {
		outline: 2px solid var(--primary);
		outline-offset: 0.22rem;
		box-shadow: 0 0 0 0.3rem color-mix(in srgb, var(--primary) 14%, transparent);
	}

	.editor-element:active { cursor: grabbing; }

	.qr-block img,
	.qr-placeholder {
		display: block;
		width: 100%;
		aspect-ratio: 1;
		background: #fff;
	}

	.replay-button {
		position: absolute;
		left: 1rem;
		bottom: 0.9rem;
		z-index: 8;
		width: 2.25rem;
		height: 2.25rem;
		border: 1px solid rgba(38, 50, 41, 0.25);
		border-radius: 999px;
		background: rgba(255, 255, 255, 0.9);
		color: #263229;
		box-shadow: 0 0.35rem 1rem rgba(38, 50, 41, 0.16);
		cursor: pointer;
	}

	.replay-button:hover { background: #fff; }
	.replay-button span[aria-hidden="true"] { font-size: 1.1rem; }

	.return-button,
	.play-reveal-button {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.45rem;
		margin: 0.85rem auto 0;
		border: 1px solid color-mix(in srgb, var(--primary) 40%, #ffffff);
		border-radius: 999px;
		padding: 0.6rem 1rem;
		background: var(--primary);
		color: #ffffff;
		font-family: ui-sans-serif, system-ui, sans-serif;
		font-size: 0.8rem;
		font-weight: 700;
		cursor: pointer;
	}

	.return-button {
		position: absolute;
		right: 1rem;
		bottom: 0.9rem;
		z-index: 8;
		margin: 0;
	}

	.play-reveal-button:hover,
	.return-button:hover { filter: brightness(0.94); }

	@keyframes envelope-descend {
		0%, 18% { transform: translate(-50%, -28%) scale(1.2); }
		55% { transform: translate(-50%, 10%) scale(1.2); }
		100% { transform: translate(-50%, 40%) scale(1.2); }
	}

	@keyframes card-rise {
		0%, 18% { opacity: 1; z-index: 3; clip-path: inset(0 0 55%); transform: translate(-50%, 24%) scale(0.68); }
		52% { opacity: 1; z-index: 3; clip-path: inset(0 0 22%); transform: translate(-50%, -40%) scale(0.68); }
		78% { opacity: 1; visibility: visible; z-index: 3; clip-path: inset(0); transform: translate(-50%, -210%) scale(0.75); }
		79%, 100% { opacity: 1; visibility: hidden; z-index: 3; clip-path: inset(0); transform: translate(-50%, -210%) scale(0.75); }
	}

	@keyframes reveal-final-card {
		0%, 78% { visibility: hidden; }
		79%, 100% { visibility: visible; }
	}

	@keyframes settle-final-card {
		0%, 78% { transform: translate(-50%, 8%) scale(0.75); }
		79% { transform: translate(-50%, 8%) scale(0.75); }
		100% { transform: translate(-50%, 0) scale(0.9); }
	}

	@media (max-width: 640px) {
		.botanical-stage { min-height: 28rem; }
		.botanical-card { padding: 0.8rem; }
	}
</style>
