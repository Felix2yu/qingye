<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import type { Plant, Room, Task, OnlineCandidate, PlantLibrary } from '$lib/api';
	import { showToast } from '$lib/stores';
	import { imgUrl } from '$lib/api';
	import PlantCard from '$lib/components/PlantCard.svelte';

	let plants = $state<Plant[]>([]);
	let rooms = $state<Room[]>([]);
	let tasks = $state<Task[]>([]);
	let loading = $state(true);

	let roomFilter = $state<number>(0); // 0 = 全部
	let showForm = $state(false);
	let showRoomForm = $state(false);

	// 表单
	let fName = $state('');
	let fSpecies = $state('');
	let fRoom = $state<number>(0);
	let fNote = $state('');
	let fPhoto = $state('');
	let fLocation = $state('');
	let fLightReq = $state('');
	let fAcquired = $state('');
	let fAttributes = $state('');

	let rName = $state('');
	let rSort = $state(0);
	let rOutdoor = $state(false);

	// 在线匹配（Plantbook）
	let onlineEnabled = $state(false);
	let onlineCands = $state<OnlineCandidate[]>([]);
	let onlineLoading = $state(false);
	let onlineMsg = $state('');
	let matchedGuide = $state<PlantLibrary | null>(null); // 选中后带回的指南

	async function toggleRoomOutdoor(r: Room) {
		try {
			await api.updateRoom(r.id, r.name, r.sort, !r.isOutdoor);
			showToast(!r.isOutdoor ? '已标记为室外' : '已标记为室内');
			await load();
		} catch (e) {
			showToast((e as Error).message, 'err');
		}
	}

	async function load() {
		loading = true;
		try {
			const [p, r, t] = await Promise.all([
				api.listPlants(),
				api.listRooms(),
				api.listTasks({ includeDone: false })
			]);
			plants = p;
			rooms = r;
			tasks = t;
			if (fRoom === 0 && r.length > 0) fRoom = r[0].id;
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			loading = false;
		}
	}

	function pendingCount(plantId: number): number {
		return tasks.filter((t) => t.plantId === plantId && t.active).length;
	}

	// 计算每种房间待办数（用于角标展示，可选）
	let filtered = $derived(
		roomFilter === 0 ? plants : plants.filter((p) => p.roomId === roomFilter)
	);

	async function submitPlant() {
		if (!fName.trim()) {
			showToast('请填写植物名称', 'err');
			return;
		}
		try {
			await api.createPlant({
				name: fName.trim(),
				species: fSpecies.trim(),
				roomId: fRoom,
				note: fNote.trim(),
				photo: fPhoto.trim(),
				location: fLocation.trim(),
				lightReq: fLightReq.trim(),
				acquiredDate: fAcquired.trim(),
				attributes: fAttributes.trim()
			});
			showToast('已添加植物 🌱');
			showForm = false;
			fName = fSpecies = fNote = fPhoto = fLocation = fLightReq = fAcquired = fAttributes = '';
			await load();
		} catch (e) {
			showToast((e as Error).message, 'err');
		}
	}

	async function submitRoom() {
		if (!rName.trim()) {
			showToast('请填写房间名称', 'err');
			return;
		}
		try {
			await api.createRoom(rName.trim(), rSort, rOutdoor);
			showToast('已添加房间');
			showRoomForm = false;
			rName = '';
			rSort = 0;
			rOutdoor = false;
			await load();
		} catch (e) {
			showToast((e as Error).message, 'err');
		}
	}

	onMount(load);

	// 手动触发在线匹配（输入名称后点"在线查找"）
	async function searchOnline() {
		const kw = fName.trim();
		if (kw.length < 2) {
			showToast('请输入至少 2 个字再查找', 'err');
			return;
		}
		onlineLoading = true;
		onlineMsg = '';
		onlineCands = [];
		matchedGuide = null;
		try {
			const res = await api.searchLibraryOnline(kw);
			onlineEnabled = res.enabled;
			if (!res.enabled) {
				onlineMsg = '未配置 Plantbook token，在线匹配已禁用';
				return;
			}
			onlineCands = res.list;
			if (res.list.length === 0) onlineMsg = '在线库未找到匹配，可手动填写';
		} catch (e) {
			onlineMsg = (e as Error).message;
		} finally {
			onlineLoading = false;
		}
	}

	// 选中候选 → 拉详情并写回本地资料库，带回中文指南
	async function pickOnline(c: OnlineCandidate) {
		onlineLoading = true;
		try {
			const lib = await api.importLibraryOnline(c.pid);
			matchedGuide = lib;
			fName = lib.name || c.name;
			fSpecies = c.alias;
			fNote = lib.guide;
			if (c.image && !fPhoto) fPhoto = c.image;
			showToast('已从在线库带入养护指南');
			onlineCands = [];
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			onlineLoading = false;
		}
	}</script>

<svelte:head><title>青野 · 植物</title></svelte:head>

<div class="page">
	<div class="head">
		<div>
			<h1 class="page-title">我的植物</h1>
			<p class="page-sub">共 {plants.length} 株 · 按房间整理</p>
		</div>
		<div class="head-actions">
			<button class="btn btn-ghost btn-sm" onclick={() => (showRoomForm = true)}>＋ 房间</button>
			<button class="btn btn-primary btn-sm" onclick={() => (showForm = true)}>＋ 植物</button>
		</div>
	</div>

	{#if rooms.length > 0}
		<div class="chips">
			<button class="chip" class:active={roomFilter === 0} onclick={() => (roomFilter = 0)}>全部</button>
			{#each rooms as r}
				<button class="chip" class:active={roomFilter === r.id} onclick={() => (roomFilter = r.id)} title={r.isOutdoor ? '室外 · 点击切换' : '室内 · 点击切换'}>
					{r.name}{r.count ? ` · ${r.count}` : ''}
					{#if r.isOutdoor}<span class="outdoor-dot" title="室外">☀️</span>{/if}
				</button>
			{/each}
		</div>
	{/if}

	{#if loading}
		<p class="empty"><span class="emoji">🌿</span>加载中…</p>
	{:else if filtered.length === 0}
		<p class="empty"><span class="emoji">🪴</span>这里还没有植物，点右上角添加一株吧</p>
	{:else}
		<div class="grid grid-cards">
			{#each filtered as p (p.id)}
				<PlantCard plant={p} pending={pendingCount(p.id)} />
			{/each}
		</div>
	{/if}
</div>

<!-- 新增植物 -->
{#if showForm}
	<div class="modal-backdrop" onclick={() => (showForm = false)}>
		<div class="modal" onclick={(e) => e.stopPropagation()}>
			<div class="modal-title">添加植物</div>
			<div class="form-field">
				<label for="">名称 *</label>
				<div class="name-row">
					<input bind:value={fName} placeholder="如：绿萝" />
					<button class="btn btn-ghost btn-sm" type="button" onclick={searchOnline} disabled={onlineLoading}>
						{onlineLoading ? '查找中…' : '在线查找'}
					</button>
				</div>
				{#if onlineMsg}<p class="hint">{onlineMsg}</p>{/if}
				{#if onlineCands.length > 0}
					<div class="online-list">
						{#each onlineCands as c (c.pid)}
							<button class="online-item" type="button" onclick={() => pickOnline(c)}>
								{#if c.image}<img src={imgUrl(c.image)} alt="" class="online-thumb" />{/if}
								<span class="online-name">{c.name}</span>
								<span class="online-alias">{c.alias}</span>
							</button>
						{/each}
					</div>
				{/if}
				{#if matchedGuide}
					<div class="matched-guide">
						<span class="badge">✓ 已带入在线指南</span>
						<pre>{matchedGuide.guide}</pre>
					</div>
				{/if}
			</div>
			<div class="form-field">
				<label for="">品种</label>
				<input bind:value={fSpecies} placeholder="如：Epipremnum aureum" />
			</div>
			<div class="form-field">
				<label for="">所在房间 / 分组</label>
				<select bind:value={fRoom}>
					{#each rooms as r}<option value={r.id}>{r.name}</option>{/each}
				</select>
			</div>
			<div class="form-field">
				<label for="">详细方位</label>
				<input bind:value={fLocation} placeholder="如：阳台左侧" />
			</div>
			<div class="form-field">
				<label for="">光照需求</label>
				<input bind:value={fLightReq} placeholder="如：散射光 / 耐阴" />
			</div>
			<div class="form-field">
				<label for="">获得日期</label>
				<input bind:value={fAcquired} type="date" />
			</div>
			<div class="form-field">
				<label for="">照片链接（可选）</label>
				<input bind:value={fPhoto} placeholder="https://… 或留空用首字占位" />
			</div>
			<div class="form-field">
				<label for="">备注</label>
				<textarea bind:value={fNote} rows="2" placeholder="养护小贴士…"></textarea>
			</div>
			<div class="form-field">
				<label for="">扩展属性 (JSON)</label>
				<textarea bind:value={fAttributes} rows="2" placeholder={'{"土壤":"泥炭土"}'}></textarea>
			</div>
			<div class="modal-actions">
				<button class="btn btn-ghost" onclick={() => (showForm = false)}>取消</button>
				<button class="btn btn-primary" onclick={submitPlant}>保存</button>
			</div>
		</div>
	</div>
{/if}

<!-- 新增房间 -->
{#if showRoomForm}
	<div class="modal-backdrop" onclick={() => (showRoomForm = false)}>
		<div class="modal" onclick={(e) => e.stopPropagation()}>
			<div class="modal-title">添加房间 / 分组</div>
			<div class="form-field">
				<label for="">名称 *</label>
				<input bind:value={rName} placeholder="如：书房" />
			</div>
			<div class="form-field">
				<label for="">排序</label>
				<input type="number" bind:value={rSort} />
			</div>
			<label class="toggle-row">
				<input type="checkbox" bind:checked={rOutdoor} />
				<span>室外（阳台/花园，降雨时自动推迟浇水）</span>
			</label>
			<div class="modal-actions">
				<button class="btn btn-ghost" onclick={() => (showRoomForm = false)}>取消</button>
				<button class="btn btn-primary" onclick={submitRoom}>保存</button>
			</div>
		</div>
	</div>
{/if}
