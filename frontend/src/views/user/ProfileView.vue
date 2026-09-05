<template>
  <AppLayout>
    <div data-testid="profile-shell" class="mx-auto max-w-[950px] space-y-6">
      <div class="sub3-glass-panel p-1">
        <ProfileInfoCard :user="user" :show-bindings="false" />
      </div>
      <div class="sub3-glass-panel p-6">
        <div class="mb-5">
          <p class="sub3-eyebrow">operator security</p>
          <h2 class="mt-3 text-xl font-semibold">{{ securityTitle }}</h2>
          <p class="mt-1 text-sm text-token-muted">{{ securityDescription }}</p>
        </div>
        <ProfilePasswordForm />
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ProfileInfoCard from '@/components/user/profile/ProfileInfoCard.vue'
import ProfilePasswordForm from '@/components/user/profile/ProfilePasswordForm.vue'
import { useAuthStore } from '@/stores/auth'

const { locale } = useI18n()
const authStore = useAuthStore()
const user = computed(() => authStore.user)
const securityTitle = computed(() => locale.value.startsWith('zh') ? '\u7ad9\u957f\u5bc6\u7801' : 'Operator password')
const securityDescription = computed(() => locale.value.startsWith('zh') ? '\u4e2a\u4eba\u7248\u53ea\u4fdd\u7559\u552f\u4e00\u7ad9\u957f\u8d26\u53f7\u7684\u57fa\u672c\u8d44\u6599\u4e0e\u5bc6\u7801\u7ba1\u7406\u3002' : 'The personal edition keeps only the operator profile and password controls.')

onMounted(() => {
  void authStore.refreshUser()
})
</script>
