<script lang="ts">
	import { onMount } from 'svelte';
	import { api, type ImportPreview, type ImportResult, type ImportRow } from '$lib/api';
	import { showToast } from '$lib/stores';
	import { TASK_TYPE_LABEL } from '$lib/format';
	import type { Plant } from '$lib/api';

	type Mode = 'plants' | 'tasks' | 'template';
	let mode = $state<Mode>('plants');

	// 植物/任务 CSV
	let file = $state<File | null>(null);
	let preview = $state<ImportPreview | null>(null);
	let fileText = $state(''); // 保留原文用于确认
	let loading = $state(false);
	let result = $state<ImportResult | null>(null);
	let selected = $state<Set<number>>(new Set());

	// 模板复制
	let plants = $state<Plant[]>([]);
	let sourceId = $state<number>(0);
	let targetSel = $state<Set<number>>(new Set());
	let tplPreview = $state<ImportPreview | null>(null);

	onMount(async () => {
		try {
			plants = await api.listPlants();
			if (plants.length) {
				sourceId = plants[0].id;
				targetSel = new Set(plants.map((p) => p.id));
			}
		} catch (e) {
			showToast((e as Error).message, 'err');
		}
	});

	function onFile(e: Event) {
		const input = e.target as HTMLInputElement;
		file = input.files?.[0] ?? null;
		preview = null;
		result = null;
		selected = new Set();
	}

	async function doPreview() {
		if (!file) return showToast('请先选择 CSV 文件', 'err');
		loading = true;
		try {
			fileText = await file.text();
			preview = await api.importPreview(mode as 'plants' | 'tasks', file);
			selected = new Set(preview.rows.filter((r) => r.status !== 'error').map((r) => r.line));
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			loading = false;
		}
	}

	function toggle(line: number) {
		const s = new Set(selected);
		if (s.has(line)) s.delete(line);
		else s.add(line);
		selected = s;
	}

	async function doConfirm() {
		if (!preview) return;
		loading = true;
		try {
			result = await api.importConfirm({
				kind: preview.kind,
				content: fileText,
				accepted: [...selected]
			});
			showToast(result.message, 'ok');
			preview = null;
			file = null;
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			loading = false;
		}
	}

	// 模板
	function toggleTarget(id: number) {
		const s = new Set(targetSel);
		if (s.has(id)) s.delete(id);
		else s.add(id);
		targetSel = s;
	}

	async function doTplPreview() {
		if (!sourceId) return showToast('请选择来源植物', 'err');
		loading = true;
		try {
			tplPreview = await api.importTemplatePreview(sourceId, [...targetSel]);
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			loading = false;
		}
	}

	async function doTplConfirm() {
		if (!tplPreview || !sourceId) return;
		loading = true;
		try {
			result = await api.importConfirm({ kind: 'template', sourceId, targetIds: [...targetSel] });
			showToast(result.message, 'ok');
			tplPreview = null;
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			loading = false;
		}
	}

	function rowText(r: ImportRow): string {
		const d = r.data as Record<string, unknown>;
		if (d.plantName !== undefined) {
			return `${d.plantName} · ${TASK_TYPE_LABEL[d.type as string] ?? d.type} · 每${d.intervalDays}天`;
		}
		return [d.name, d.species, d.room, d.location, d.lightReq, d.note]
			.filter((x) => x)
			.join(' / ');
	}

	const samplePlants = 'name,species,room,note,location,lightReq,acquiredDate\n龟背竹,Monstera deliciosa,客厅,喜散射光,阳台左侧,散射光,2026-05-01\n绿萝,,卧室,耐阴,,耐阴,';
	const sampleTasks = 'plant,type,intervalDays,title,startDate\n龟背竹,water,7,浇水,2026-09-01\n绿萝,fertilize,30,施肥,\n薄荷,prune,14,修剪,\n月季,pesticide,21,除虫,';
</script>

<svelte:head><title>批量导入 · 青野集</title></svelte:head>

<div class="wrap">
	<h1>批量导入</h1>
	<p class="sub">支持 CSV 批量导入植物/任务，以及把某株植物的任务模板复制给多株植物。导入前先预览，确认后再落库。</p>

	<div class="tabs">
		<button class:active={mode === 'plants'} onclick={() => (mode = 'plants')}>导入植物 (CSV)</button>
		<button class:active={mode === 'tasks'} onclick={() => (mode = 'tasks')}>导入任务 (CSV)</button>
		<button class:active={mode === 'template'} onclick={() => (mode = 'template')}>模板复制</button>
	</div>

	{#if mode !== 'template'}
		<div class="card">
			<label class="filelabel">
				选择 CSV 文件
				<input type="file" accept=".csv,text/csv" onchange={onFile} />
			</label>
			{#if file}<div class="filename">已选择：{file.name}</div>{/if}
			<button class="primary" onclick={doPreview} disabled={!file || loading}>
				{loading ? '解析中…' : '预览'}
			</button>
			<div class="sample">
				表头示例：
				<code>{mode === 'plants' ? samplePlants : sampleTasks}</code>
				<span class="hint">任务类型取值：water(浇水) / fertilize(施肥) / mist(喷雾) / repot(换盆) / prune(修剪) / clean(清理) / pesticide(除虫) / other(其他)</span>
			</div>
		</div>

		{#if preview}
			<div class="card">
				<div class="summary">
					{preview.summary}
					<span class="ok">可导入 {preview.valid}</span>
					<span class="err">错误 {preview.invalid}</span>
				</div>
				<table>
					<thead>
						<tr>
							<th>选择</th><th>行</th><th>状态</th><th>内容</th><th>说明</th>
						</tr>
					</thead>
					<tbody>
						{#each preview.rows as r (r.line)}
							<tr class:errrow={r.status === 'error'}>
								<td><input type="checkbox" checked={selected.has(r.line)} onchange={() => toggle(r.line)} disabled={r.status === 'error'} /></td>
								<td>{r.line}</td>
								<td>
									<span class="badge {r.status}">
										{r.status === 'ok' ? '正常' : r.status === 'warning' ? '提醒' : '错误'}
									</span>
								</td>
								<td class="content">{rowText(r)}</td>
								<td class="reason">{r.reason}</td>
							</tr>
						{/each}
					</tbody>
				</table>
				<button class="primary" onclick={doConfirm} disabled={loading || selected.size === 0}>
					确认导入（{selected.size} 行）
				</button>
			</div>
		{/if}
	{:else}
		<div class="card">
			<label>来源植物（复制其任务配置）</label>
			<select bind:value={sourceId}>
				{#each plants as p}<option value={p.id}>{p.name}</option>{/each}
			</select>
			<label class="mt">目标植物（勾选要应用模板的植物）</label>
			<div class="checks">
				{#each plants as p}
					<label class="chk"><input type="checkbox" checked={targetSel.has(p.id)} onchange={() => toggleTarget(p.id)} /> {p.name}</label>
				{/each}
			</div>
			<button class="primary" onclick={doTplPreview} disabled={loading || !sourceId}>预览</button>
		</div>

		{#if tplPreview}
			<div class="card">
				<div class="summary">{tplPreview.summary}</div>
				<table>
					<thead><tr><th>目标植物</th><th>来源</th><th>任务数</th><th>说明</th></tr></thead>
					<tbody>
						{#each tplPreview.rows as r (r.line)}
							<tr class:errrow={r.status === 'error'}>
								<td>{r.data.targetName}</td>
								<td>{r.data.sourceName}</td>
								<td>{r.data.taskCount}</td>
								<td class="reason">{r.reason}</td>
							</tr>
						{/each}
					</tbody>
				</table>
				<button class="primary" onclick={doTplConfirm} disabled={loading}>确认复制</button>
			</div>
		{/if}
	{/if}

	{#if result}
		<div class="result">✅ {result.message}</div>
	{/if}
	</div>

	<style>
	.wrap { max-width: 880px; margin: 0 auto; padding: 28px 20px 60px; }
	h1 { font-size: 24px; margin: 0 0 6px; }
	.sub { color: var(--text-secondary); margin: 0 0 18px; line-height: 1.6; }
	.tabs { display: flex; gap: 8px; margin-bottom: 18px; flex-wrap: wrap; }
	.tabs button {
		padding: 9px 16px; border-radius: 999px; border: 1px solid var(--border);
		background: var(--surface); color: var(--text-secondary); cursor: pointer; font-size: 14px;
	}
	.tabs button.active { background: var(--green-600); color: var(--on-accent); border-color: var(--green-600); }
	.card {
		background: var(--surface); border: 1px solid var(--border); border-radius: 16px;
		padding: 18px; margin-bottom: 16px; display: flex; flex-direction: column; gap: 12px;
	}
	.filelabel { font-weight: 600; display: flex; flex-direction: column; gap: 6px; }
	.filename { font-size: 13px; color: var(--green-700); }
	.primary {
		align-self: flex-start; background: var(--green-600); color: var(--on-accent); border: none;
		padding: 10px 22px; border-radius: 12px; font-size: 15px; cursor: pointer;
	}
	.primary:disabled { opacity: 0.5; cursor: not-allowed; }
	.sample { font-size: 12px; color: var(--text-secondary); display: flex; flex-direction: column; gap: 4px; }
	.sample code { background: var(--bg); padding: 6px 8px; border-radius: 8px; font-size: 12px; white-space: pre-wrap; }
	.hint { color: var(--green-700); }
	.summary { font-weight: 600; display: flex; gap: 12px; align-items: center; }
	.summary .ok { color: var(--green-600); }
	.summary .err { color: var(--danger); }
	table { width: 100%; border-collapse: collapse; font-size: 13px; }
	th, td { text-align: left; padding: 8px 6px; border-bottom: 1px solid var(--border); vertical-align: top; }
	.errrow { background: var(--err-row-bg); }
	.content { color: var(--text-secondary); }
	.reason { color: var(--warn-strong); }
	.badge { padding: 2px 8px; border-radius: 999px; font-size: 12px; }
	.badge.ok { background: var(--green-100); color: var(--green-700); }
	.badge.warning { background: var(--warn-soft-bg); color: var(--warn-strong); }
	.badge.error { background: var(--danger-soft); color: var(--danger-strong); }
	select { padding: 9px 12px; border-radius: 10px; border: 1px solid var(--border); background: var(--surface); }
	.checks { display: flex; flex-wrap: wrap; gap: 10px; }
	.chk { display: inline-flex; align-items: center; gap: 6px; }
	.mt { margin-top: 8px; }
	.result { background: var(--green-50); color: var(--green-700); padding: 12px 16px; border-radius: 12px; font-weight: 600; }
</style>
