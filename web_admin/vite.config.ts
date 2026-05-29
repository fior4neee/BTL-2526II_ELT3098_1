import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		proxy: {
			'/api/v1': {
				target: 'http://localhost:8081',
				changeOrigin: true,
				secure: false,
			},
		},
	},
});
