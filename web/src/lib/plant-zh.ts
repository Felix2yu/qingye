// Open Plantbook 元数据的展示层中文映射。
// 数据库保留 API 原文；此处仅负责界面呈现，未命中的条目回退英文原文。
// 养护描述（light/watering/soil 等）为自由文本，不做翻译。

// 类别：按关键词模糊匹配（API 取值为英文短语）
const CATEGORY_RULES: Array<[string[], string]> = [
	[['foliage', 'foliary'], '观叶植物'],
	[['flowering', 'flower'], '开花植物'],
	[['succulent'], '多肉植物'],
	[['cacti', 'cactus'], '仙人掌类'],
	[['orchid'], '兰科植物'],
	[['fern'], '蕨类植物'],
	[['palm'], '棕榈类'],
	[['bonsai'], '盆景'],
	[['herb'], '香草'],
	[[ 'climber', 'vine'], '藤蔓植物'],
	[['bulb'], '球根植物'],
	[[ 'aquatic'], '水生植物'],
	[['tree'], '乔木'],
	[['shrub'], '灌木'],
	[['grass', 'bamboo'], '草本竹类'],
	[['vegetable'], '蔬菜'],
	[[ 'fruit'], '果树'],
];

export function zhCategory(v?: string): string {
	if (!v) return '';
	const s = v.toLowerCase();
	for (const [keys, zh] of CATEGORY_RULES) {
		if (keys.some((k) => s.includes(k))) return zh;
	}
	return v;
}

// 原产地 / 分布区：常见国家与地理区域
const ORIGIN_RULES: Array<[string[], string]> = [
	[['mexico'], '墨西哥'],
	[['guatemala'], '危地马拉'],
	[['costa rica'], '哥斯达黎加'],
	[['central america'], '中美洲'],
	[['south america'], '南美洲'],
	[['tropical america'], '热带美洲'],
	[['brazil'], '巴西'],
	[['argentina'], '阿根廷'],
	[['colombia'], '哥伦比亚'],
	[['peru'], '秘鲁'],
	[['ecuador'], '厄瓜多尔'],
	[['bolivia'], '玻利维亚'],
	[['paraguay'], '巴拉圭'],
	[['chile'], '智利'],
	[['madagascar'], '马达加斯加'],
	[['south africa'], '南非'],
	[['africa'], '非洲'],
	[['ethiopia'], '埃塞俄比亚'],
	[['kenya'], '肯尼亚'],
	[['tanzania'], '坦桑尼亚'],
	[['china'], '中国'],
	[['japan'], '日本'],
	[['korea'], '朝鲜半岛'],
	[['taiwan'], '台湾'],
	[['india'], '印度'],
	[['sri lanka'], '斯里兰卡'],
	[['himalaya'], '喜马拉雅'],
	[['southeast asia'], '东南亚'],
	[['asia'], '亚洲'],
	[['thailand'], '泰国'],
	[['vietnam'], '越南'],
	[['philippines'], '菲律宾'],
	[['malaysia'], '马来西亚'],
	[['indonesia'], '印度尼西亚'],
	[['new guinea'], '新几内亚'],
	[['australia'], '澳大利亚'],
	[['mediterranean'], '地中海地区'],
	[['europe'], '欧洲'],
	[['north america'], '北美洲'],
	[['usa', 'united states', 'america'], '美国'],
	[['canada'], '加拿大'],
	[['caribbean'], '加勒比地区'],
	[['hawaii'], '夏威夷'],
	[['middle east'], '中东'],
	[['arabia'], '阿拉伯半岛'],
	[['turkey'], '土耳其'],
	[['iran'], '伊朗'],
];

export function zhOrigin(v?: string): string {
	if (!v) return '';
	const s = v.toLowerCase();
	for (const [keys, zh] of ORIGIN_RULES) {
		if (keys.some((k) => s.includes(k))) return zh;
	}
	return v;
}
