import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  integrations: [
    starlight({
      title: 'Stencil',
      description: 'A Go library for building CLI applications.',
      social: {
        github: 'https://github.com/bgrewell/stencil',
      },
      sidebar: [
        { label: 'Start here', items: [{ label: 'Introduction', slug: 'index' }] },
      ],
    }),
  ],
});
