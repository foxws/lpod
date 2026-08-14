// @ts-check
const { themes: prismThemes } = require('prism-react-renderer');

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'lpod',
  tagline: 'A single bash script for managing Podman Quadlet services on the host',

  url: 'https://foxws.github.io',
  baseUrl: '/lpod/',

  organizationName: 'foxws',
  projectName: 'lpod',
  deploymentBranch: 'gh-pages',
  trailingSlash: false,

  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          path: '../docs',
          routeBasePath: '/',
          sidebarPath: require.resolve('./sidebars.js'),
          editUrl: 'https://github.com/foxws/lpod/edit/main/docs/',
        },
        blog: false,
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      }),
    ],
  ],

  themes: [
    [
      '@easyops-cn/docusaurus-search-local',
      /** @type {import('@easyops-cn/docusaurus-search-local').PluginOptions} */
      ({
        hashed: true,
        indexBlog: false,
        docsRouteBasePath: '/',
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      navbar: {
        title: 'lpod',
        items: [
          {
            href: 'https://github.com/foxws/lpod',
            label: 'GitHub',
            position: 'right',
          },
          {
            href: 'https://github.com/foxws/lpod/releases',
            label: 'Releases',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Docs',
            items: [
              { label: 'Introduction', to: '/' },
              { label: 'Installation', to: '/installation' },
              { label: 'Commands', to: '/commands' },
            ],
          },
          {
            title: 'More',
            items: [
              { label: 'GitHub', href: 'https://github.com/foxws/lpod' },
              { label: 'Releases', href: 'https://github.com/foxws/lpod/releases' },
              { label: 'laravel-podman', href: 'https://github.com/foxws/laravel-podman' },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} foxws. Built with Docusaurus.`,
      },
      prism: {
        theme: prismThemes.github,
        darkTheme: prismThemes.dracula,
      },
    }),
};

module.exports = config;
