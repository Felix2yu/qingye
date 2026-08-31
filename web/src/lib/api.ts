// 与后端统一响应结构 {code, message, data} 对应的封装

import { browser } from '$app/environment';
import { showToast } from './stores';
import { enqueue, prepareRequest } from './offline';

export interface ApiResponse<T> {
	code: number;
	message: string;
	data: T;
}

// ---- 实体类型 ----
export interface Room {
	id: number;
	name: string;
	sort: number;
	count?: number;
	isOutdoor?: boolean;
	icon?: string;
}

export interface Plant {
	id: number;
	name: string;
	species: string;
	photo: string;
	roomId: number;
	room?: { id: number; name: string; sort: number; icon?: string } | null;
	note: string;
	// 扩充属性（参考 hortusfox）
	location: string;
	acquiredDate: string | null;
	lightReq: string;
	attributes: string;
	createdAt: string;
	updatedAt: string;
}

export interface Task {
	id: number;
	plantId: number;
	plant?: Plant;
	type: string; // water / fertilize / repot
	title: string;
	intervalDays: number;
	baseIntervalDays: number;
	lastDoneAt: string | null;
	nextDue: string;
	active: boolean;
	createdAt: string;
}

export interface CareLog {
	id: number;
	plantId: number;
	plant?: Plant;
	type: string; // water / fertilize / repot / other
	title: string;
	at: string;
	note: string;
	source: 'manual' | 'task';
	taskId?: number | null;
}

export interface TaskLog {
	id: number;
	taskId: number;
	action: 'done' | 'postpone';
	at: string;
	note: string;
}

export interface PhotoDiary {
	id: number;
	plantId: number;
	plant?: Plant;
	image: string;
	caption: string;
	takenAt: string;
	createdAt: string;
}

export interface PlantLibrary {
	id: number;
	pid?: string;
	displayPid?: string; // 学名展示形式
	name: string;
	alias: string;
	category?: string; // 植物类别
	origin?: string; // 原产地
	commonNames?: string; // 全部常见名（JSON 字符串数组）
	guide: string;
	metrics?: string; // 结构化环境阈值 JSON（见 library 页解析展示）
	image: string;
	link?: string; // Plantbook 在线详情页
}

export interface OnlineCandidate {
	pid: string;
	name: string;   // 优先中文常见名
	alias: string;  // 学名
	image: string;  // 缩略图
	commonNames: string[];
}

export interface UserSetting {
	id: number;
	workdays: string; // "1,2,3,4,5"
	prefs: string; // JSON
	notifyURL: string; // shoutrrr URL，为空表示未开启通知
	digestHour: number; // 每日摘要推送小时（0-23）
}

export interface DiaryPage {
	list: PhotoDiary[];
	total: number;
	page: number;
	pageSize: number;
}

const BASE = (import.meta.env.VITE_API_BASE as string | undefined) ?? '';

// 把 JSON 请求体解析回对象，作为离线占位返回值（调用方不依赖其做跳转）
function parseBodyPlaceholder(body?: BodyInit | null): unknown {
	if (!body || typeof body !== 'string') return {};
	try {
		return JSON.parse(body);
	} catch {
		return {};
	}
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
	const isForm = options?.body instanceof FormData;
	const method = options?.method ?? 'GET';
	const isMutation = method !== 'GET' && method !== 'HEAD';
	const url = `${BASE}${path}`;
	try {
		const res = await fetch(url, {
			headers: isForm ? undefined : { 'Content-Type': 'application/json' },
			...options
		});
		const json = (await res.json().catch(() => null)) as ApiResponse<T> | null;
		if (!res.ok || !json || json.code !== 0) {
			throw new Error(json?.message || `请求失败 (${res.status})`);
		}
		return json.data;
	} catch (err) {
		// 仅在“断网”时把写操作写入离线队列，其余错误原样抛出
		if (isMutation && browser && !navigator.onLine) {
			try {
				const prepared = prepareRequest(method, url, options);
				await enqueue(prepared);
				showToast('已离线保存，联网后自动同步', 'info');
				// 返回占位对象，让调用方走成功分支（其不依赖返回值做跳转）
				return (isForm ? undefined : parseBodyPlaceholder(options?.body)) as T;
			} catch {
				// 队列不可用（如隐私模式禁用 IndexedDB）时退回原始错误
				throw err;
			}
		}
		throw err;
	}
}

// 资料库批量同步：单条进度事件（SSE event: progress）
export interface SyncProgressEvent {
	type?: string;
	index: number; // 当前植物在待同步队列中的位置（1-based）
	total: number; // 本轮待同步队列长度
	name: string; // 当前植物中文名
	status: string; // added | failed | duplicate
	added: number;
	failed: number;
	duplicated: number; // 同物异名：解析到的 pid 本地已有，仅耗 1 次检索
	skipped: number; // 建队列时即排除（本地已同步 / 已确认未收录）
	remaining: number; // 队列中尚未开始的条目数
}

// 资料库批量同步：整轮汇总（SSE event: done / JSON 降级）
export interface SyncReport {
	added: number;
	failed: number;
	duplicated: number;
	skipped: number;
	remaining: number;
	total: number;
	throttled: boolean;
	quotaHit: boolean;
	message: string;
	failedItems: string[];
}

// 重新拉取并翻译：单条进度事件（SSE event: progress）
export interface ResyncProgress {
	type?: string;
	index: number;
	total: number;
	name: string;
	status: string; // success | failed | skipped
	count: number;  // 本轮成功数
}

// 重新拉取并翻译：整轮汇总（SSE event: done）
export interface ResyncReport {
	total: number;
	success: number;
	failed: number;
}

// 解析单个 SSE 帧：返回 {event, data}
function parseSSEFrame(frame: string): { event: string; data: string } | null {
	let event = 'message';
	let data = '';
	for (const line of frame.split('\n')) {
		if (line.startsWith('event:')) event = line.slice(6).trim();
		else if (line.startsWith('data:')) data += line.slice(5).trim();
	}
	return { event, data };
}

export const api = {
	// ---- 房间 ----
	listRooms: () => request<Room[]>('/api/rooms'),
	createRoom: (name: string, sort = 0, isOutdoor = false, icon = '') =>
		request<Room>('/api/rooms', { method: 'POST', body: JSON.stringify({ name, sort, isOutdoor, icon }) }),
	updateRoom: (id: number, name: string, sort = 0, isOutdoor?: boolean, icon?: string) =>
		request<Room>(`/api/rooms/${id}`, {
			method: 'PUT',
			body: JSON.stringify({ name, sort, isOutdoor, icon })
		}),
	deleteRoom: (id: number) => request<void>(`/api/rooms/${id}`, { method: 'DELETE' }),

	// ---- 植物 ----
	listPlants: (roomId?: number) =>
		request<Plant[]>(`/api/plants${roomId ? `?roomId=${roomId}` : ''}`),
	getPlant: (id: number) => request<Plant>(`/api/plants/${id}`),
	createPlant: (data: Partial<Plant>) =>
		request<Plant>('/api/plants', { method: 'POST', body: JSON.stringify(data) }),
	updatePlant: (id: number, data: Partial<Plant>) =>
		request<Plant>(`/api/plants/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
	deletePlant: (id: number) => request<void>(`/api/plants/${id}`, { method: 'DELETE' }),

	// ---- 任务 ----
	listTasks: (params: { type?: string; includeDone?: boolean; plantId?: number } = {}) => {
		const q = new URLSearchParams();
		if (params.type) q.set('type', params.type);
		if (params.includeDone) q.set('includeDone', 'true');
		if (params.plantId) q.set('plantId', String(params.plantId));
		const qs = q.toString();
		return request<Task[]>(`/api/tasks${qs ? `?${qs}` : ''}`);
	},
	todayTasks: () => request<Task[]>('/api/tasks/today'),
	upcomingTasks: (days = 3) => request<Task[]>(`/api/tasks/upcoming?days=${days}`),
	createTask: (data: Partial<Task>) =>
		request<Task>('/api/tasks', { method: 'POST', body: JSON.stringify(data) }),
	doneTask: (id: number, note = '') =>
		request<Task>(`/api/tasks/${id}/done`, {
			method: 'POST',
			body: JSON.stringify({ note })
		}),
	postponeTask: (id: number, days: number, note = '') =>
		request<Task>(`/api/tasks/${id}/postpone`, {
			method: 'POST',
			body: JSON.stringify({ days, note })
		}),
	taskLogs: (id: number) => request<TaskLog[]>(`/api/tasks/${id}/logs`),
	deleteTask: (id: number) => request<void>(`/api/tasks/${id}`, { method: 'DELETE' }),

	// ---- 日记 ----
	listDiaries: (params: { plantId?: number; page?: number; pageSize?: number } = {}) => {
		const q = new URLSearchParams();
		if (params.plantId) q.set('plantId', String(params.plantId));
		if (params.page) q.set('page', String(params.page));
		if (params.pageSize) q.set('pageSize', String(params.pageSize));
		const qs = q.toString();
		return request<DiaryPage>(`/api/diaries${qs ? `?${qs}` : ''}`);
	},
	createDiary: (form: FormData) =>
		request<PhotoDiary>('/api/diaries', { method: 'POST', body: form }),
	deleteDiary: (id: number) => request<void>(`/api/diaries/${id}`, { method: 'DELETE' }),

	// ---- 统一养护时间线 ----
	careLogs: (plantId?: number, limit = 0) => {
		const q = plantId ? `?plantId=${plantId}` : limit ? `?limit=${limit}` : '';
		return request<CareLog[]>('/api/care-logs' + q);
	},
	recordCare: (plantId: number, type: string, title: string, note: string, at?: string) =>
		request<CareLog>('/api/care-logs', {
			method: 'POST',
			body: JSON.stringify({ plantId, type, title, note, at: at ?? '' })
		}),

	// ---- 资料库 ----
	searchLibrary: (q: string) =>
		request<PlantLibrary[]>(`/api/library?q=${encodeURIComponent(q)}`),
	// 在线匹配（Plantbook）：返回候选列表，未配置 token 时 enabled=false
	searchLibraryOnline: (q: string) =>
		request<{ enabled: boolean; list: OnlineCandidate[] }>(`/api/library/online?keyword=${encodeURIComponent(q)}`),
	// 按 pid 拉取详情并写回本地资料库
	importLibraryOnline: (pid: string) =>
		request<PlantLibrary>('/api/library/import', {
			method: 'POST',
			body: JSON.stringify({ pid })
		}),
	// 本地刷新：将所有英文养护指南翻译为中文（不调用外部API）
	refreshLibraryGuide: () =>
		request<{ refreshed: number }>('/api/library/refresh-guide', {
			method: 'POST'
		}),
	// 重新拉取所有植物的英文Guide并翻译为中文（消耗API配额）
	resyncAndTranslateLibrary: (onProgress: (p: ResyncProgress) => void): Promise<ResyncReport> =>
		new Promise((resolve, reject) => {
			const es = new EventSource(`/api/library/resync-and-translate`);
			es.addEventListener('progress', (e) => {
				try {
					onProgress(JSON.parse(e.data));
				} catch {}
			});
			es.addEventListener('done', (e) => {
				es.close();
				try {
					resolve(JSON.parse(e.data));
				} catch (err) {
					reject(err);
				}
			});
			es.addEventListener('error', () => {
				es.close();
				reject(new Error('连接失败'));
			});
		}),

	// 植物详情页：取该植物在资料库中最匹配的养护指南（found=false 表示无匹配）
	getPlantCareGuide: (id: number) =>
		request<{
			found: boolean;
			libraryId?: number;
			name?: string;
			alias?: string;
			guide?: string;
			link?: string;
		}>(`/api/plants/${id}/care-guide`),
	// 批量同步内置热门植物到本地资料库（离线可用，每轮限量）
	// 以 SSE 实时推送进度，onProgress 每处理完一种植物回调一次；
	// 返回整轮 SyncReport。服务端不支持 SSE 时降级为单次 JSON。
	syncPopularLibraryStream: (onProgress: (p: SyncProgressEvent) => void): Promise<SyncReport> =>
		new Promise<SyncReport>((resolve, reject) => {
			const url = `${BASE}/api/library/sync-popular`;
			fetch(url, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' }
			})
				.then((resp) => {
					if (!resp.ok) {
						resp
							.text()
							.then((t) => reject(new Error(t || `同步失败 (${resp.status})`)))
							.catch(() => reject(new Error(`同步失败 (${resp.status})`)));
						return;
					}
					const ct = resp.headers.get('content-type') || '';
					if (ct.includes('text/event-stream') && resp.body) {
						const reader = resp.body.getReader();
						const decoder = new TextDecoder();
						let buf = '';
						const pump = (): Promise<void> =>
							reader.read().then(({ done, value }) => {
								if (done) {
									// 流结束：若残留可直接解析的 JSON（降级场景）则解析
									if (buf.trim()) {
										try {
											resolve(JSON.parse(buf) as SyncReport);
										} catch {
											reject(new Error('同步流异常结束'));
										}
									}
									return;
								}
								buf += decoder.decode(value, { stream: true });
								let idx: number;
								while ((idx = buf.indexOf('\n\n')) >= 0) {
									const frame = buf.slice(0, idx);
									buf = buf.slice(idx + 2);
									const parsed = parseSSEFrame(frame);
									if (!parsed) continue;
									if (parsed.event === 'progress') {
										try {
											onProgress(JSON.parse(parsed.data) as SyncProgressEvent);
										} catch {
											/* 忽略坏帧 */
										}
									} else if (parsed.event === 'done') {
										try {
											resolve(JSON.parse(parsed.data) as SyncReport);
										} catch (e) {
											reject(e as Error);
										}
										return;
									}
								}
								return pump();
							});
						pump().catch(reject);
					} else {
						// 非流式（单测/降级）：直接 JSON
						resp.json().then((r) => resolve(r as SyncReport)).catch(reject);
					}
				})
				.catch(reject);
		}),

	// ---- 设置 ----
	getSettings: () => request<UserSetting>('/api/settings'),
	updateSettings: (workdays: number[], prefs?: Record<string, unknown>) =>
		request<UserSetting>('/api/settings', {
			method: 'PUT',
			body: JSON.stringify({ workdays, prefs })
		}),
	// 通知
	saveNotify: (url: string) =>
		request<UserSetting>('/api/settings/notify', {
			method: 'PUT',
			body: JSON.stringify({ url })
		}),
	saveDigestHour: (hour: number) =>
		request<UserSetting>('/api/settings/digest-hour', {
			method: 'PUT',
			body: JSON.stringify({ hour })
		}),
	testNotify: () =>
		request<{ message: string }>('/api/notify/test', {
			method: 'POST'
		}),

	// ---- 批量导入 ----
	importPreview: (kind: 'plants' | 'tasks', file: File) => {
		const form = new FormData();
		form.append('kind', kind);
		form.append('file', file);
		return request<ImportPreview>('/api/import/preview', { method: 'POST', body: form });
	},
	importConfirm: (req: ImportConfirmRequest) =>
		request<ImportResult>('/api/import/confirm', { method: 'POST', body: JSON.stringify(req) }),
	importTemplatePreview: (sourceId: number, targetIds: number[]) =>
		request<ImportPreview>('/api/import/template-preview', {
			method: 'POST',
			body: JSON.stringify({ sourceId, targetIds })
		}),

	// ---- 天气与智能养护 ----
	weatherCurrent: () => request<WeatherCurrent>('/api/weather/current'),
	getWeatherConfig: () => request<WeatherConfig>('/api/weather/config'),
	saveWeatherConfig: (cfg: WeatherConfig) =>
		request<WeatherConfig>('/api/weather/config', { method: 'PUT', body: JSON.stringify(cfg) }),
	weatherLogs: (limit = 50) => request<WeatherLog[]>(`/api/weather/logs?limit=${limit}`),
	weatherRefresh: () => request<WeatherNow | null>('/api/weather/refresh', { method: 'POST' })
};

// ---- 导入相关类型 ----
export type ImportRowStatus = 'ok' | 'warning' | 'error';

export interface ImportRow {
	line: number;
	status: ImportRowStatus;
	reason: string;
	data: Record<string, unknown>;
}

export interface ImportPreview {
	kind: 'plants' | 'tasks' | 'template';
	rows: ImportRow[];
	valid: number;
	invalid: number;
	summary: string;
}

export interface ImportConfirmRequest {
	kind: 'plants' | 'tasks' | 'template';
	content?: string;
	accepted?: number[];
	sourceId?: number;
	targetIds?: number[];
}

export interface ImportResult {
	kind: string;
	created: number;
	skipped: number;
	plantIds: number[];
	taskIds: number[];
	message: string;
}

// ---- 天气与智能养护类型 ----
export interface WeatherNow {
	temp: number;
	condition: string;
	icon: string;
	obsTime: string;
}

export interface WeatherConfig {
	city: string;
	lat: number;
	lon: number;
	coldTemp: number;
	hotTemp: number;
	waterAdj: number;
	fertAdj: number;
	rainDelayH: number;
	enabled: boolean;
	pollMinutes: number;
}

export interface WeatherCurrent {
	config: WeatherConfig;
	current: WeatherNow | null;
	available: boolean;
}

export interface WeatherLog {
	id: number;
	at: string;
	temp: number;
	condition: string;
	kind: 'cold' | 'hot' | 'rain' | 'refresh';
	taskId?: number | null;
	plantId?: number | null;
	detail: string;
}

// 解析图片地址：相对路径 /uploads/... 在代理或部署下都可直接使用
export function imgUrl(path: string): string {
	if (!path) return '';
	return path.startsWith('http') ? path : `${BASE}${path}`;
}
