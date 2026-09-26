import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'
import { fileURLToPath } from 'url'
import vueDevTools from 'vite-plugin-vue-devtools'
import viteCompression from 'vite-plugin-compression'
import Components from 'unplugin-vue-components/vite'
import AutoImport from 'unplugin-auto-import/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import tailwindcss from '@tailwindcss/vite'
// import { visualizer } from 'rollup-plugin-visualizer'

export default ({ mode }: { mode: string }) => {
  const root = process.cwd()
  const env = loadEnv(mode, root)
  const { VITE_VERSION, VITE_PORT, VITE_BASE_URL, VITE_API_URL, VITE_API_PROXY_URL } = env

  console.log(`🚀 API_URL = ${VITE_API_URL}`)
  console.log(`🚀 VERSION = ${VITE_VERSION}`)

  return defineConfig({
    define: {
      __APP_VERSION__: JSON.stringify(VITE_VERSION)
    },
    base: VITE_BASE_URL,
    server: {
      port: Number(VITE_PORT),
      proxy: {
        '/api': {
          target: VITE_API_PROXY_URL,
          changeOrigin: true,
          rewrite: (path) => path.replace(new RegExp('^' + VITE_API_URL), '')
        }
      },
      host: true,
      // 开发服务器启动后预热「首屏必经」的几个文件，避免打开就白屏等编译。
      //
      // 不要用 './src/views/**/*.vue' 这种全量写法：78 个页面会在启动瞬间一起进
      // 预转换队列，把 Vite 的 transform 流水线占满，开发者真正请求的那个页面反而要
      // 排队等几十秒（实测热启动首个请求 50s）。按 Vite 文档的建议只列常用文件即可，
      // 其余页面在导航时按需转换，单页成本只有几百毫秒。
      warmup: {
        clientFiles: [
          './src/App.vue',
          './src/views/index/index.vue',
          './src/views/auth/login/index.vue',
          './src/views/dashboard/console/index.vue'
        ]
      }
    },
    // 路径别名
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
        '@views': resolvePath('src/views'),
        '@imgs': resolvePath('src/assets/images'),
        '@icons': resolvePath('src/assets/icons'),
        '@utils': resolvePath('src/utils'),
        '@stores': resolvePath('src/store'),
        '@styles': resolvePath('src/assets/styles')
      }
    },
    build: {
      target: 'es2015',
      outDir: 'dist',
      chunkSizeWarningLimit: 2000,
      minify: 'terser',
      terserOptions: {
        compress: {
          // 生产环境去除 console
          drop_console: true,
          // 生产环境去除 debugger
          drop_debugger: true
        }
      },
      dynamicImportVarsOptions: {
        warnOnError: true,
        exclude: [],
        include: ['src/views/**/*.vue']
      }
    },
    plugins: [
      vue(),
      tailwindcss(),
      // 自动按需导入 API
      AutoImport({
        imports: ['vue', 'vue-router', 'pinia', '@vueuse/core'],
        dts: 'src/types/import/auto-imports.d.ts',
        // importStyle: false —— 样式统一由 src/assets/styles/core/el-theme.scss 全量引入，
        // 原因见该文件里的说明
        resolvers: [ElementPlusResolver({ importStyle: false })],
        eslintrc: {
          enabled: true,
          filepath: './.auto-import.json',
          globalsPropValue: true
        }
      }),
      // 自动按需导入组件
      Components({
        dts: 'src/types/import/components.d.ts',
        resolvers: [ElementPlusResolver({ importStyle: false })]
      }),
      // 压缩
      viteCompression({
        verbose: false, // 是否在控制台输出压缩结果
        disable: false, // 是否禁用
        algorithm: 'gzip', // 压缩算法
        ext: '.gz', // 压缩后的文件名后缀
        threshold: 10240, // 只有大小大于该值的资源会被处理 10240B = 10KB
        deleteOriginFile: false // 压缩后是否删除原文件
      }),
      vueDevTools()
      // 打包分析
      // visualizer({
      //   open: true,
      //   gzipSize: true,
      //   brotliSize: true,
      //   filename: 'dist/stats.html' // 分析图生成的文件名及路径
      // }),
    ],
    // 依赖预构建：避免运行时重复请求与转换，提升首次加载速度
    //
    // 重要：路由组件通过 router/core/ComponentLoader.ts 里的
    // `import.meta.glob('../../views/**/*.vue')` 加载，而 Vite 的依赖扫描器（esbuild）
    // 无法穿透 import.meta.glob，因此 src/views 下的页面对启动时的扫描是「不可见」的。
    // 结果是：只被页面组件用到的第三方依赖（md-editor-v3、wangEditor、highlight.js 等）
    // 全部要等到用户第一次导航到该页面时才被发现，届时 Vite 会现场 esbuild 预构建
    // （页面卡住），完成后通过 HMR 强制 full-reload；而此时异步组件还没 resolve，
    // 路由的 hash 仍停留在上一个页面，于是刷新后就「退回到之前的页面」。
    //
    // 因此这里必须显式把页面文件加入 entries 让扫描器能看到它们，
    // 并把重依赖列进 include 兜底。
    optimizeDeps: {
      entries: ['index.html', 'src/views/**/*.vue', 'src/components/**/*.vue'],
      include: [
        'echarts/core',
        'echarts/charts',
        'echarts/components',
        'echarts/renderers',
        'xlsx',
        'xgplayer',
        'crypto-js',
        'file-saver',
        'vue-img-cutter',
        'element-plus/es',
        'element-plus/es/locale/lang/zh-cn',
        'element-plus/es/locale/lang/en',
        '@element-plus/icons-vue',
        // 以下依赖只在懒加载页面/组件中使用，必须显式声明，否则会在运行时触发
        // 「new dependencies optimized → reloading」造成卡顿 + 页面回退
        'md-editor-v3',
        '@wangeditor/editor',
        '@wangeditor/editor-for-vue',
        'highlight.js',
        'highlight.js/lib/core',
        '@iconify/vue',
        'vue-draggable-plus',
        'spark-md5',
        'mitt',
        'axios',
        'nprogress',
        'vue-i18n',
        '@vueuse/core',
        'pinia',
        'pinia-plugin-persistedstate'
      ]
    },
    css: {
      preprocessorOptions: {
        // sass variable and mixin
        //
        // 这里只前置项目自己的 mixin（157 行，很便宜）。
        // el-light.scss 不放这里：它 `@forward` 了 element-plus 的 common/var.scss
        // （几百行、多个大 map + 函数），一旦进 additionalData，每个 .vue 的
        // <style lang="scss"> 块、每个全局样式文件都要重新求值一遍，实测开发服务器
        // 预热 78 个页面时会因此多花几十秒。
        // Element Plus 的变量覆盖只有 el-theme.scss / el-dark.scss 需要，
        // 已改为在 src/assets/styles/index.scss 顶部显式 `@use` 一次。
        scss: {
          additionalData: `@use "@styles/core/mixin.scss" as *;`
        }
      },
      postcss: {
        plugins: [
          {
            postcssPlugin: 'internal:charset-removal',
            AtRule: {
              charset: (atRule) => {
                if (atRule.name === 'charset') {
                  atRule.remove()
                }
              }
            }
          }
        ]
      }
    }
  })
}

function resolvePath(paths: string) {
  return path.resolve(__dirname, paths)
}
