<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import type { PlantLibrary } from '$lib/api';
	import { showToast } from '$lib/stores';

	let list = $state<PlantLibrary[]>([]);
	let keyword = $state('');
	let loading = $state(true);
	let showDetail = $state<PlantLibrary | null>(null);

	async function search(q = '') {
		loading = true;
		try {
			list = await api.searchLibrary(q);
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			loading = false;
		}
	}

	let timer: ReturnType<typeof setTimeout>;
	function onInput() {
		clearTimeout(timer);
		timer = setTimeout(() => search(keyword.trim()), 250);
	}

	onMount(() => search(''));
</script>

<svelte:head><title>青野 · 资料库</title></svelte:head>

<div class="page">
	<h1 class="page-title">植物资料库</h1>
	<p class="page-sub">常见室内植物养护指南</p>

	<div class="search">
		<input bind:value={keyword} oninput={onInput} placeholder="搜索植物名称或别名，如：绿萝、多肉" />
	</div>

	{#if loading}
		<p class="empty"><span class="emoji">🌿</span>加载中…</p>
	{:else if list.length === 0}
		<p class="empty"><span class="emoji">🔍</span>没有找到相关植物</p>
	{:else}
		<div class="grid grid-cards">
			{#each list as p (p.id)}
				<button class="card clickable lib-card" onclick={() => (showDetail = p)}>
					<div class="lib-name">{p.name}</div>
					{#if p.alias}<div class="lib-alias">别名：{p.alias}</div>{/if}
					<p class="lib-guide">{p.guide}</p>
				</button>
			{/each}
		</div>
	{/if}
</div>

{#if showDetail}
	<div class="modal-backdrop" onclick={() => (showDetail = null)}>
		<div class="modal" onclick={(e) => e.stopPropagation()}>
			<div class="modal-title">{showDetail.name}</div>
			{#if showDetail.alias}<p class="muted" style="margin-bottom:12px">别名：{showDetail.alias}</p>{/if}
			<p class="guide-text">{showDetail.guide}</p>
			<div class="modal-actions">
				<button class="btn btn-primary" onclick={() => (showDetail = null)}>知道了</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.search {
		margin-bottom: 20px;
	}
	.lib-card {
		text-align: left;
		padding: 14px 16px;
		cursor: pointer;
		display: block;
	}
	.lib-name {
		font-weight: 700;
		font-size: 15px;
	}
	.lib-alias {
		font-size: 12px;
		color: var(--text-secondary);
		margin-top: 2px;
	}
	.lib-guide {
		font-size: 13px;
		color: var(--text-secondary);
		margin-top: 8px;
		display: -webkit-box;
		-webkit-line-clamp: 3;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
	.guide-text {
		line-height: 1.8;
		color: var(--text);
	}
</style>
