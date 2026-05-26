import {defineConfig} from 'vite'

export default defineConfig({
    base: '',
    build: {
        rollupOptions: {
            input: {
                main: 'index.html',
                user: 'user/index.html',
                analyze: 'analyze/index.html'
            }
        }
    }
})
