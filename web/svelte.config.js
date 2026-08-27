import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	compilerOptions: {
		// 全局强制 Svelte 5 runes 模式，锁定迁移成果，防止误写遗留语法而不报错
		runes: true
	},
	kit: {
		// 单镜像自部署：构建为静态 SPA，由后端 Go 直接托管
		adapter: adapter({
			pages: 'build',
			assets: 'build',
			fallback: 'index.html',
			precompress: false,
			strict: false
		})
	}
};

export default config;
