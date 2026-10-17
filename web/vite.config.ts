import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { VitePWA } from 'vite-plugin-pwa';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [
    // SvelteKit 3 起配置统一切入 sveltekit() 插件（svelte.config.js 已移除）
    sveltekit({
      preprocess: vitePreprocess(),
      compilerOptions: {
        // 全局强制 Svelte 5 runes 模式，锁定迁移成果，防止误写遗留语法而不报错
        runes: true
      },
      // 单镜像自部署：构建为静态 SPA，由后端 Go 直接托管
      adapter: adapter({
        pages: 'build',
        assets: 'build',
        fallback: 'index.html',
        precompress: false,
        strict: false
      })
    }),
    ...VitePWA({
      // 仅缓存壳页与静态资源，不缓存 POST，避免数据不一致
      registerType: 'autoUpdate',
      // sw.js 必须落进 kit 的 client 产物目录，才能被 adapter-static 一并复制进 build/
      outDir: '.svelte-kit/output/client',
      includeAssets: [
        'favicon.ico',
        'favicon.svg',
        'icon-192.png',
        'icon-512.png',
        'icon-maskable-192.png',
        'icon-maskable-512.png',
        'apple-touch-icon.png'
      ],
      manifest: {
        id: '/',
        name: '青野集 · 家庭园艺植物记录与养护',
        short_name: '青野集',
        description: '家庭园艺植物记录与养护工具',
        lang: 'zh-CN',
        dir: 'ltr',
        scope: '/',
        theme_color: '#43a047',
        background_color: '#f7faf5',
        display: 'standalone',
        display_override: ['standalone', 'minimal-ui'],
        orientation: 'portrait-primary',
        categories: ['lifestyle', 'productivity'],
        start_url: '/',
        icons: [
          { src: 'icon-192.png', sizes: '192x192', type: 'image/png', purpose: 'any' },
          { src: 'icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'any' },
          {
            src: 'icon-maskable-192.png',
            sizes: '192x192',
            type: 'image/png',
            purpose: 'maskable'
          },
          {
            src: 'icon-maskable-512.png',
            sizes: '512x512',
            type: 'image/png',
            purpose: 'maskable'
          },
          { src: 'favicon.svg', sizes: 'any', type: 'image/svg+xml', purpose: 'any' }
        ]
      },
      workbox: {
        // kit 3 下 @vite-pwa/sveltekit 不再生成 sw（其 closeBundle 挂在已消失的 SSR 构建阶段），
        // 改由 vite-plugin-pwa 直连：globDirectory 直接指向 client 产物目录，pattern 去掉 client/ 前缀
        globDirectory: '.svelte-kit/output/client',
        globPatterns: ['**/*.{js,css,ico,png,svg,webp,webmanifest}'],
        // API 与上传走网络
        navigateFallbackDenylist: [/^\/api/],
        runtimeCaching: [
          {
            // 业务数据 GET：离线时从缓存读取（stale-while-revalidate）
            urlPattern: ({ url, request }) =>
              url.pathname.startsWith('/api/') && request.method === 'GET',
            handler: 'StaleWhileRevalidate',
            options: {
              cacheName: 'api-get',
              expiration: { maxEntries: 300, maxAgeSeconds: 60 * 60 * 24 * 30 },
              cacheableResponse: { statuses: [200] }
            }
          },
          {
            // 上传的植物/日记图片：离线可读
            urlPattern: ({ url }) => url.pathname.startsWith('/uploads/'),
            handler: 'CacheFirst',
            options: {
              cacheName: 'uploads',
              expiration: { maxEntries: 200, maxAgeSeconds: 60 * 60 * 24 * 30 },
              cacheableResponse: { statuses: [200] }
            }
          }
        ]
      },
      devOptions: { enabled: false }
    })
  ],
  server: {
    port: 5173,
    proxy: {
      // 开发期将 API 与照片请求代理到后端（用 127.0.0.1 避免 localhost 解析歧义）
      '/api': { target: 'http://127.0.0.1:8081', changeOrigin: true },
      '/uploads': { target: 'http://127.0.0.1:8081', changeOrigin: true }
    }
  },
  test: {
    include: ['src/**/*.{test,spec}.ts'],
    environment: 'node'
  }
});
