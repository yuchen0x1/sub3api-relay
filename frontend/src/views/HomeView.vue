<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="hasHomeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Compact Home Page -->
  <div
    v-else-if="compactHomeEnabled"
    data-testid="compact-home"
    class="flex min-h-screen flex-col bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white"
  >
    <header class="border-b border-gray-200 px-4 py-4 sm:px-6 dark:border-dark-800">
      <nav class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <img
            :src="siteLogo || '/logo.svg'"
            alt="Logo"
            class="h-9 w-9 shrink-0 rounded-lg object-contain"
          />
          <span class="min-w-0 truncate text-base font-semibold">{{ siteName }}</span>
        </div>
        <div class="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="t('home.viewDocs')"
            :aria-label="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <button
            type="button"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            :aria-label="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="flex min-w-0 flex-1 items-center justify-center px-4 py-16 sm:px-6">
      <div class="min-w-0 max-w-2xl text-center">
        <img
          :src="siteLogo || '/logo.svg'"
          alt="Logo"
          class="mx-auto mb-6 h-20 w-20 rounded-2xl object-contain"
        />
        <h1 class="[overflow-wrap:anywhere] text-3xl font-bold md:text-4xl">{{ siteName }}</h1>
        <p class="mt-4 whitespace-pre-wrap [overflow-wrap:anywhere] text-base text-gray-600 dark:text-dark-300">{{ siteSubtitle }}</p>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="mt-8 inline-flex min-h-10 items-center justify-center rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-700"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.login') }}
        </router-link>
      </div>
    </main>

    <footer class="min-w-0 border-t border-gray-200 px-4 py-5 text-center text-sm text-gray-500 [overflow-wrap:anywhere] sm:px-6 dark:border-dark-800 dark:text-dark-400">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>

  <!-- Default Home Page -->
  <div
    v-else
    data-testid="default-home"
    class="flex min-h-screen flex-col bg-token-bg text-token-text"
  >
    <!-- Header -->
    <header class="sticky top-0 z-20 border-b border-token-border bg-token-bg/95 backdrop-blur-sm">
      <nav class="mx-auto flex h-16 max-w-6xl items-center justify-between gap-3 px-4 sm:px-6">
        <!-- Brand -->
        <div class="flex min-w-0 items-center gap-3">
          <img
            :src="siteLogo || '/logo.svg'"
            :alt="siteName"
            class="h-9 w-9 shrink-0 object-contain"
          />
          <span class="min-w-0 truncate text-base font-semibold">{{ siteName }}</span>
        </div>

        <!-- Actions -->
        <div class="flex shrink-0 items-center gap-1 sm:gap-2">
          <LocaleSwitcher />

          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 items-center justify-center rounded-lg text-token-muted transition-colors hover:bg-token-surface-2 hover:text-token-text"
            :title="t('home.viewDocs')"
            :aria-label="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>

          <button
            type="button"
            @click="toggleTheme"
            class="flex h-10 w-10 items-center justify-center rounded-lg text-token-muted transition-colors hover:bg-token-surface-2 hover:text-token-text"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            :aria-label="isDark ? t('home.switchToLight') : t('home.switchToDark')"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>

          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex h-10 shrink-0 items-center justify-center gap-1.5 rounded-lg bg-token-text px-4 text-sm font-medium text-token-text-inverse transition-opacity hover:opacity-90"
          >
            <Icon v-if="!isAuthenticated" name="login" size="sm" />
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <!-- Main Content -->
    <main class="flex-1">
      <div class="mx-auto w-full max-w-6xl px-4 py-10 sm:px-6 md:py-14">
        <!-- Hero -->
        <section class="mx-auto max-w-3xl text-center">
          <img
            :src="siteLogo || '/logo.svg'"
            :alt="siteName"
            class="mx-auto h-16 w-16 object-contain md:h-20 md:w-20"
          />
          <h1 class="mt-6 [overflow-wrap:anywhere] text-3xl font-bold tracking-tight md:text-5xl">
            {{ siteName }}
          </h1>
          <p class="mt-3 text-lg text-token-text-secondary md:text-xl">
            {{ siteSubtitle || t('home.heroSubtitle') }}
          </p>
          <p class="mx-auto mt-3 max-w-2xl text-sm text-token-muted md:text-base">
            {{ t('home.heroDescription') }}
          </p>

          <div class="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
            <router-link
              :to="isAuthenticated ? dashboardPath : '/login'"
              class="btn btn-primary inline-flex h-11 min-w-44 items-center justify-center gap-2 px-6 text-base"
            >
              {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
              <Icon name="arrowRight" size="md" :stroke-width="2" />
            </router-link>
            <a
              v-if="docUrl"
              :href="docUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="inline-flex h-11 items-center justify-center gap-2 rounded-lg border border-token-border px-5 text-sm font-medium text-token-text-secondary transition-colors hover:bg-token-surface-2"
            >
              <Icon name="book" size="sm" />
              {{ t('home.viewDocs') }}
            </a>
          </div>

          <ul class="mt-8 flex flex-wrap items-center justify-center gap-2">
            <li
              class="inline-flex items-center gap-1.5 rounded-full border border-token-border bg-token-surface px-3 py-1.5 text-xs font-medium text-token-text-secondary"
            >
              <Icon name="swap" size="sm" class="text-primary-600 dark:text-primary-400" />
              {{ t('home.tags.subscriptionToApi') }}
            </li>
            <li
              class="inline-flex items-center gap-1.5 rounded-full border border-token-border bg-token-surface px-3 py-1.5 text-xs font-medium text-token-text-secondary"
            >
              <Icon name="shield" size="sm" class="text-primary-600 dark:text-primary-400" />
              {{ t('home.tags.stickySession') }}
            </li>
            <li
              class="inline-flex items-center gap-1.5 rounded-full border border-token-border bg-token-surface px-3 py-1.5 text-xs font-medium text-token-text-secondary"
            >
              <Icon name="chart" size="sm" class="text-primary-600 dark:text-primary-400" />
              {{ t('home.tags.realtimeBilling') }}
            </li>
          </ul>
        </section>

        <!-- Gateway Contract -->
        <section class="mx-auto mt-12 max-w-3xl md:mt-16" aria-labelledby="gateway-heading">
          <div class="mb-4 text-center">
            <h2 id="gateway-heading" class="text-xl font-semibold md:text-2xl">
              {{ t('home.gateway.title') }}
            </h2>
            <p class="mt-1 text-sm text-token-muted">{{ t('home.gateway.description') }}</p>
          </div>

          <div class="overflow-hidden rounded-xl border border-token-border bg-token-surface shadow-card">
            <div class="flex items-center justify-between gap-2 border-b border-token-border px-4 py-2.5">
              <div class="flex min-w-0 items-center gap-2 font-mono text-xs text-token-muted">
                <Icon name="terminal" size="sm" class="shrink-0 text-primary-600 dark:text-primary-400" />
                <span class="truncate">POST /v1/messages</span>
              </div>
              <span
                class="inline-flex shrink-0 items-center gap-1 rounded-md bg-token-success/10 px-2 py-0.5 text-xs font-medium text-token-success"
              >
                <Icon name="check" size="xs" />
                200 OK
              </span>
            </div>

            <div class="border-b border-token-border">
              <p class="px-4 pt-3 text-[11px] font-semibold uppercase tracking-wide text-token-muted">
                {{ t('home.gateway.request') }}
              </p>
              <pre
                class="overflow-x-auto px-4 py-3 font-mono text-xs leading-relaxed text-token-text-secondary"
                aria-label="curl POST /v1/messages"
                ><code>{{ requestExample }}</code></pre
              >
            </div>

            <div>
              <p class="px-4 pt-3 text-[11px] font-semibold uppercase tracking-wide text-token-muted">
                {{ t('home.gateway.response') }}
              </p>
              <pre
                class="overflow-x-auto px-4 py-3 font-mono text-xs leading-relaxed text-token-text-secondary"
                aria-label="POST /v1/messages response"
                ><code>{{ responseExample }}</code></pre
              >
            </div>
          </div>

          <ul class="mt-4 grid gap-2 sm:grid-cols-2">
            <li
              class="flex items-start gap-2.5 rounded-lg border border-token-border bg-token-surface px-3.5 py-3"
            >
              <Icon name="chat" size="sm" class="mt-0.5 shrink-0 text-primary-600 dark:text-primary-400" />
              <div class="min-w-0">
                <p class="text-sm font-medium">{{ t('home.gateway.chat') }}</p>
                <p class="mt-0.5 text-xs leading-relaxed text-token-muted">
                  {{ t('home.gateway.chatDesc') }}
                </p>
              </div>
            </li>
            <li
              class="flex items-start gap-2.5 rounded-lg border border-token-border bg-token-surface px-3.5 py-3"
            >
              <Icon name="cube" size="sm" class="mt-0.5 shrink-0 text-primary-600 dark:text-primary-400" />
              <div class="min-w-0">
                <p class="text-sm font-medium">{{ t('home.gateway.models') }}</p>
                <p class="mt-0.5 text-xs leading-relaxed text-token-muted">
                  {{ t('home.gateway.modelsDesc') }}
                </p>
              </div>
            </li>
            <li
              class="flex items-start gap-2.5 rounded-lg border border-token-border bg-token-surface px-3.5 py-3"
            >
              <Icon name="bolt" size="sm" class="mt-0.5 shrink-0 text-primary-600 dark:text-primary-400" />
              <div class="min-w-0">
                <p class="text-sm font-medium">{{ t('home.gateway.streaming') }}</p>
                <p class="mt-0.5 text-xs leading-relaxed text-token-muted">
                  {{ t('home.gateway.streamingDesc') }}
                </p>
              </div>
            </li>
            <li
              class="flex items-start gap-2.5 rounded-lg border border-token-border bg-token-surface px-3.5 py-3"
            >
              <Icon name="key" size="sm" class="mt-0.5 shrink-0 text-primary-600 dark:text-primary-400" />
              <div class="min-w-0">
                <p class="text-sm font-medium">{{ t('home.gateway.auth') }}</p>
                <p class="mt-0.5 text-xs leading-relaxed text-token-muted">
                  {{ t('home.gateway.authDesc') }}
                </p>
              </div>
            </li>
          </ul>
        </section>

        <!-- Features -->
        <section class="mx-auto mt-12 max-w-3xl md:mt-16">
          <h2 class="text-center text-xl font-semibold md:text-2xl">
            {{ t('home.solutions.title') }}
          </h2>
          <p class="mt-1 text-center text-sm text-token-muted">
            {{ t('home.solutions.subtitle') }}
          </p>
          <div class="mt-6 grid gap-4 md:grid-cols-3">
            <div
              v-for="feature in featureItems"
              :key="feature.icon"
              class="rounded-xl border border-token-border bg-token-surface p-5"
            >
              <Icon
                :name="feature.icon"
                size="lg"
                class="text-primary-600 dark:text-primary-400"
              />
              <h3 class="mt-3 text-base font-semibold">{{ feature.title }}</h3>
              <p class="mt-1.5 text-sm leading-relaxed text-token-muted">{{ feature.desc }}</p>
            </div>
          </div>
        </section>

        <!-- Providers -->
        <section class="mx-auto mt-12 max-w-3xl md:mt-16">
          <h2 class="text-center text-xl font-semibold md:text-2xl">
            {{ t('home.providers.title') }}
          </h2>
          <p class="mt-1 text-center text-sm text-token-muted">
            {{ t('home.providers.description') }}
          </p>
          <ul class="mt-6 flex flex-wrap items-center justify-center gap-2">
            <li
              v-for="provider in providerItems"
              :key="provider.name"
              class="inline-flex items-center gap-1.5 rounded-lg border border-token-border bg-token-surface px-3 py-2 text-sm text-token-text-secondary"
            >
              <Icon name="check" size="xs" class="text-token-success" />
              {{ provider.name }}
              <span
                class="rounded bg-token-success/10 px-1.5 py-0.5 text-[10px] font-medium text-token-success"
                >{{ t('home.providers.supported') }}</span
              >
            </li>
            <li
              class="inline-flex items-center gap-1.5 rounded-lg border border-token-border bg-token-surface px-3 py-2 text-sm opacity-70"
            >
              <Icon name="more" size="xs" class="text-token-muted" />
              <span class="text-token-text-secondary">{{ t('home.providers.more') }}</span>
              <span
                class="rounded bg-token-surface-2 px-1.5 py-0.5 text-[10px] font-medium text-token-muted"
                >{{ t('home.providers.soon') }}</span
              >
            </li>
          </ul>
        </section>

        <!-- Bottom CTA -->
        <section class="mx-auto mt-12 max-w-3xl text-center md:mt-16">
          <h2 class="text-xl font-semibold md:text-2xl">{{ t('home.cta.title') }}</h2>
          <p class="mt-2 text-sm text-token-muted">{{ t('home.cta.description') }}</p>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="btn btn-primary mt-6 inline-flex h-11 items-center justify-center gap-2 px-6"
          >
            {{ isAuthenticated ? t('home.goToDashboard') : t('home.cta.button') }}
            <Icon name="arrowRight" size="md" :stroke-width="2" />
          </router-link>
        </section>
      </div>
    </main>

    <!-- Footer -->
    <footer class="border-t border-token-border px-4 py-6 sm:px-6">
      <div
        class="mx-auto flex max-w-6xl flex-col items-center justify-between gap-3 text-center sm:flex-row sm:text-left"
      >
        <p class="min-w-0 text-sm text-token-muted [overflow-wrap:anywhere]">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
        <div class="flex shrink-0 items-center gap-4">
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-sm text-token-muted transition-colors hover:text-token-text"
          >
            {{ t('home.docs') }}
          </a>
          <a
            :href="githubUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="text-sm text-token-muted transition-colors hover:text-token-text"
          >
            {{ t('home.viewOnGithub') }}
          </a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub3API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || '')
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)

// Check if homeContent is a URL (for iframe display)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

// Theme
const isDark = computed(() => appStore.isDark)

// GitHub URL
const githubUrl = 'https://github.com/nhzhongguo/sub3api-relay'

// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())

// Real gateway origin for the request example (same-origin deployment)
const apiOrigin = typeof window !== 'undefined' ? window.location.origin : ''

// Static, real API contract examples (Anthropic-compatible Messages API)
const requestExample = computed(() => `curl -X POST ${apiOrigin}/v1/messages \\
  -H "Authorization: Bearer sk-..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "your-model-id",
    "max_tokens": 1024,
    "messages": [
      { "role": "user", "content": "Hello" }
    ]
  }'`)

const responseExample = computed(() => `{
  "id": "msg_01...",
  "type": "message",
  "role": "assistant",
  "content": [
    { "type": "text", "text": "Hi! How can I help?" }
  ],
  "stop_reason": "end_turn",
  "usage": {
    "input_tokens": 8,
    "output_tokens": 9
  }
}`)

const featureItems = computed(() => [
  { icon: 'server', title: t('home.features.unifiedGateway'), desc: t('home.features.unifiedGatewayDesc') },
  { icon: 'users', title: t('home.features.multiAccount'), desc: t('home.features.multiAccountDesc') },
  { icon: 'shield', title: t('home.features.balanceQuota'), desc: t('home.features.balanceQuotaDesc') },
] as const)

const providerItems = computed(() => [
  { name: t('home.providers.claude') },
  { name: 'GPT' },
  { name: t('home.providers.gemini') },
  { name: t('home.providers.antigravity') },
])

// Toggle theme
function toggleTheme() {
  appStore.toggleTheme()
}

onMounted(() => {
  // Check auth state
  authStore.checkAuth()

  // Ensure public settings are loaded (will use cache if already loaded from injected config)
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>
