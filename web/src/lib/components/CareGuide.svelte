<script lang="ts">
	import Icon from '$lib/components/Icon.svelte';
	import { parseCareGuide } from '$lib/careGuide';

	let { guide }: { guide: string | null | undefined } = $props();

	const sections = $derived(parseCareGuide(guide));
</script>

{#if sections.length}
	<div class="care-guide">
		{#each sections as s (s.key + s.label)}
			<div class="care-sec">
				<span class="care-ic"><Icon name={s.icon} size={18} /></span>
				<div class="care-main">
					<div class="care-label">{s.label}</div>
					{#if s.bullets.length > 1}
						<ul class="care-bullets">
							{#each s.bullets as b}<li>{b}</li>{/each}
						</ul>
					{:else}
						<div class="care-text">{s.bullets[0]}</div>
					{/if}
				</div>
			</div>
		{/each}
	</div>
{/if}

<style>
	.care-guide {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}
	.care-sec {
		display: flex;
		gap: 12px;
		padding: 12px 14px;
		border-radius: 12px;
		background: var(--green-50);
		border: 1px solid var(--border);
	}
	.care-ic {
		width: 34px;
		height: 34px;
		border-radius: 10px;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		background: var(--bg);
		color: var(--green-700);
		border: 1px solid var(--border);
	}
	.care-main {
		flex: 1;
		min-width: 0;
	}
	.care-label {
		font-weight: 700;
		font-size: 13px;
		color: var(--green-700);
		margin-bottom: 4px;
	}
	.care-text {
		font-size: 13.5px;
		line-height: 1.65;
		color: var(--text);
	}
	.care-bullets {
		margin: 0;
		padding-left: 18px;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.care-bullets li {
		font-size: 13.5px;
		line-height: 1.6;
		color: var(--text);
	}
</style>
