import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { AntDesignVueResolver } from 'unplugin-vue-components/resolvers'
import svgLoader from 'vite-svg-loader'
const rootPath = new URL('.', import.meta.url).pathname
// https://vitejs.dev/config/
export default defineConfig({
    server: {
        host: '127.0.0.1', // 使用 IPv4
        port: 34115
    },
    plugins: [
        vue(),
        svgLoader(),
        AutoImport({
            resolvers: [AntDesignVueResolver()],
        }),
        Components({
            resolvers: [AntDesignVueResolver({ importStyle: false })],
        }),
    ],
   
    resolve: {
        alias: {
            '@': rootPath + 'src',
            'wailsjs': rootPath + 'wailsjs',
        },
    },
})
