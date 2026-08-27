<script lang="ts">
	import { onMount } from 'svelte';
	import { api, imgUrl } from '$lib/api';
	import type { Plant, PhotoDiary } from '$lib/api';
	import { showToast } from '$lib/stores';
	import { compressImage } from '$lib/compress';
	import DiaryTimeline from '$lib/components/DiaryTimeline.svelte';
	import { page } from '$app/state';

	let items = $state<PhotoDiary[]>([]);
	let plants = $state<Plant[]>([]);
	let loading = $state(true);
	let plantFilter = $state<number>(0);

	let showForm = $state(false);
	let fPlant = $state<number>(0);
	let fCaption = $state('');
	let fTakenAt = $state('');
	let fFile = $state<File | null>(null);
	let preview = $state('');
	let busy = $state(false);

	async function load() {
		loading = true;
		try {
			const [d, p] = await Promise.all([
				api.listDiaries({ plantId: plantFilter || undefined }),
				api.listPlants()
			]);
			items = d.list;
			plants = p;
			if (fPlant === 0 && p.length > 0) fPlant = p[0].id;
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			loading = false;
		}
	}

	async function onFile(e: Event) {
		const input = e.target as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;
		fFile = file;
		preview = URL.createObjectURL(file);
	}

	async function submit() {
		if (!fPlant) {
			showToast('请选择植物', 'err');
			return;
		}
		if (!fFile) {
			showToast('请选择图片', 'err');
			return;
		}
		busy = true;
		try {
			const img = await compressImage(fFile);
			const form = new FormData();
			form.append('plantId', String(fPlant));
			form.append('caption', fCaption);
			form.append('takenAt', fTakenAt ? new Date(fTakenAt).toISOString() : new Date().toISOString());
			form.append('image', img);
			await api.createDiary(form);
			showToast('已记录 📷');
			showForm = false;
			fCaption = '';
			fFile = null;
			preview = '';
			await load();
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			busy = false;
		}
	}

	async function del(id: number) {
		if (!confirm('删除这条日记？')) return;
		try {
			await api.deleteDiary(id);
			showToast('已删除');
			await load();
		} catch (e) {
			showToast((e as Error).message, 'err');
		}
	}

	onMount(() => {
		const q = page.url.searchParams.get('plantId');
		if (q) plantFilter = Number(q);
		load();
	});
</script>

<svelte:head><title>青野集 · 日记</title></svelte:head>

<div class="page">
	<div class="head">
		<div>
			<h1 class="page-title">照片日记</h1>
			<p class="page-sub">记录每一株植物的成长瞬间</p>
		</div>
		<button class="btn btn-primary btn-sm" onclick={() => (showForm = true)}>＋ 记一笔</button>
	</div>

	{#if plants.length > 0}
		<div class="chips">
			<button class="chip" class:active={plantFilter === 0} onclick={() => ((plantFilter = 0), load())}>全部</button>
			{#each plants as p}
				<button class="chip" class:active={plantFilter === p.id} onclick={() => ((plantFilter = p.id), load())}>
					{p.name}
				</button>
			{/each}
		</div>
	{/if}

	{#if loading}
		<p class="empty"><span class="emoji">🌿</span>加载中…</p>
	{:else if items.length === 0}
		<p class="empty"><span class="emoji">📷</span>还没有日记，给植物拍张照吧</p>
	{:else}
		<DiaryTimeline {items} onDelete={del} />
	{/if}
</div>

{#if showForm}
	<div
		class="modal-backdrop"
		role="button"
		tabindex="-1"
		onclick={(e) => {
			if (e.target === e.currentTarget) showForm = false;
		}}
		onkeydown={(e) => {
			if (e.key === 'Escape') showForm = false;
		}}
	>
		<div class="modal">
			<div class="modal-title">记录一笔</div>
			<div class="form-field">
				<label for="diary-plant">植物 *</label>
				<select id="diary-plant" bind:value={fPlant}>
					{#each plants as p}<option value={p.id}>{p.name}</option>{/each}
				</select>
			</div>
			<div class="form-field">
				<label for="diary-image">图片 *</label>
				<input id="diary-image" type="file" accept="image/*" onchange={onFile} />
				{#if preview}
					<img class="preview" src={preview} alt="预览" />
				{/if}
			</div>
			<div class="form-field">
				<label for="diary-caption">说明</label>
				<textarea id="diary-caption" bind:value={fCaption} rows="2" placeholder="今天的状态…"></textarea>
			</div>
			<div class="form-field">
				<label for="diary-takenat">拍摄日期</label>
				<input id="diary-takenat" type="date" bind:value={fTakenAt} />
			</div>
			<div class="modal-actions">
				<button class="btn btn-ghost" onclick={() => (showForm = false)}>取消</button>
				<button class="btn btn-primary" onclick={submit} disabled={busy}>保存</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.head {
		display: flex;
		justify-content: space-between;
		align-items: flex-end;
		gap: 12px;
		flex-wrap: wrap;
	}
	.preview {
		margin-top: 10px;
		width: 100%;
		max-height: 200px;
		object-fit: cover;
		border-radius: 10px;
	}
</style>
