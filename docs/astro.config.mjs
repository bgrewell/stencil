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
        {
          label: 'Guides',
          items: [
            { label: 'Getting started', slug: 'guides/getting-started' },
            { label: 'Commands and flags', slug: 'guides/commands-and-flags' },
            { label: 'Config files and provenance', slug: 'guides/config' },
            { label: 'Console UI', slug: 'guides/ui' },
            { label: 'Developer CLI', slug: 'guides/developer-cli' },
          ],
        },
        { label: 'Project', items: [{ label: 'Roadmap', slug: 'roadmap' }] },
      ],
    }),
  ],
});
