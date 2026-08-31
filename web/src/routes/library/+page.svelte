<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import type { PlantLibrary } from '$lib/api';
	import { showToast } from '$lib/stores';
	import { zhCategory, zhOrigin } from '$lib/plant-zh';
	import CareGuide from '$lib/components/CareGuide.svelte';

	let list = $state<PlantLibrary[]>([]);
	let keyword = $state('');
	let loading = $state(true);
	let showDetail = $state<PlantLibrary | null>(null);
	let refreshing = $state(false);
	let resyncing = $state(false);
	let resyncProgress = $state({ index: 0, total: 0, name: '', count: 0 });

	// 当前详情条目的全部常见名
	const detailNames = $derived(showDetail ? commonNames(showDetail) : []);

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

	// ---- 结构化环境阈值（Plantbook 同步条目携带）----
	interface Metrics {
		minLightMmol?: number;
		maxLightMmol?: number;
		minLightLux?: number;
		maxLightLux?: number;
		minTemp?: number;
		maxTemp?: number;
		minEnvHumid?: number;
		maxEnvHumid?: number;
		minSoilMoist?: number;
		maxSoilMoist?: number;
		minSoilEc?: number;
		maxSoilEc?: number;
	}

	function parseMetrics(raw?: string): Metrics | null {
		if (!raw) return null;
		try {
			return JSON.parse(raw) as Metrics;
		} catch {
			return null;
		}
	}

	const range = (a?: number, b?: number, unit = '') =>
		a || b ? `${a ?? '–'} ~ ${b ?? '–'}${unit}` : '';

	// 生成指标展示行（仅列出有数据的项）
	function metricRows(p: PlantLibrary): [string, string][] {
		const m = parseMetrics(p.metrics);
		if (!m) return [];
		const rows: [string, string][] = [];
		const add = (label: string, v: string) => v && rows.push([label, v]);
		add('光照 PPFD', range(m.minLightMmol, m.maxLightMmol, ' μmol/㎡·s'));
		add('光照照度', range(m.minLightLux, m.maxLightLux, ' lx'));
		add('温度', range(m.minTemp, m.maxTemp, ' ℃'));
		add('空气湿度', range(m.minEnvHumid, m.maxEnvHumid, ' %'));
		add('土壤水分', range(m.minSoilMoist, m.maxSoilMoist, ' %'));
		add('土壤 EC', range(m.minSoilEc, m.maxSoilEc, ' µS/cm'));
		return rows;
	}

	// 解析常见名 JSON 数组，仅保留中文名
	function commonNames(p: PlantLibrary): string[] {
		if (!p.commonNames) return [];
		try {
			const arr = JSON.parse(p.commonNames);
			if (!Array.isArray(arr)) return [];
			return arr.filter((x) => typeof x === 'string' && /[\u4e00-\u9fa5]/.test(x));
		} catch {
			return [];
		}
	}

	onMount(() => search(''));

	async function refreshGuide() {
		refreshing = true;
		try {
			const res = await api.refreshLibraryGuide();
			showToast(`已刷新 ${res.refreshed} 条指南`, 'ok');
			await search(keyword.trim());
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			refreshing = false;
		}
	}

	async function resyncAndTranslate() {
		if (!confirm('将从Plantbook重新拉取所有植物的英文指南并翻译为中文，消耗API配额（每日200次上限）。确定继续？')) {
			return;
		}
		resyncing = true;
		resyncProgress = { index: 0, total: 0, name: '', count: 0 };
		try {
			const res = await api.resyncAndTranslateLibrary((p) => {
			resyncProgress = { index: p.index, total: p.total, name: p.name, count: p.count };
		});
		showToast(`重新拉取完成：成功 ${res.success} 条，失败 ${res.failed} 条`, 'ok');
		await search(keyword.trim());
	} catch (e) {
		showToast((e as Error).message, 'err');
	} finally {
		resyncing = false;
	}
}

async function clearLibrary() {
	if (!confirm('确定要清空资料库所有数据吗？此操作不可恢复！')) {
		return;
	}
	try {
		await api.clearLibrary();
		showToast('资料库已清空', 'ok');
		await search(keyword.trim());
	} catch (e) {
		showToast((e as Error).message, 'err');
	}
}
</script>

<svelte:head><title>青野集 · 资料库</title></svelte:head>

<div class="page">
	<h1 class="page-title">植物资料库</h1>
	<p class="page-sub">常见室内植物养护指南</p>

	<div class="search">
		<input bind:value={keyword} oninput={onInput} placeholder="搜索植物名称或别名，如：绿萝、多肉" />
		<button class="btn btn-ghost btn-sm refresh-btn" onclick={refreshGuide} disabled={refreshing}>
			{refreshing ? '翻译中…' : '刷新中文'}
		</button>
		<button class="btn btn-ghost btn-sm resync-btn" onclick={resyncAndTranslate} disabled={resyncing}>
			{resyncing ? `重新拉取中 ${resyncProgress.index}/${resyncProgress.total}` : '重新拉取并翻译'}
		</button>
		<button class="btn btn-ghost btn-sm clear-btn" onclick={clearLibrary}>
			清空资料库
		</button>
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
	<div
		class="modal-backdrop"
		role="button"
		tabindex="-1"
		onclick={(e) => {
			if (e.target === e.currentTarget) showDetail = null;
		}}
		onkeydown={(e) => {
			if (e.key === 'Escape') showDetail = null;
		}}
	>
		<div class="modal">
			<div class="modal-title">{showDetail.name}</div>
			<p class="muted meta-line">
				{#if showDetail.displayPid || showDetail.alias}
					<span title={showDetail.alias}>{showDetail.displayPid || showDetail.alias}</span>
				{/if}
				{#if showDetail.category}
					<span title={showDetail.category}>· {zhCategory(showDetail.category)}</span>
				{/if}
				{#if showDetail.origin}
					<span title={showDetail.origin}>· 原产 {zhOrigin(showDetail.origin)}</span>
				{/if}
			</p>

			<div class="metrics">
				{#each metricRows(showDetail) as [label, value]}
					<div class="metric">
						<span class="metric-label">{label}</span>
						<span class="metric-value">{value}</span>
					</div>
				{/each}
			</div>

			<CareGuide guide={showDetail.guide} />

			{#if detailNames.length > 0}
				<p class="muted cn-line">常见名：{detailNames.join('、')}</p>
			{/if}

			<div class="modal-actions">
				{#if showDetail.link}
					<a class="btn btn-ghost" href={showDetail.link} target="_blank" rel="noopener noreferrer">
						Plantbook 页面 ↗
					</a>
				{/if}
				<button class="btn btn-primary" onclick={() => (showDetail = null)}>知道了</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.search {
		margin-bottom: 20px;
		display: flex;
		gap: 10px;
		align-items: center;
	}
	.search input {
		flex: 1;
	}
	.refresh-btn {
		white-space: nowrap;
	}
	.resync-btn {
		white-space: nowrap;
	}
	.clear-btn {
		white-space: nowrap;
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
		line-clamp: 3;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
	.meta-line {
		display: flex;
		gap: 6px;
		flex-wrap: wrap;
		margin-bottom: 12px;
		font-style: italic;
	}
	.cn-line {
		margin-top: 14px;
		padding-top: 10px;
		border-top: 1px dashed var(--border);
	}
	.metrics {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
		gap: 8px;
		margin-bottom: 14px;
	}
	.metric {
		display: flex;
		flex-direction: column;
		gap: 2px;
		background: var(--green-50);
		border-radius: 10px;
		padding: 8px 12px;
	}
	.metric-label {
		font-size: 11px;
		color: var(--text-secondary);
	}
	.metric-value {
		font-size: 13px;
		font-weight: 600;
		color: var(--green-700);
		font-variant-numeric: tabular-nums;
	}
</style>
