<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="space-y-6" data-test="user-dashboard-skeleton">
        <div v-for="row in 2" :key="`stats-row-${row}`" class="grid grid-cols-2 gap-4 lg:grid-cols-3">
          <div v-for="card in 3" :key="`stat-${row}-${card}`" class="card hud-corners p-4">
            <div class="flex items-center gap-3">
              <Skeleton variant="rect" width="40px" height="40px" />
              <div class="min-w-0 flex-1 space-y-2">
                <Skeleton variant="text" width="55%" />
                <Skeleton variant="text" width="35%" />
                <Skeleton variant="text" width="45%" />
              </div>
            </div>
          </div>
        </div>

        <div class="card hud-corners p-6">
          <Skeleton variant="text" width="35%" />
          <Skeleton variant="rect" height="260px" class="mt-4" />
        </div>

        <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div class="card hud-corners p-4 lg:col-span-2">
            <Skeleton variant="text" width="30%" />
            <div class="mt-4 space-y-3">
              <div v-for="row in 5" :key="`usage-${row}`" class="flex items-center gap-3">
                <Skeleton variant="rect" width="32px" height="32px" />
                <Skeleton variant="text" width="55%" />
                <Skeleton variant="text" width="20%" />
              </div>
            </div>
          </div>
          <div class="card hud-corners p-4">
            <Skeleton variant="text" width="45%" />
            <div class="mt-4 space-y-3">
              <Skeleton variant="rect" height="44px" />
              <Skeleton variant="rect" height="44px" />
            </div>
          </div>
        </div>
      </div>
      <template v-else-if="stats">
        <UserDashboardStats :stats="stats" :is-simple="authStore.isSimpleMode" :platform-quotas="platformQuotas" />
        <UserDashboardCharts v-model:startDate="startDate" v-model:endDate="endDate" v-model:granularity="granularity" :loading="loadingCharts" :trend="trendData" :models="modelStats" @dateRangeChange="loadCharts" @granularityChange="loadCharts" @refresh="refreshAll" />
        <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div class="lg:col-span-2"><UserDashboardRecentUsage :data="recentUsage" :loading="loadingUsage" /></div>
          <div class="lg:col-span-1"><UserDashboardQuickActions /></div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'; import { useAuthStore } from '@/stores/auth'; import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import AppLayout from '@/components/layout/AppLayout.vue'; import Skeleton from '@/components/common/Skeleton.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'; import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardRecentUsage from '@/components/user/dashboard/UserDashboardRecentUsage.vue'; import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import type { UsageLog, TrendDataPoint, ModelStat, PlatformQuotaItem } from '@/types'
import { getMyPlatformQuotas } from '@/api/user'
import { formatDateLocalInput } from '@/utils/format'

const authStore = useAuthStore()
const stats = ref<UserStatsType | null>(null); const loading = ref(false); const loadingUsage = ref(false); const loadingCharts = ref(false)
const trendData = ref<TrendDataPoint[]>([]); const modelStats = ref<ModelStat[]>([]); const recentUsage = ref<UsageLog[]>([])
const platformQuotas = ref<PlatformQuotaItem[] | null>(null)

const startDate = ref(formatDateLocalInput(new Date(Date.now() - 6 * 86400000))); const endDate = ref(formatDateLocalInput(new Date())); const granularity = ref('day')

const loadStats = async () => { loading.value = true; try { await authStore.refreshUser(); stats.value = await usageAPI.getDashboardStats() } catch (error) { console.error('Failed to load dashboard stats:', error) } finally { loading.value = false } }
const loadCharts = async () => { loadingCharts.value = true; try { const res = await Promise.all([usageAPI.getDashboardTrend({ start_date: startDate.value, end_date: endDate.value, granularity: granularity.value as any }), usageAPI.getDashboardModels({ start_date: startDate.value, end_date: endDate.value })]); trendData.value = res[0].trend || []; modelStats.value = res[1].models || [] } catch (error) { console.error('Failed to load charts:', error) } finally { loadingCharts.value = false } }
const loadRecent = async () => { loadingUsage.value = true; try { const res = await usageAPI.getByDateRange(startDate.value, endDate.value); recentUsage.value = res.items.slice(0, 5) } catch (error) { console.error('Failed to load recent usage:', error) } finally { loadingUsage.value = false } }
const loadPlatformQuotas = async () => { try { const data = await getMyPlatformQuotas(); platformQuotas.value = data.platform_quotas ?? [] } catch (error) { console.warn('Failed to load platform quotas:', error); platformQuotas.value = [] } }
const refreshAll = () => { loadStats(); loadCharts(); loadRecent(); loadPlatformQuotas() }

onMounted(() => { refreshAll() })
</script>
