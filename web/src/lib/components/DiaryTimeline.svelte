<script lang="ts">
	import type { PhotoDiary } from '$lib/api';
	import { imgUrl } from '$lib/api';
	import { formatDate } from '$lib/format';
	import { goto } from '$app/navigation';

	let { items, onDelete }: { items: PhotoDiary[]; onDelete?: (id: number) => void } = $props();
</script>

<div class="timeline">
	{#each items as d (d.id)}
		<div class="entry">
			<div class="dot"></div>
			<div class="date">{formatDate(d.takenAt)}</div>
			<div class="card bubble pop">
				<div class="media">
					{#if d.image}
						<img src={imgUrl(d.image)} alt={d.caption} />
					{:else}
						<div class="no-img">🪴</div>
					{/if}
				</div>
				<div class="content">
					{#if d.caption}<p class="caption">{d.caption}</p>{/if}
					<div class="meta">
						<button class="link" onclick={() => goto(`/plants/${d.plantId}`)}>{d.plant?.name ?? '植物'}</button>
						{#if onDelete}
							<button class="del" onclick={() => onDelete(d.id)}>删除</button>
						{/if}
					</div>
				</div>
			</div>
		</div>
	{/each}
</div>

<style>
	.timeline {
		position: relative;
		padding-left: 18px;
	}
	.timeline::before {
		content: '';
		position: absolute;
		left: 4px;
		top: 6px;
		bottom: 6px;
		width: 2px;
		background: var(--green-100);
	}
	.entry {
		position: relative;
		margin-bottom: 20px;
	}
	.dot {
		position: absolute;
		left: -18px;
		top: 6px;
		width: 10px;
		height: 10px;
		border-radius: 50%;
		background: var(--green-500);
		box-shadow: 0 0 0 3px var(--green-50);
	}
	.date {
		font-size: 12px;
		color: var(--text-secondary);
		margin-bottom: 6px;
	}
	.bubble {
		padding: 0;
		overflow: hidden;
	}
	.media {
		width: 100%;
		aspect-ratio: 16 / 9;
		background: var(--green-50);
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.media img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.no-img {
		font-size: 40px;
	}
	.content {
		padding: 12px 14px;
	}
	.caption {
		font-size: 14px;
		margin-bottom: 8px;
	}
	.meta {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-size: 13px;
	}
	.link {
		color: var(--green-700);
		font-weight: 500;
	}
	.del {
		color: var(--danger);
		font-size: 12px;
	}
</style>
