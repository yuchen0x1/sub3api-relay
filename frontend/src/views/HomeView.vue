<template>
  <div v-if="hasHomeContent" class="min-h-screen">
    <iframe v-if="homeContentUrl" :src="homeContentUrl" class="h-screen w-full border-0" sandbox="allow-forms allow-modals allow-popups allow-popups-to-escape-sandbox allow-presentation allow-same-origin allow-scripts" referrerpolicy="no-referrer" allowfullscreen />
    <div v-else v-html="sanitizedHomeContent" />
  </div>

  <div v-else-if="compactHomeEnabled" data-testid="compact-home" class="sub3-public-shell flex min-h-screen flex-col">
    <header class="sub3-glass-header">
      <div class="mx-auto flex w-full max-w-6xl items-center justify-between gap-4 px-5 py-4">
        <BrandMark :site-logo="siteLogo" :site-name="siteName" />
        <div class="flex items-center gap-2">
          <LocaleSwitcher />
          <ThemeButton :is-dark="isDark" @toggle="toggleTheme" />
          <router-link :to="entryPath" class="btn btn-primary">{{ entryLabel }}</router-link>
        </div>
      </div>
    </header>
    <main class="flex flex-1 items-center justify-center px-5 py-20">
      <section class="sub3-glass-panel w-full max-w-2xl p-8 text-center sm:p-12">
        <BrandMark :site-logo="siteLogo" :site-name="siteName" centered />
        <p class="mx-auto mt-6 max-w-xl text-lg text-token-text-secondary">{{ siteSubtitle || t('home.heroSubtitle') }}</p>
        <router-link :to="entryPath" class="btn btn-primary mt-8">{{ entryLabel }} <Icon name="arrowRight" size="md" /></router-link>
      </section>
    </main>
  </div>

  <div v-else data-testid="default-home" class="min-h-screen w-full overflow-x-hidden bg-[#0d1321]">
    <span class="sr-only">{{ siteName }}</span>
    <Sub3LandingView />
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import DOMPurify from 'dompurify'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import Sub3LandingView from '@/components/home/Sub3LandingView.vue'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub3API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || '')
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const homeContentUrl = computed(() => {
  const value = homeContent.value.trim()
  if (!/^https:\/\//i.test(value)) return ''
  return sanitizeUrl(value)
})
const sanitizedHomeContent = computed(() => DOMPurify.sanitize(homeContent.value, {
  USE_PROFILES: { html: true },
}))
const isDark = computed(() => appStore.isDark)
const isAuthenticated = computed(() => authStore.isAuthenticated)
const entryPath = computed(() => isAuthenticated.value ? (authStore.isAdmin ? '/admin/dashboard' : '/dashboard') : '/login')
const entryLabel = computed(() => isAuthenticated.value ? t('home.goToDashboard') : t('home.login'))
function toggleTheme() {
  appStore.toggleTheme()
}
const BrandMark = defineComponent({
  props: { siteLogo: String, siteName: { type: String, required: true }, centered: Boolean },
  setup(props) {
    return () => h('div', { class: ['flex items-center gap-3', props.centered && 'justify-center'] }, [
      h('img', { src: props.siteLogo || '/logo.svg', alt: props.siteName, class: 'h-11 w-11 object-contain' }),
      h('span', { class: 'text-lg font-semibold tracking-tight' }, props.siteName),
    ])
  },
})
const ThemeButton = defineComponent({
  props: { isDark: Boolean },
  emits: ['toggle'],
  setup(props, { emit }) {
    return () => h('button', {
      type: 'button',
      class: 'sub3-icon-button',
      title: props.isDark ? t('home.switchToLight') : t('home.switchToDark'),
      onClick: () => emit('toggle'),
    }, [h(Icon, { name: props.isDark ? 'sun' : 'moon', size: 'md' })])
  },
})
onMounted(() => {
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) appStore.fetchPublicSettings()
})
</script>
