// 结构化环境阈值解析与格式化工具（从 library 页面提取，便于测试）

export interface Metrics {
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

export function parseMetrics(raw?: string): Metrics | null {
	if (!raw) return null;
	try {
		return JSON.parse(raw) as Metrics;
	} catch {
		return null;
	}
}

export const range = (a?: number, b?: number, unit = '') =>
	a || b ? `${a ?? '–'} ~ ${b ?? '–'}${unit}` : '';
