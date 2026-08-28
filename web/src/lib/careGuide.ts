// 养护指南解析：把后端 buildGuide 生成的「字段：描述」纯文本，
// 解析为结构化分区，便于前端做图标 + 标签 + 要点列表的排版。
//
// 输入示例（Plantbook 同步条目，描述多为英文）：
//   浇水：Drought-tolerant; water when soil is dry; reduce watering in winter.
//   光照：Tolerates half-shade, prefers moderate sunlight, avoid direct strong light.
//   温度：10℃ ~ 32℃
//   土壤：No strict soil requirements; prefers sandy loam.
//   施肥：Dilute fertilizers as instructed; apply once monthly ...
//   修剪：Remove dead, yellow and diseased leaves promptly

export interface CareSection {
	key: string; // 原始字段名（如「浇水」），用作列表 key
	label: string; // 展示标签（与 key 一致，预留翻译空间）
	icon: string; // Icon.svelte 中的图标名
	tone: string; // 主题色基调（green/amber/red/blue/brown/gray）
	bullets: string[]; // 拆分后的要点；长度为 1 时按单行展示
}

interface SectionMeta {
	icon: string;
	tone: string;
}

// 已知分区 → 图标 / 主题色。未匹配到的字段回落到 leaf/green。
const SECTION_META: Record<string, SectionMeta> = {
	浇水: { icon: 'droplet', tone: 'blue' },
	光照: { icon: 'sun', tone: 'amber' },
	温度: { icon: 'thermometer', tone: 'red' },
	湿度: { icon: 'sprayCan', tone: 'cyan' },
	土壤: { icon: 'mountain', tone: 'brown' },
	施肥: { icon: 'sprout', tone: 'green' },
	修剪: { icon: 'scissors', tone: 'gray' }
};

const LINE_RE = /^(.+?)[:：]\s*(.+)$/;

export function parseCareGuide(guide: string | null | undefined): CareSection[] {
	if (!guide || !guide.trim()) return [];
	const sections: CareSection[] = [];

	for (const raw of guide.split(/\r?\n/)) {
		const line = raw.trim();
		if (!line) continue;

		const m = LINE_RE.exec(line);
		if (!m) {
			// 无「字段：」前缀的自由文本 → 归入「通用建议」
			const last = sections[sections.length - 1];
			if (last && last.key === '通用建议') {
				last.bullets.push(line);
			} else {
				sections.push({ key: '通用建议', label: '通用建议', icon: 'info', tone: 'gray', bullets: [line] });
			}
			continue;
		}

		const label = m[1].trim();
		const value = m[2].trim();
		const meta = SECTION_META[label] ?? { icon: 'leaf', tone: 'green' };
		sections.push({
			key: label,
			label,
			icon: meta.icon,
			tone: meta.tone,
			bullets: splitBullets(value)
		});
	}

	return sections;
}

// 按中/英文分号拆分为要点；单条则保持原样（单行展示）。
// 去掉拆分后尾随的句号，避免列表项以标点结尾。
function splitBullets(value: string): string[] {
	const parts = value
		.split(/[;；]/)
		.map((s) => s.trim().replace(/[.。]+$/, '').trim())
		.filter(Boolean);
	if (parts.length <= 1) return [value.replace(/\s+/g, ' ').trim()];
	return parts;
}
