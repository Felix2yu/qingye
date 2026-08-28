<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import type { WeatherConfig, WeatherLog, SyncReport } from '$lib/api';
	import { showToast, settings } from '$lib/stores';
	import { theme, THEME_MODE_LABEL, type ThemeMode } from '$lib/theme.svelte';
	import Icon from '$lib/components/Icon.svelte';

	const THEME_OPTIONS: { mode: ThemeMode; icon: string; desc: string }[] = [
		{ mode: 'auto', icon: 'monitor', desc: '跟随系统亮暗变化' },
		{ mode: 'light', icon: 'sun', desc: '明亮清爽的日间配色' },
		{ mode: 'dark', icon: 'moon', desc: '柔和护眼的夜间配色' }
	];

	const WEEK = [
		{ value: 1, label: '周一' },
		{ value: 2, label: '周二' },
		{ value: 3, label: '周三' },
		{ value: 4, label: '周四' },
		{ value: 5, label: '周五' },
		{ value: 6, label: '周六' },
		{ value: 7, label: '周日' }
	];

	let selected = $state<number[]>([1, 2, 3, 4, 5]);
	let loading = $state(true);
	let saving = $state(false);

	// 天气配置
	let w = $state<WeatherConfig>({
		city: '', lat: 0, lon: 0,
		coldTemp: 5, hotTemp: 32, waterAdj: 30, fertAdj: 30,
		rainDelayH: 24, enabled: false, pollMinutes: 60
	});
	let weatherAvailable = $state(false);
	let wSaving = $state(false);
	let wLogs = $state<WeatherLog[]>([]);
	let showLogs = $state(false);

	// 通知
	let notifyURL = $state('');
	let notifySaving = $state(false);
	let notifyTesting = $state(false);
	let notifyMsg = $state('');
	let digestHour = $state(8);
	let digestSaving = $state(false);

	function toggle(d: number) {
		if (selected.includes(d)) {
			selected = selected.filter((x) => x !== d);
		} else {
			selected = [...selected, d].sort((a, b) => a - b);
		}
	}

	async function load() {
		loading = true;
		try {
			const [s, wc] = await Promise.all([api.getSettings(), api.getWeatherConfig()]);
			settings.set(s);
			notifyURL = s.notifyURL ?? '';
			digestHour = s.digestHour ?? 8;
			selected = s.workdays
				? s.workdays
						.split(',')
						.map((x) => Number(x.trim()))
						.filter((x) => x >= 1 && x <= 7)
				: [1, 2, 3, 4, 5];
			w = wc;
			weatherAvailable = (await api.weatherCurrent()).available;
			// 预先拉取日志（仅展示时再刷新）
			wLogs = await api.weatherLogs(30);
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			loading = false;
		}
	}

	async function save() {
		if (selected.length === 0) {
			showToast('至少选择一个工作日', 'err');
			return;
		}
		saving = true;
		try {
			const s = await api.updateSettings(selected);
			settings.set(s);
			showToast('已保存 🌿');
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			saving = false;
		}
	}

	async function saveWeather() {
		if (w.enabled && !w.city && !(w.lat && w.lon)) {
			showToast('启用天气策略需填写城市或经纬度', 'err');
			return;
		}
		wSaving = true;
		try {
			w = await api.saveWeatherConfig(w);
			showToast('天气策略已保存 🌤️');
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			wSaving = false;
		}
	}

	async function refreshNow() {
		wSaving = true;
		try {
			await api.weatherRefresh();
			showToast('已触发一次策略调整');
			wLogs = await api.weatherLogs(30);
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			wSaving = false;
		}
	}

	function kindLabel(k: string): string {
		return { cold: '❄️ 低温', hot: '🔥 高温', rain: '🌧️ 降雨', refresh: '🔄 刷新' }[k] ?? k;
	}

	// 批量同步热门植物资料库
	let syncing = $state(false);
	let syncNow = $state<{
		index: number;
		total: number;
		name: string;
		added: number;
		failed: number;
		duplicated: number;
		skipped: number;
	} | null>(null);
	let syncReport = $state<SyncReport | null>(null);
	async function syncPopular() {
		syncing = true;
		syncNow = null;
		syncReport = null;
		try {
			const r = await api.syncPopularLibraryStream((p) => {
				syncNow = {
					index: p.index,
					total: p.total,
					name: p.name,
					added: p.added,
					failed: p.failed,
					duplicated: p.duplicated,
					skipped: p.skipped
				};
			});
			syncReport = r;
			showToast(r.message);
		} catch (e) {
			syncReport = null;
			showToast((e as Error).message, 'err');
		} finally {
			syncing = false;
		}
	}
	// 进度百分比（用于进度条）
	let syncPct = $derived(syncNow && syncNow.total > 0 ? Math.round((syncNow.index / syncNow.total) * 100) : 0);

	onMount(load);

	// ---- 通知 ----
	async function saveNotify() {
		notifyMsg = '';
		notifySaving = true;
		try {
			const s = await api.saveNotify(notifyURL.trim());
			settings.set(s);
			notifyURL = s.notifyURL ?? '';
			showToast(notifyURL ? '通知已保存 🔔' : '已关闭通知');
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			notifySaving = false;
		}
	}

	async function testNotify() {
		notifyMsg = '';
		notifyTesting = true;
		try {
			const r = await api.testNotify();
			notifyMsg = r.message;
			showToast(r.message);
		} catch (e) {
			notifyMsg = (e as Error).message;
			showToast((e as Error).message, 'err');
		} finally {
			notifyTesting = false;
		}
	}

	async function saveDigestHour() {
		digestSaving = true;
		try {
			const s = await api.saveDigestHour(Number(digestHour));
			settings.set(s);
			digestHour = s.digestHour ?? 8;
			showToast(`每日推送时间已设为 ${String(digestHour).padStart(2, '0')}:00 🔔`);
		} catch (e) {
			showToast((e as Error).message, 'err');
		} finally {
			digestSaving = false;
		}
	}
</script>

<svelte:head><title>青野集 · 设置</title></svelte:head>

<div class="page">
	<h1 class="page-title">设置</h1>
	<p class="page-sub">配置养护节奏与智能策略</p>

	<div class="card setting-card">
		<div class="setting-head">
			<div class="setting-title">外观</div>
			<div class="muted">选择应用的主题外观，偏好保存在本设备。</div>
		</div>
		<div class="theme-options" role="radiogroup" aria-label="主题模式">
			{#each THEME_OPTIONS as opt}
				<button
					class="theme-option"
					class:on={theme.mode === opt.mode}
					onclick={() => theme.set(opt.mode)}
					role="radio"
					aria-checked={theme.mode === opt.mode}
				>
					<span class="theme-option-icon"><Icon name={opt.icon} size={20} /></span>
					<span class="theme-option-text">
						<span class="theme-option-name">{THEME_MODE_LABEL[opt.mode]}</span>
						<span class="theme-option-desc">{opt.desc}</span>
					</span>
				</button>
			{/each}
		</div>
	</div>

	<div class="card setting-card">
		<div class="setting-head">
			<div class="setting-title">工作日 / 休息日</div>
			<div class="muted">系统仅在「工作日」展示当日养护任务；休息日任务顺延，不会丢失。</div>
		</div>
		<div class="week">
			{#each WEEK as d}
				<button class="day" class:on={selected.includes(d.value)} onclick={() => toggle(d.value)}>
					{d.label}
				</button>
			{/each}
		</div>
		<div class="setting-actions">
			<button class="btn btn-primary" onclick={save} disabled={saving}>
				{saving ? '保存中…' : '保存设置'}
			</button>
		</div>
	</div>

	<div class="card setting-card">
		<div class="setting-head">
			<div class="setting-title">🌤️ 天气智能养护</div>
			<div class="muted">
				{#if !weatherAvailable}
					<span class="warn">未配置和风天气 API Key（环境变量 QWEATHER_KEY），天气策略不可用。</span>
				{:else}
					依据实时天气自动调整养护策略，并在下方查看调整日志。
				{/if}
			</div>
		</div>

		<label class="toggle-row">
			<input type="checkbox" bind:checked={w.enabled} />
			<span>启用天气智能养护</span>
		</label>

		<div class="form-grid">
			<div class="form-field"><label for="">城市（如：北京）</label><input bind:value={w.city} placeholder="或填写经纬度" /></div>
			<div class="form-field"><label for="">纬度 Lat</label><input type="number" step="0.01" bind:value={w.lat} /></div>
			<div class="form-field"><label for="">经度 Lon</label><input type="number" step="0.01" bind:value={w.lon} /></div>
		</div>

		<div class="rule-title">温度策略阈值（℃）</div>
		<div class="form-grid">
			<div class="form-field">
				<label for="">低温阈值（低于则降低浇水/施肥）</label>
				<input type="number" step="0.5" bind:value={w.coldTemp} />
			</div>
			<div class="form-field">
				<label for="">高温阈值（高于则降低施肥、保持/增加浇水）</label>
				<input type="number" step="0.5" bind:value={w.hotTemp} />
			</div>
		</div>

		<div class="rule-title">调整幅度与降雨</div>
		<div class="form-grid">
			<div class="form-field">
				<label for="">浇水调整幅度（%）</label>
				<input type="number" min="0" max="100" bind:value={w.waterAdj} />
			</div>
			<div class="form-field">
				<label for="">施肥调整幅度（%）</label>
				<input type="number" min="0" max="100" bind:value={w.fertAdj} />
			</div>
			<div class="form-field">
				<label for="">降雨推迟时长（小时）</label>
				<input type="number" min="1" bind:value={w.rainDelayH} />
			</div>
			<div class="form-field">
				<label for="">轮询间隔（分钟）</label>
				<input type="number" min="10" bind:value={w.pollMinutes} />
			</div>
		</div>

		<div class="setting-actions">
			<button class="btn btn-ghost" onclick={refreshNow} disabled={wSaving || !weatherAvailable || !w.enabled}>
				立即调整一次
			</button>
			<button class="btn btn-primary" onclick={saveWeather} disabled={wSaving}>
				{wSaving ? '保存中…' : '保存天气策略'}
			</button>
		</div>

		<div class="log-toggle">
			<button class="btn btn-ghost btn-sm" onclick={() => (showLogs = !showLogs)}>
				天气调整日志（{wLogs.length}）
			</button>
		</div>
		{#if showLogs}
			<div class="logs">
				{#if wLogs.length === 0}
					<p class="muted">暂无天气调整日志</p>
				{:else}
					{#each wLogs as lg}
						<div class="log">
							<span class="badge-kind">{kindLabel(lg.kind)}</span>
							<span class="log-detail">{lg.detail}</span>
							<span class="muted log-time">{lg.at.slice(0, 16).replace('T', ' ')}</span>
						</div>
					{/each}
				{/if}
			</div>
		{/if}
	</div>

	<div class="card setting-card">
		<div class="setting-title">📚 植物资料库同步</div>
		<p class="muted">
			从在线植物库（Plantbook）批量拉取常见植物的中文养护指南，沉淀到本地资料库，离线可用。需在服务端配置环境变量 PLANTBOOK_CLIENT_ID 与 PLANTBOOK_CLIENT_SECRET（open.plantbook.io 注册获取）。
			免费账户每日请求上限 200 次；每轮最多同步约 30 种。已入库条目、已确认在线库未收录的条目、以及同物异名（不同学名指向同一株）都会在建队列时直接排除，不消耗请求；单次点击不会越界。点击后实时显示进度，多次点击即可逐步补齐全表。
		</p>
		<div class="sync-row">
			<button class="btn btn-primary btn-sm" onclick={syncPopular} disabled={syncing}>
				{syncing ? '同步中…' : '同步热门植物'}
			</button>
			{#if syncNow}
				<span class="muted sync-live">正在同步第 {syncNow.index}/{syncNow.total} 个：{syncNow.name}（已加 {syncNow.added} · 失败 {syncNow.failed} · 同物异名 {syncNow.duplicated} · 已排除 {syncNow.skipped}）</span>
			{:else if syncReport}
				<span class="muted">{syncReport.message}</span>
			{/if}
		</div>
		{#if syncing && syncNow}
			<div class="progress" aria-hidden="true">
				<div class="progress-bar" style="width: {syncPct}%"></div>
			</div>
		{/if}
		{#if syncReport && syncReport.failedItems && syncReport.failedItems.length}
			<div class="sync-failed">
				<div class="muted">本轮失败 {syncReport.failedItems.length} 种：</div>
				<ul>
					{#each syncReport.failedItems as fi}<li>{fi}</li>{/each}
				</ul>
			</div>
		{/if}
	</div>

	<div class="card setting-card">
		<div class="setting-title">🔔 通知推送</div>
		<p class="muted">
			填写 <a href="https://containrrr.dev/shoutrrr/" target="_blank" rel="noreferrer">shoutrrr</a>
			服务地址，即可接收养护提醒与天气策略调整通知。支持 Gotify、Telegram、企业微信、钉钉、Bark、Webhook、SMTP 等。留空表示不推送。
		</p>
		<div class="form-field notify-field">
			<label for="">shoutrrr URL</label>
			<input
				id="notify"
				bind:value={notifyURL}
				placeholder="例如：gotify://gotify.example.com/AaBbCcDdEeFf"
				spellcheck="false"
				autocomplete="off"
			/>
		</div>
		{#if notifyMsg}
			<p class="muted notify-msg">{notifyMsg}</p>
		{/if}
		<div class="digest-row">
			<label for="digest-hour">每日养护摘要推送时间</label>
			<select id="digest-hour" bind:value={digestHour}>
				{#each Array.from({ length: 24 }, (_, h) => h) as h}
					<option value={h}>{String(h).padStart(2, '0')}:00</option>
				{/each}
			</select>
			<button class="btn btn-ghost btn-sm" onclick={saveDigestHour} disabled={digestSaving}>
				{digestSaving ? '保存中…' : '保存时间'}
			</button>
		</div>
		<div class="setting-actions">
			<button class="btn btn-ghost" onclick={testNotify} disabled={notifyTesting || !notifyURL.trim()}>
				{notifyTesting ? '发送中…' : '发送测试'}
			</button>
			<button class="btn btn-primary" onclick={saveNotify} disabled={notifySaving}>
				{notifySaving ? '保存中…' : notifyURL.trim() ? '保存通知' : '关闭通知'}
			</button>
		</div>
	</div>

	<div class="card setting-card">
		<div class="setting-title">关于</div>
		<p class="muted">青野集 · 家庭园艺植物记录与养护</p>
	</div>
</div>

<style>
	.setting-card {
		padding: 20px;
		margin-bottom: 16px;
	}
	.theme-options {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
		gap: 10px;
	}
	.theme-option {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 14px;
		border-radius: var(--radius-sm);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--text);
		text-align: left;
		transition: border-color 0.15s, background 0.15s;
	}
	.theme-option:hover {
		border-color: var(--green-500);
	}
	.theme-option.on {
		border-color: var(--green-600);
		background: var(--green-50);
	}
	.theme-option-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 38px;
		height: 38px;
		border-radius: 10px;
		background: var(--green-100);
		color: var(--green-700);
		flex-shrink: 0;
	}
	.theme-option-text {
		display: flex;
		flex-direction: column;
		min-width: 0;
	}
	.theme-option-name {
		font-size: 14px;
		font-weight: 600;
		line-height: 1.4;
	}
	.theme-option-desc {
		font-size: 12px;
		color: var(--text-secondary);
		line-height: 1.4;
	}
	.setting-head {
		margin-bottom: 16px;
	}
	.setting-title {
		font-weight: 600;
		font-size: 15px;
		margin-bottom: 4px;
	}
	.week {
		display: flex;
		flex-wrap: wrap;
		gap: 10px;
		margin-bottom: 18px;
	}
	.day {
		padding: 10px 16px;
		border-radius: 12px;
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--text-secondary);
		font-weight: 500;
		transition: all 0.15s;
	}
	.day.on {
		background: var(--green-600);
		border-color: var(--green-600);
		color: var(--on-accent);
	}
	.setting-actions {
		display: flex;
		justify-content: flex-end;
		gap: 10px;
		flex-wrap: wrap;
	}
	.warn {
		color: var(--warning);
	}
	.toggle-row {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-bottom: 16px;
		cursor: pointer;
	}
	.toggle-row input {
		width: auto;
	}
	.form-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
		gap: 12px;
		margin-bottom: 14px;
		align-items: end;
	}
	.form-field {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.form-field label {
		min-height: 34px;
		line-height: 17px;
		font-size: 13px;
		color: var(--text-secondary);
	}
	.rule-title {
		font-weight: 600;
		font-size: 13px;
		margin: 6px 0 10px;
		color: var(--text-secondary);
	}
	.log-toggle {
		margin-top: 16px;
	}
	.logs {
		margin-top: 10px;
		border-top: 1px dashed var(--border);
		padding-top: 10px;
		display: flex;
		flex-direction: column;
		gap: 8px;
		max-height: 280px;
		overflow-y: auto;
	}
	.log {
		display: flex;
		gap: 10px;
		align-items: baseline;
		font-size: 13px;
	}
	.badge-kind {
		flex-shrink: 0;
		background: var(--green-50);
		padding: 1px 8px;
		border-radius: 999px;
		font-size: 12px;
	}
	.log-detail {
		flex: 1;
	}
	.log-time {
		flex-shrink: 0;
	}
	.notify-field {
		margin: 14px 0 4px;
	}
	.notify-field input {
		font-family: var(--mono, ui-monospace, SFMono-Regular, Menlo, monospace);
		font-size: 13px;
	}
	.notify-msg {
		margin: 6px 0 0;
		word-break: break-all;
	}
	.digest-row {
		display: flex;
		align-items: center;
		gap: 10px;
		flex-wrap: wrap;
		margin: 14px 0 4px;
	}
	.digest-row label {
		font-size: 13px;
		color: var(--text-secondary);
	}
	.digest-row select {
		width: auto;
		min-width: 96px;
	}
	.sync-row {
		display: flex;
		align-items: center;
		gap: 12px;
		flex-wrap: wrap;
	}
	.sync-live {
		font-variant-numeric: tabular-nums;
	}
	.progress {
		margin-top: 12px;
		height: 8px;
		border-radius: 999px;
		background: var(--border);
		overflow: hidden;
	}
	.progress-bar {
		height: 100%;
		border-radius: 999px;
		background: linear-gradient(90deg, var(--green-500), var(--green-600));
		transition: width 0.2s ease;
	}
	.sync-failed {
		margin-top: 12px;
		padding: 10px 12px;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface);
	}
	.sync-failed ul {
		margin: 6px 0 0;
		padding-left: 18px;
		display: flex;
		flex-direction: column;
		gap: 4px;
		font-size: 13px;
		color: var(--text-secondary);
	}
</style>
