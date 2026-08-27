<script lang="ts">
	import type { Plant } from '$lib/api';
	import { imgUrl } from '$lib/api';
	import { goto } from '$app/navigation';
	import Icon from '$lib/components/Icon.svelte';

	let { plant, pending = 0 }: { plant: Plant; pending?: number } = $props();

	function open() {
		goto(`/plants/${plant.id}`);
	}
</script>

<div class="card clickable plant-card" onclick={open} role="button" tabindex="0" onkeydown={(e) => e.key === 'Enter' && open()}>
	<div class="thumb">
		{#if plant.photo}
			<img src={imgUrl(plant.photo)} alt={plant.name} />
		{:else}
			<span class="placeholder">{plant.name.charAt(0)}</span>
		{/if}
		{#if pending > 0}
			<span class="badge" title="待办任务">{pending}</span>
		{/if}
	</div>
	<div class="info">
		<div class="name">{plant.name}</div>
		<div class="meta">
			{#if plant.species}
				{plant.species}
			{:else if plant.room}
				<Icon name={plant.room.icon || 'house'} size={13} /> {plant.room.name}
			{:else}
				未分组
			{/if}
		</div>
	</div>
</div>

<style>
	.plant-card {
		cursor: pointer;
		overflow: hidden;
	}
	.thumb {
		position: relative;
		aspect-ratio: 4 / 3;
		background: linear-gradient(135deg, var(--green-100), var(--green-50));
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.thumb img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.placeholder {
		font-size: 46px;
		font-weight: 700;
		color: var(--green-500);
	}
	.thumb .badge {
		position: absolute;
		top: 8px;
		right: 8px;
	}
	.info {
		padding: 12px 14px;
	}
	.name {
		font-weight: 600;
		font-size: 15px;
	}
	.meta {
		color: var(--text-secondary);
		font-size: 12px;
		margin-top: 2px;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
</style>
