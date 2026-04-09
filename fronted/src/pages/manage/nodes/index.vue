<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  Search, Plus, RefreshCw, MoreHorizontal, X, Tag,
  Server, Wifi, WifiOff, Clock, MapPin, Network,
  KeyRound, Cpu, Activity, ChevronRight,
} from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Sheet, SheetContent } from '@/components/ui/sheet'
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem,
  DropdownMenuSeparator, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

definePage({
  meta: { title: 'Node 管理', description: '管理网络中的所有节点。' },
})

type NodeStatus = 'online' | 'offline' | 'pending'

interface NodeItem {
  id: string
  name: string
  appId: string
  publicKey: string
  region: string
  address: string
  network: string
  status: NodeStatus
  lastSeen: string
  labels: string[]
}

const nodes = ref<NodeItem[]>([
  { id: '1', name: 'node-alpha', appId: 'app-001', publicKey: 'wg1abc...xyz', region: 'us-west-2', address: '10.0.0.1', network: 'main-net', status: 'online', lastSeen: '2 分钟前', labels: ['prod', 'gateway'] },
  { id: '2', name: 'node-beta', appId: 'app-002', publicKey: 'wg2def...uvw', region: 'eu-central-1', address: '10.0.0.2', network: 'main-net', status: 'online', lastSeen: '5 分钟前', labels: ['prod'] },
  { id: '3', name: 'node-gamma', appId: 'app-003', publicKey: 'wg3ghi...rst', region: 'ap-southeast-1', address: '10.0.0.3', network: 'dev-net', status: 'offline', lastSeen: '2 小时前', labels: ['dev'] },
  { id: '4', name: 'node-delta', appId: 'app-004', publicKey: 'wg4jkl...opq', region: 'us-east-1', address: '10.0.0.4', network: 'main-net', status: 'pending', lastSeen: '从未', labels: [] },
  { id: '5', name: 'node-epsilon', appId: 'app-005', publicKey: 'wg5mno...lmn', region: 'us-west-2', address: '10.0.0.5', network: 'test-net', status: 'online', lastSeen: '1 分钟前', labels: ['test', 'relay'] },
  { id: '6', name: 'node-zeta', appId: 'app-006', publicKey: 'wg6pqr...ijk', region: 'eu-west-1', address: '10.0.0.6', network: 'main-net', status: 'offline', lastSeen: '1 天前', labels: ['prod'] },
])

const search = ref('')
const statusFilter = ref<NodeStatus | 'all'>('all')
const drawerOpen = ref(false)
const selectedNode = ref<NodeItem | null>(null)
const newLabel = ref('')
const isRefreshing = ref(false)

const stats = computed(() => ({
  total: nodes.value.length,
  online: nodes.value.filter(n => n.status === 'online').length,
  offline: nodes.value.filter(n => n.status === 'offline').length,
  pending: nodes.value.filter(n => n.status === 'pending').length,
}))

const filtered = computed(() => nodes.value.filter(n => {
  const q = search.value.toLowerCase()
  const matchSearch = !q || n.name.includes(q) || n.appId.includes(q) || n.address.includes(q) || n.region.includes(q)
  const matchStatus = statusFilter.value === 'all' || n.status === statusFilter.value
  return matchSearch && matchStatus
}))

function openDrawer(node: NodeItem) {
  selectedNode.value = { ...node, labels: [...node.labels] }
  drawerOpen.value = true
}

function addLabel() {
  const val = newLabel.value.trim()
  if (val && selectedNode.value && !selectedNode.value.labels.includes(val)) {
    selectedNode.value.labels.push(val)
    newLabel.value = ''
  }
}

function removeLabel(i: number) {
  selectedNode.value?.labels.splice(i, 1)
}

function handleRefresh() {
  isRefreshing.value = true
  setTimeout(() => (isRefreshing.value = false), 1200)
}

// ── Style maps ────────────────────────────────────────────────────
const statusDot: Record<NodeStatus, string> = {
  online: 'bg-emerald-500',
  offline: 'bg-rose-500',
  pending: 'bg-amber-400',
}
const statusBadge: Record<NodeStatus, string> = {
  online: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 ring-1 ring-emerald-500/20',
  offline: 'bg-rose-500/10 text-rose-600 dark:text-rose-400 ring-1 ring-rose-500/20',
  pending: 'bg-amber-400/10 text-amber-600 dark:text-amber-400 ring-1 ring-amber-400/20',
}
const statusLabel: Record<NodeStatus, string> = {
  online: '在线', offline: '离线', pending: '待接入',
}
const labelColors = [
  'bg-blue-500/10 text-blue-600 dark:text-blue-400 ring-1 ring-blue-500/20',
  'bg-violet-500/10 text-violet-600 dark:text-violet-400 ring-1 ring-violet-500/20',
  'bg-cyan-500/10 text-cyan-600 dark:text-cyan-400 ring-1 ring-cyan-500/20',
  'bg-orange-500/10 text-orange-600 dark:text-orange-400 ring-1 ring-orange-500/20',
]
function labelColor(label: string) {
  let h = 0
  for (const c of label) h = (h * 31 + c.charCodeAt(0)) & 0xff
  return labelColors[h % labelColors.length]
}

const regionFlag: Record<string, string> = {
  'us-west-2': '🇺🇸', 'us-east-1': '🇺🇸',
  'eu-central-1': '🇩🇪', 'eu-west-1': '🇬🇧',
  'ap-southeast-1': '🇸🇬',
}
</script>

<template>
  <div class="flex flex-col gap-5 p-6 animate-in fade-in duration-300">

    <!-- ── Summary stat cards ─────────────────────────────────────── -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
      <button
        class="bg-card border border-border rounded-xl p-4 text-left hover:border-primary/30 transition-colors group"
        :class="statusFilter === 'all' ? 'border-primary/40 ring-1 ring-primary/10' : ''"
        @click="statusFilter = 'all'"
      >
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs font-medium text-muted-foreground">全部节点</span>
          <Server class="size-4 text-muted-foreground/50 group-hover:text-muted-foreground transition-colors" />
        </div>
        <p class="text-2xl font-black tracking-tighter">{{ stats.total }}</p>
      </button>

      <button
        class="bg-card border border-border rounded-xl p-4 text-left hover:border-emerald-500/30 transition-colors group"
        :class="statusFilter === 'online' ? 'border-emerald-500/40 ring-1 ring-emerald-500/10' : ''"
        @click="statusFilter = 'online'"
      >
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs font-medium text-muted-foreground">在线</span>
          <Wifi class="size-4 text-emerald-500/60 group-hover:text-emerald-500 transition-colors" />
        </div>
        <p class="text-2xl font-black tracking-tighter text-emerald-500">{{ stats.online }}</p>
      </button>

      <button
        class="bg-card border border-border rounded-xl p-4 text-left hover:border-rose-500/30 transition-colors group"
        :class="statusFilter === 'offline' ? 'border-rose-500/40 ring-1 ring-rose-500/10' : ''"
        @click="statusFilter = 'offline'"
      >
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs font-medium text-muted-foreground">离线</span>
          <WifiOff class="size-4 text-rose-500/60 group-hover:text-rose-500 transition-colors" />
        </div>
        <p class="text-2xl font-black tracking-tighter text-rose-500">{{ stats.offline }}</p>
      </button>

      <button
        class="bg-card border border-border rounded-xl p-4 text-left hover:border-amber-400/30 transition-colors group"
        :class="statusFilter === 'pending' ? 'border-amber-400/40 ring-1 ring-amber-400/10' : ''"
        @click="statusFilter = 'pending'"
      >
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs font-medium text-muted-foreground">待接入</span>
          <Clock class="size-4 text-amber-400/60 group-hover:text-amber-400 transition-colors" />
        </div>
        <p class="text-2xl font-black tracking-tighter text-amber-400">{{ stats.pending }}</p>
      </button>
    </div>

    <!-- ── Toolbar ────────────────────────────────────────────────── -->
    <div class="flex flex-wrap items-center gap-2">
      <div class="relative flex-1 min-w-52">
        <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
        <Input v-model="search" placeholder="搜索名称、AppID、地址、区域..." class="pl-8 h-9" />
      </div>
      <div class="ml-auto flex items-center gap-2">
        <Button
          variant="outline" size="sm"
          :class="isRefreshing ? 'opacity-60 pointer-events-none' : ''"
          class="gap-1.5"
          @click="handleRefresh"
        >
          <RefreshCw class="size-3.5" :class="isRefreshing ? 'animate-spin' : ''" />
          刷新
        </Button>
        <Button size="sm" class="gap-1.5">
          <Plus class="size-3.5" /> 添加节点
        </Button>
      </div>
    </div>

    <!-- ── Node table ─────────────────────────────────────────────── -->
    <div class="bg-card border border-border rounded-xl overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-border">
            <th class="text-left px-5 py-3 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70 w-32">状态</th>
            <th class="text-left px-5 py-3 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">节点</th>
            <th class="text-left px-5 py-3 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70 hidden md:table-cell">区域</th>
            <th class="text-left px-5 py-3 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70 hidden lg:table-cell">网络 / 地址</th>
            <th class="text-left px-5 py-3 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70 hidden xl:table-cell">标签</th>
            <th class="text-left px-5 py-3 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70 hidden sm:table-cell">最后在线</th>
            <th class="px-5 py-3 w-10" />
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="node in filtered"
            :key="node.id"
            class="border-b border-border last:border-0 transition-colors cursor-pointer group"
            :class="node.status === 'offline' ? 'opacity-60 hover:opacity-100' : 'hover:bg-muted/30'"
            @click="openDrawer(node)"
          >
            <!-- Status -->
            <td class="px-5 py-3.5">
              <div class="flex items-center gap-2">
                <span class="relative flex size-2">
                  <span
                    v-if="node.status === 'online'"
                    class="absolute inline-flex h-full w-full animate-ping rounded-full opacity-60"
                    :class="statusDot[node.status]"
                  />
                  <span class="relative inline-flex size-2 rounded-full" :class="statusDot[node.status]" />
                </span>
                <span class="text-xs font-medium px-2 py-0.5 rounded-full" :class="statusBadge[node.status]">
                  {{ statusLabel[node.status] }}
                </span>
              </div>
            </td>

            <!-- Node identity -->
            <td class="px-5 py-3.5">
              <div class="flex flex-col gap-0.5">
                <span class="font-semibold text-sm leading-none">{{ node.name }}</span>
                <span class="font-mono text-[11px] text-muted-foreground/60">{{ node.appId }}</span>
              </div>
            </td>

            <!-- Region -->
            <td class="px-5 py-3.5 hidden md:table-cell">
              <div class="flex items-center gap-1.5 text-sm text-muted-foreground">
                <span class="text-base leading-none">{{ regionFlag[node.region] ?? '🌐' }}</span>
                <span class="text-xs">{{ node.region }}</span>
              </div>
            </td>

            <!-- Network / Address -->
            <td class="px-5 py-3.5 hidden lg:table-cell">
              <div class="flex flex-col gap-0.5">
                <span class="text-xs text-muted-foreground">{{ node.network }}</span>
                <span class="font-mono text-[11px] text-muted-foreground/60">{{ node.address }}</span>
              </div>
            </td>

            <!-- Labels -->
            <td class="px-5 py-3.5 hidden xl:table-cell">
              <div class="flex flex-wrap gap-1">
                <span
                  v-for="label in node.labels"
                  :key="label"
                  class="text-[11px] font-medium px-2 py-0.5 rounded-full"
                  :class="labelColor(label)"
                >{{ label }}</span>
                <span v-if="node.labels.length === 0" class="text-[11px] text-muted-foreground/40">—</span>
              </div>
            </td>

            <!-- Last seen -->
            <td class="px-5 py-3.5 hidden sm:table-cell">
              <span
                class="text-xs"
                :class="node.status === 'offline' ? 'text-rose-500/70' : 'text-muted-foreground'"
              >{{ node.lastSeen }}</span>
            </td>

            <!-- Actions -->
            <td class="px-5 py-3.5" @click.stop>
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button
                    variant="ghost" size="icon-sm"
                    class="opacity-0 group-hover:opacity-100 transition-opacity"
                  >
                    <MoreHorizontal class="size-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" class="w-40">
                  <DropdownMenuItem @click="openDrawer(node)">
                    <ChevronRight class="size-3.5 mr-2" /> 查看详情
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem class="text-destructive focus:text-destructive">
                    删除节点
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </td>
          </tr>

          <!-- Empty state -->
          <tr v-if="filtered.length === 0">
            <td colspan="7" class="py-16 text-center">
              <div class="flex flex-col items-center gap-2">
                <Server class="size-8 text-muted-foreground/20" />
                <p class="text-sm text-muted-foreground">未找到匹配节点</p>
                <p class="text-xs text-muted-foreground/60">尝试调整搜索或筛选条件</p>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- Table footer -->
      <div v-if="filtered.length > 0" class="border-t border-border px-5 py-2.5 flex items-center justify-between">
        <span class="text-xs text-muted-foreground/60">
          显示 {{ filtered.length }} / {{ nodes.length }} 个节点
        </span>
      </div>
    </div>

  </div>

  <!-- ── Node Detail Drawer ──────────────────────────────────────── -->
  <Sheet v-model:open="drawerOpen">
    <SheetContent class="w-[440px] sm:w-[500px] p-0 overflow-y-auto flex flex-col gap-0">
      <div v-if="selectedNode">

        <!-- Drawer header -->
        <div class="px-6 pt-7 pb-5 border-b border-border">
          <div class="flex items-start justify-between gap-4">
            <div class="flex items-center gap-3">
              <div
                class="size-10 rounded-xl flex items-center justify-center"
                :class="selectedNode.status === 'online' ? 'bg-emerald-500/10' : selectedNode.status === 'offline' ? 'bg-rose-500/10' : 'bg-amber-400/10'"
              >
                <Server
                  class="size-5"
                  :class="selectedNode.status === 'online' ? 'text-emerald-500' : selectedNode.status === 'offline' ? 'text-rose-500' : 'text-amber-400'"
                />
              </div>
              <div>
                <h2 class="text-base font-bold leading-tight">{{ selectedNode.name }}</h2>
                <span class="text-xs font-medium px-2 py-0.5 rounded-full mt-1 inline-block" :class="statusBadge[selectedNode.status]">
                  {{ statusLabel[selectedNode.status] }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- Drawer body -->
        <div class="px-6 py-5 space-y-6">

          <!-- Identity section -->
          <div>
            <h3 class="text-[11px] font-bold uppercase tracking-widest text-muted-foreground/50 mb-3 flex items-center gap-1.5">
              <KeyRound class="size-3" /> 身份信息
            </h3>
            <div class="space-y-2">
              <div class="flex items-center justify-between py-2 border-b border-border/40">
                <span class="text-xs text-muted-foreground">App ID</span>
                <span class="font-mono text-xs">{{ selectedNode.appId }}</span>
              </div>
              <div class="flex items-start justify-between py-2 border-b border-border/40 gap-4">
                <span class="text-xs text-muted-foreground shrink-0">公钥</span>
                <span class="font-mono text-xs text-right break-all opacity-70">{{ selectedNode.publicKey }}</span>
              </div>
            </div>
          </div>

          <!-- Network section -->
          <div>
            <h3 class="text-[11px] font-bold uppercase tracking-widest text-muted-foreground/50 mb-3 flex items-center gap-1.5">
              <Network class="size-3" /> 网络信息
            </h3>
            <div class="space-y-2">
              <div class="flex items-center justify-between py-2 border-b border-border/40">
                <span class="text-xs text-muted-foreground flex items-center gap-1"><MapPin class="size-3" /> 区域</span>
                <span class="text-xs flex items-center gap-1">
                  {{ regionFlag[selectedNode.region] ?? '🌐' }} {{ selectedNode.region }}
                </span>
              </div>
              <div class="flex items-center justify-between py-2 border-b border-border/40">
                <span class="text-xs text-muted-foreground">网络</span>
                <span class="text-xs">{{ selectedNode.network }}</span>
              </div>
              <div class="flex items-center justify-between py-2 border-b border-border/40">
                <span class="text-xs text-muted-foreground">IP 地址</span>
                <span class="font-mono text-xs">{{ selectedNode.address }}</span>
              </div>
              <div class="flex items-center justify-between py-2 border-b border-border/40">
                <span class="text-xs text-muted-foreground flex items-center gap-1"><Clock class="size-3" /> 最后在线</span>
                <span
                  class="text-xs"
                  :class="selectedNode.status === 'offline' ? 'text-rose-500' : 'text-muted-foreground'"
                >{{ selectedNode.lastSeen }}</span>
              </div>
            </div>
          </div>

          <!-- Labels section -->
          <div>
            <h3 class="text-[11px] font-bold uppercase tracking-widest text-muted-foreground/50 mb-3 flex items-center gap-1.5">
              <Tag class="size-3" /> 标签
            </h3>
            <div class="flex flex-wrap gap-1.5 min-h-7 mb-3">
              <span
                v-for="(label, i) in selectedNode.labels"
                :key="i"
                class="flex items-center gap-1 text-[11px] font-medium px-2 py-0.5 rounded-full"
                :class="labelColor(label)"
              >
                {{ label }}
                <button class="opacity-60 hover:opacity-100 hover:text-destructive transition-colors" @click="removeLabel(i)">
                  <X class="size-3" />
                </button>
              </span>
              <span v-if="selectedNode.labels.length === 0" class="text-xs text-muted-foreground/40">暂无标签</span>
            </div>
            <div class="flex gap-2">
              <Input
                v-model="newLabel"
                placeholder="输入标签后按 Enter..."
                class="h-8 text-xs"
                @keydown.enter="addLabel"
              />
              <Button size="sm" variant="outline" class="shrink-0" @click="addLabel">添加</Button>
            </div>
          </div>

        </div>

        <!-- Drawer footer -->
        <div class="mt-auto border-t border-border px-6 py-4 flex justify-end gap-2">
          <Button variant="outline" @click="drawerOpen = false">取消</Button>
          <Button>保存更改</Button>
        </div>

      </div>
    </SheetContent>
  </Sheet>
</template>
