// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import tailwindcss from '@tailwindcss/vite';

// https://astro.build/config
export default defineConfig({
  site: 'https://0funct0ry.github.io',
  base: '/maelsink',
  integrations: [
    starlight({
      title: 'maelsink',
      logo: {
        src: './src/assets/maelsink-logo.svg',
        alt: 'maelsink',
        replacesTitle: false,
      },
      customCss: ['./src/styles/starlight-tokens.css'],
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/0funct0ry/maelsink' },
      ],
      sidebar: [
        { label: 'Getting Started', slug: 'docs/getting-started' },
        { label: 'Features', slug: 'docs/features' },
        {
          label: 'Installation',
          items: [
            { label: 'Installation', slug: 'docs/installation' },
            { label: 'Building from Source', slug: 'docs/installation/building-from-source' },
            { label: 'Testing maelsink', slug: 'docs/installation/testing-maelsink' },
          ],
        },
        {
          label: 'Configuration',
          items: [
            { label: 'Runtime Options', slug: 'docs/configuration/runtime-options' },
            { label: 'Web UI and API Server', slug: 'docs/configuration/web-ui-and-api-server' },
            { label: 'SMTP Server', slug: 'docs/configuration/smtp-server' },
            { label: 'TLS Certificates', slug: 'docs/configuration/tls-certificates' },
            { label: 'Password Files', slug: 'docs/configuration/password-files' },
          ],
        },
        {
          label: 'CLI Reference',
          items: [{ autogenerate: { directory: 'docs/cli-reference' } }],
        },
        { label: 'REST API Reference', slug: 'docs/rest-api-reference' },
        {
          label: 'Usage',
          items: [
            { label: 'Filters and Search', slug: 'docs/usage/filters-and-search' },
            { label: 'Advanced Search Patterns', slug: 'docs/usage/advanced-search-patterns' },
            { label: 'Deleting Messages', slug: 'docs/usage/deleting-messages' },
            { label: 'Tagging Messages', slug: 'docs/usage/tagging-messages' },
            { label: 'SMTP Sessions', slug: 'docs/usage/smtp-sessions' },
            {
              label: 'Sending Mail',
              items: [
                { label: 'Programmatically', slug: 'docs/usage/sending-mail/programmatically' },
                { label: 'Via CLI Commands', slug: 'docs/usage/sending-mail/via-cli-commands' },
                { label: 'Via Shell', slug: 'docs/usage/sending-mail/via-shell' },
                { label: 'Via Composer UI', slug: 'docs/usage/sending-mail/via-composer-ui' },
              ],
            },
            { label: 'Export', slug: 'docs/usage/export' },
            { label: 'Using CLI', slug: 'docs/usage/using-cli' },
            { label: 'Using Shell', slug: 'docs/usage/using-shell' },
            { label: 'Using Composer', slug: 'docs/usage/using-composer' },
          ],
        },
        { label: 'Integration Testing', slug: 'docs/integration-testing' },
        { label: 'Shell Builtin Reference', slug: 'docs/shell-builtin-reference' },
        { label: 'Shell Functions Reference', slug: 'docs/shell-functions-reference' },
        {
          label: 'Guides',
          items: [{ label: 'Running maelsink behind HAProxy', slug: 'docs/guides/haproxy' }],
        },
        { label: 'Screenshots', slug: 'docs/screenshots' },
      ],
    }),
  ],

  vite: {
    plugins: [tailwindcss()],
  },
});
