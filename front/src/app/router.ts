import { createRouter, createWebHistory } from 'vue-router'
import StreamersPage from '../pages/StreamersPage.vue'

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/pipeline' },
    { path: '/streamers', component: StreamersPage },
    {
      path: '/search',
      component: () => import('../pages/ClipDiscoveryPage.vue'),
      children: [
        { path: '', redirect: { name: 'clip-subscriptions' } },
        {
          path: 'subscriptions',
          name: 'clip-subscriptions',
          component: () => import('../pages/ClipSubscriptionsPage.vue'),
        },
        {
          path: 'clips',
          name: 'clip-search',
          component: () => import('../pages/ClipsPage.vue'),
        },
      ],
    },
    { path: '/clips', redirect: '/search/clips' },
    { path: '/pipeline', component: () => import('../pages/PipelinePage.vue') },
    { path: '/assets', component: () => import('../pages/AssetsPage.vue') },
    {
      path: '/accounts',
      component: () => import('../pages/AccountManagerPage.vue'),
    },
    {
      path: '/templates',
      component: () => import('../pages/TemplatesPage.vue'),
    },
    {
      path: '/templates/:id',
      component: () => import('../pages/TemplateEditorPage.vue'),
    },
  ],
})
