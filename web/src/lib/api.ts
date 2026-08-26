// 与后端统一响应结构 {code, message, data} 对应的封装

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
}

export interface Plant {
	id: number;
	name: string;
	species: string;
	photo: string;
	roomId: number;
	room?: { id: number; name: string; sort: number } | null;
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
	name: string;
	alias: string;
	guide: string;
	image: string;
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
}

export interface DiaryPage {
	list: PhotoDiary[];
	total: number;
	page: number;
	pageSize: number;
}

const BASE = (import.meta.env.VITE_API_BASE as string | undefined) ?? '';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
	const isForm = options?.body instanceof FormData;
	const res = await fetch(`${BASE}${path}`, {
		headers: isForm ? undefined : { 'Content-Type': 'application/json' },
		...options
	});
	const json = (await res.json().catch(() => null)) as ApiResponse<T> | null;
	if (!res.ok || !json || json.code !== 0) {
		throw new Error(json?.message || `请求失败 (${res.status})`);
	}
	return json.data;
}

export const api = {
	// ---- 房间 ----
	listRooms: () => request<Room[]>('/api/rooms'),
	createRoom: (name: string, sort = 0, isOutdoor = false) =>
		request<Room>('/api/rooms', { method: 'POST', body: JSON.stringify({ name, sort, isOutdoor }) }),
	updateRoom: (id: number, name: string, sort = 0, isOutdoor?: boolean) =>
		request<Room>(`/api/rooms/${id}`, {
			method: 'PUT',
			body: JSON.stringify({ name, sort, isOutdoor })
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
	// 批量同步内置热门植物到本地资料库（离线可用）
	syncPopularLibrary: () =>
		request<{ added: number; failed: number; total: number; message: string }>('/api/library/sync-popular', {
			method: 'POST'
		}),

	// ---- 设置 ----
	getSettings: () => request<UserSetting>('/api/settings'),
	updateSettings: (workdays: number[], prefs?: Record<string, unknown>) =>
		request<UserSetting>('/api/settings', {
			method: 'PUT',
			body: JSON.stringify({ workdays, prefs })
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
