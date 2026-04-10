<script setup lang="ts">
import { ref, computed, onMounted, h } from 'vue'
import {
  useVueTable, getCoreRowModel, FlexRender, type ColumnDef,
} from '@tanstack/vue-table'
import {
  Search, RefreshCw, MoreHorizontal, X, Tag,
  Server, Wifi, WifiOff, Clock, MapPin, Network,
  KeyRound, ChevronRight, ChevronLeft, Trash2, Pencil,
  Globe, ArrowUpRight, ArrowDownRight,
} from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem,
  DropdownMenuSeparator, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import AppAlertDialog from '@/components/AlertDialog.vue'
import { usePeerPageStore } from '@/stores/peerPage'

definePage({
  meta: { title: 'Node 管理', description: '管理网络中的所有节点。' },
})

const store = usePeerPageStore()
onMounted(() => store.actions.refresh())

// ── Types ─────────────────────────────────────────────────────────
type PeerRow = (typeof store.rows)[number]
type NodeStatus = 'online' | 'offline' | 'pending'

// ── Style maps ────────────────────────────────────────────────────
const statusDot: Record<string, string> = {
  online: 'bg-emerald-500', offline: 'bg-rose-500', pending: 'bg-amber-400',
}
const statusBadge: Record<string, string> = {
  online:  'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 ring-1 ring-emerald-500/20',
  offline: 'bg-rose-500/10 text-rose-600 dark:text-rose-400 ring-1 ring-rose-500/20',
  pending: 'bg-amber-400/10 text-amber-600 dark:text-amber-400 ring-1 ring-amber-400/20',
}
const statusLabel: Record<string, string> = {
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

// helper: labels map → array of "k=v"
function labelsToArray(labels: any): string[] {
  if (!labels) return []
  if (Array.isArray(labels)) return labels
  return Object.entries(labels).map(([k, v]) => `${k}=${v}`)
}

// ── Stats ─────────────────────────────────────────────────────────
const statusFilter = ref<NodeStatus | 'all'>('all')

const stats = computed(() => {
  const rows    = store.rows as any[]
  const online  = rows.filter(n => n.status === 'online').length
  const offline = rows.filter(n => n.status === 'offline').length
  const pending = rows.filter(n => n.status === 'pending').length
  const total   = store.total || rows.length
  const regions = new Set(rows.map(n => n.region).filter(Boolean)).size
  return {
    total, online, offline, pending, regions,
    onlineRate: total ? Math.round((online / total) * 100) : 0,
  }
})

// ── Sparklines (mock trend data) ──────────────────────────────────
function buildPath(data: number[], w: number, h: number, pad = 2) {
  const max = Math.max(...data)
  const min = Math.min(...data)
  const range = max - min || 1
  const xStep = (w - pad * 2) / (data.length - 1)
  const pts = data.map((v, i) => ({
    x: pad + i * xStep,
    y: h - pad - ((v - min) / range) * (h - pad * 2),
  }))
  return pts.map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' ')
}
const sparklines = {
  total:   [10, 12, 11, 14, 13, 15, 14, 16, 15, 17, 16, 18],
  online:  [8,  10,  9, 12, 11, 13, 12, 14, 13, 15, 14, 16],
  offline: [3,   2,  3,  2,  2,  1,  2,  1,  2,  1,  1,  2],
  pending: [2,   3,  2,  3,  3,  2,  2,  2,  2,  2,  2,  2],
}

// ── Search / filter (client-side over loaded page) ─────────────────
const searchValue = ref('')

const filtered = computed(() => {
  const q = searchValue.value.toLowerCase().trim()
  return store.rows.filter((n: any) => {
    const matchSearch = !q
      || n.name?.toLowerCase().includes(q)
      || n.appId?.toLowerCase().includes(q)
      || n.address?.toLowerCase().includes(q)
      || n.region?.toLowerCase().includes(q)
    const matchStatus = statusFilter.value === 'all' || n.status === statusFilter.value
    return matchSearch && matchStatus
  })
})

function setStatusFilter(val: typeof statusFilter.value) {
  statusFilter.value = val
  searchValue.value = ''
}

let searchTimer: ReturnType<typeof setTimeout>
function onSearchInput() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => { statusFilter.value = 'all' }, 400)
}

// ── Pagination (server-side) ───────────────────────────────────────
const PAGE_SIZE   = store.params.pageSize ?? 10
const currentPage = computed(() => store.params.page ?? 1)
const totalPages  = computed(() => Math.max(1, Math.ceil(store.total / PAGE_SIZE)))
const visiblePages = computed(() => {
  const cur   = currentPage.value
  const total = totalPages.value
  const start = Math.max(1, Math.min(cur - 1, total - 2))
  const end   = Math.min(total, start + 2)
  return Array.from({ length: end - start + 1 }, (_, i) => start + i)
})

function goToPage(p: number) {
  if (p < 1 || p > totalPages.value) return
  store.params.page = p
  store.actions.refresh()
}

// ── Delete confirm ─────────────────────────────────────────────────
const deleteTarget     = ref<PeerRow | null>(null)
const deleteDialogOpen = ref(false)

function promptDelete(node: PeerRow) {
  deleteTarget.value = node
  deleteDialogOpen.value = true
}
async function confirmDelete() {
  if (deleteTarget.value) {
    await store.actions.handleDelete(deleteTarget.value, () => Promise.resolve(true))
  }
  deleteTarget.value = null
}

// ── Column definitions ────────────────────────────────────────────
const columns: ColumnDef<PeerRow>[] = [
  {
    id: 'status',
    header: '状态',
    cell: ({ row }) => {
      const s: string = (row.original as any).status ?? 'pending'
      return h('div', { class: 'flex items-center gap-2' }, [
        h('span', { class: 'relative flex size-2 shrink-0' }, [
          s === 'online' && h('span', { class: `absolute inline-flex h-full w-full animate-ping rounded-full opacity-60 ${statusDot[s]}` }),
          h('span', { class: `relative inline-flex size-2 rounded-full ${statusDot[s] ?? 'bg-muted-foreground'}` }),
        ]),
        h('span', { class: `text-xs font-medium px-2 py-0.5 rounded-full ${statusBadge[s] ?? statusBadge.pending}` }, statusLabel[s] ?? s),
      ])
    },
  },
  {
    id: 'node',
    header: '节点',
    cell: ({ row }) => {
      const n = row.original as any
      return h('div', { class: 'flex flex-col gap-0.5' }, [
        h('span', { class: 'font-semibold text-sm leading-none' }, n.name ?? n.appId),
        h('span', { class: 'font-mono text-[11px] text-muted-foreground/60' }, n.appId),
      ])
    },
  },
  {
    id: 'region',
    header: '区域',
    cell: ({ row }) => {
      const region: string = (row.original as any).region ?? ''
      if (!region) return h('span', { class: 'text-[11px] text-muted-foreground/40' }, '—')
      return h('div', { class: 'flex items-center gap-1.5' }, [
        h('span', { class: 'text-base leading-none' }, regionFlag[region] ?? '🌐'),
        h('span', { class: 'text-xs text-muted-foreground' }, region),
      ])
    },
  },
  {
    id: 'network',
    header: '网络 / 地址',
    cell: ({ row }) => {
      const n = row.original as any
      const network = n.network ?? n.namespace ?? '—'
      const address = n.address ?? ''
      return h('div', { class: 'flex flex-col gap-0.5' }, [
        h('span', { class: 'text-xs text-muted-foreground' }, network),
        address && h('span', { class: 'font-mono text-[11px] text-muted-foreground/60' }, address),
      ])
    },
  },
  {
    id: 'labels',
    header: '标签',
    cell: ({ row }) => {
      const labels = labelsToArray((row.original as any).labels)
      if (!labels.length) return h('span', { class: 'text-[11px] text-muted-foreground/40' }, '—')
      return h('div', { class: 'flex flex-wrap gap-1' },
        labels.map(label =>
          h('span', { class: `text-[11px] font-medium px-2 py-0.5 rounded-full ${labelColor(label)}` }, label)
        )
      )
    },
  },
  {
    id: 'lastSeen',
    header: '最后在线',
    cell: ({ row }) => {
      const n = row.original as any
      if (!n.lastSeen) return h('span', { class: 'text-[11px] text-muted-foreground/40' }, '—')
      return h('span', {
        class: `text-xs ${n.status === 'offline' ? 'text-rose-500/70' : 'text-muted-foreground'}`,
      }, n.lastSeen)
    },
  },
  {
    id: 'actions',
    header: '',
    cell: ({ row }) => {
      const node = row.original
      return h(DropdownMenu, {}, {
        default: () => [
          h(DropdownMenuTrigger, { asChild: true }, () =>
            h(Button, { variant: 'ghost', size: 'sm', class: 'size-8 p-0' }, () =>
              h(MoreHorizontal, { class: 'size-4' })
            )
          ),
          h(DropdownMenuContent, { align: 'end', class: 'w-36' }, () => [
            h(DropdownMenuItem, { onClick: () => store.actions.openDrawer('view', node) }, () => [
              h(ChevronRight, { class: 'mr-2 size-3.5' }), '查看详情',
            ]),
            h(DropdownMenuItem, { onClick: () => store.actions.openDrawer('edit', node) }, () => [
              h(Pencil, { class: 'mr-2 size-3.5' }), '编辑标签',
            ]),
            h(DropdownMenuSeparator),
            h(DropdownMenuItem, {
              class: 'text-destructive focus:text-destructive',
              onClick: () => promptDelete(node),
            }, () => [h(Trash2, { class: 'mr-2 size-3.5' }), '删除节点']),
          ]),
        ],
      })
    },
  },
]

// ── TanStack Table ────────────────────────────────────────────────
const table = useVueTable({
  get data() { return filtered.value },
  columns,
  getCoreRowModel: getCoreRowModel(),
  manualPagination: true,
  manualFiltering: true,
})
</script>

<template>
  <div class="flex flex-col gap-5 p-6 animate-in fade-in duration-300">

    <!-- ── Stat cards ─────────────────────────────────────────────── -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">

      <!-- 全部节点 -->
      <button
        class="border-border bg-card text-card-foreground rounded-xl border p-5 shadow-sm text-left hover:shadow-md transition-shadow"
        :class="statusFilter === 'all' ? 'ring-2 ring-primary/20 border-primary/30' : ''"
        @click="setStatusFilter('all')"
      >
        <div class="flex items-start justify-between">
          <div class="flex flex-col gap-1">
            <span class="text-muted-foreground text-sm font-medium">全部节点</span>
            <span class="text-2xl font-bold tracking-tight">{{ stats.total }}</span>
          </div>
          <div class="bg-muted rounded-lg p-2">
            <Server class="text-muted-foreground size-4" />
          </div>
        </div>
        <div class="mt-3 flex items-center gap-1 text-sm">
          <Globe class="text-muted-foreground size-4 shrink-0" />
          <span class="text-muted-foreground">覆盖 <span class="font-semibold text-foreground">{{ stats.regions }}</span> 个地域</span>
        </div>
        <svg class="mt-3 w-full" viewBox="0 0 80 28" preserveAspectRatio="none" style="height:28px">
          <path :d="buildPath(sparklines.total, 80, 28)" fill="none" stroke="#10b981" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </button>

      <!-- 在线 -->
      <button
        class="border-border bg-card text-card-foreground rounded-xl border p-5 shadow-sm text-left hover:shadow-md transition-shadow"
        :class="statusFilter === 'online' ? 'ring-2 ring-emerald-500/20 border-emerald-500/30' : ''"
        @click="setStatusFilter('online')"
      >
        <div class="flex items-start justify-between">
          <div class="flex flex-col gap-1">
            <span class="text-muted-foreground text-sm font-medium">在线节点</span>
            <span class="text-2xl font-bold tracking-tight">{{ stats.online }}</span>
          </div>
          <div class="bg-muted rounded-lg p-2">
            <Wifi class="text-muted-foreground size-4" />
          </div>
        </div>
        <div class="mt-3 flex items-center gap-1 text-sm">
          <ArrowUpRight class="text-emerald-600 size-4 shrink-0" />
          <span class="text-emerald-600 font-semibold">{{ stats.onlineRate }}%</span>
          <span class="text-muted-foreground">在线率</span>
        </div>
        <svg class="mt-3 w-full" viewBox="0 0 80 28" preserveAspectRatio="none" style="height:28px">
          <path :d="buildPath(sparklines.online, 80, 28)" fill="none" stroke="#10b981" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </button>

      <!-- 离线 -->
      <button
        class="border-border bg-card text-card-foreground rounded-xl border p-5 shadow-sm text-left hover:shadow-md transition-shadow"
        :class="statusFilter === 'offline' ? 'ring-2 ring-red-500/20 border-red-500/30' : ''"
        @click="setStatusFilter('offline')"
      >
        <div class="flex items-start justify-between">
          <div class="flex flex-col gap-1">
            <span class="text-muted-foreground text-sm font-medium">离线节点</span>
            <span class="text-2xl font-bold tracking-tight">{{ stats.offline }}</span>
          </div>
          <div class="bg-muted rounded-lg p-2">
            <WifiOff class="text-muted-foreground size-4" />
          </div>
        </div>
        <div class="mt-3 flex items-center gap-1 text-sm">
          <component
            :is="stats.offline === 0 ? ArrowUpRight : ArrowDownRight"
            :class="stats.offline === 0 ? 'text-emerald-600' : 'text-red-500'"
            class="size-4 shrink-0"
          />
          <span :class="stats.offline === 0 ? 'text-emerald-600 font-semibold' : 'text-red-500 font-semibold'">
            {{ stats.offline === 0 ? '全部在线' : stats.offline + ' 台异常' }}
          </span>
          <span class="text-muted-foreground">{{ stats.offline === 0 ? '网络健康' : '需要检查' }}</span>
        </div>
        <svg class="mt-3 w-full" viewBox="0 0 80 28" preserveAspectRatio="none" style="height:28px">
          <path :d="buildPath(sparklines.offline, 80, 28)" fill="none" stroke="#ef4444" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </button>

      <!-- 待接入 -->
      <button
        class="border-border bg-card text-card-foreground rounded-xl border p-5 shadow-sm text-left hover:shadow-md transition-shadow"
        :class="statusFilter === 'pending' ? 'ring-2 ring-amber-400/20 border-amber-400/30' : ''"
        @click="setStatusFilter('pending')"
      >
        <div class="flex items-start justify-between">
          <div class="flex flex-col gap-1">
            <span class="text-muted-foreground text-sm font-medium">待接入</span>
            <span class="text-2xl font-bold tracking-tight">{{ stats.pending }}</span>
          </div>
          <div class="bg-muted rounded-lg p-2">
            <Clock class="text-muted-foreground size-4" />
          </div>
        </div>
        <div class="mt-3 flex items-center gap-1 text-sm">
          <component
            :is="stats.pending === 0 ? ArrowUpRight : ArrowDownRight"
            :class="stats.pending === 0 ? 'text-emerald-600' : 'text-amber-500'"
            class="size-4 shrink-0"
          />
          <span :class="stats.pending === 0 ? 'text-emerald-600 font-semibold' : 'text-amber-500 font-semibold'">
            {{ stats.pending === 0 ? '全部已接入' : stats.pending + ' 台待配置' }}
          </span>
          <span class="text-muted-foreground">{{ stats.pending === 0 ? '接入完成' : '等待配置' }}</span>
        </div>
        <svg class="mt-3 w-full" viewBox="0 0 80 28" preserveAspectRatio="none" style="height:28px">
          <path :d="buildPath(sparklines.pending, 80, 28)" fill="none" stroke="#f59e0b" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
      </button>

    </div>

    <!-- ── Toolbar ────────────────────────────────────────────────── -->
    <div class="flex items-center gap-2">
      <div class="relative w-72">
        <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
        <Input
          v-model="searchValue"
          placeholder="搜索名称、AppID、地址、区域..."
          class="pl-8 h-9"
          @input="onSearchInput"
        />
      </div>
      <div class="ml-auto flex items-center gap-2">
        <Button variant="outline" size="sm" class="gap-1.5"
          :disabled="store.loading" @click="store.actions.refresh()">
          <RefreshCw class="size-3.5" :class="store.loading ? 'animate-spin' : ''" />
          刷新
        </Button>
      </div>
    </div>

    <!-- ── Data Table ─────────────────────────────────────────────── -->
    <div class="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow v-for="hg in table.getHeaderGroups()" :key="hg.id">
            <TableHead v-for="header in hg.headers" :key="header.id">
              <FlexRender
                v-if="!header.isPlaceholder"
                :render="header.column.columnDef.header"
                :props="header.getContext()"
              />
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <template v-if="table.getRowModel().rows.length">
            <TableRow
              v-for="row in table.getRowModel().rows"
              :key="row.id"
              class="cursor-pointer"
              @click="store.actions.openDrawer('view', row.original)"
            >
              <TableCell
                v-for="cell in row.getVisibleCells()" :key="cell.id"
                @click.stop="cell.column.id === 'actions' ? undefined : store.actions.openDrawer('view', row.original)"
              >
                <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
              </TableCell>
            </TableRow>
          </template>
          <TableRow v-else>
            <TableCell :colspan="columns.length" class="h-32 text-center text-muted-foreground">
              {{ store.loading ? '加载中...' : '暂无节点数据' }}
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>

    <!-- ── Pagination ─────────────────────────────────────────────── -->
    <div class="flex items-center justify-between text-sm text-muted-foreground">
      <span>共 {{ store.total }} 条 · 第 {{ currentPage }} / {{ totalPages }} 页</span>
      <div class="flex items-center gap-1">
        <Button variant="outline" size="sm" class="size-8 p-0"
          :disabled="currentPage <= 1" @click="goToPage(currentPage - 1)">
          <ChevronLeft class="size-4" />
        </Button>
        <Button
          v-for="p in visiblePages" :key="p"
          variant="outline" size="sm" class="size-8 p-0 text-xs"
          :class="p === currentPage ? 'bg-primary text-primary-foreground border-primary hover:bg-primary/90 hover:text-primary-foreground' : ''"
          @click="goToPage(p)"
        >{{ p }}</Button>
        <Button variant="outline" size="sm" class="size-8 p-0"
          :disabled="currentPage >= totalPages" @click="goToPage(currentPage + 1)">
          <ChevronRight class="size-4" />
        </Button>
      </div>
    </div>

    <!-- ── Delete confirm ─────────────────────────────────────────── -->
    <AppAlertDialog
      v-model:open="deleteDialogOpen"
      title="删除节点"
      :description="`确认删除节点「${(deleteTarget as any)?.name ?? (deleteTarget as any)?.appId}」？该操作不可撤销。`"
      confirm-text="删除"
      variant="destructive"
      @confirm="confirmDelete"
      @cancel="deleteTarget = null"
    />

  </div>

  <!-- ── Node Detail / Edit Dialog ──────────────────────────────── -->
  <Dialog :open="store.isDrawerOpen" @update:open="v => { if (!v) store.isDrawerOpen = false }">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>
          <div class="flex items-center gap-3">
            <div
              class="size-9 rounded-xl flex items-center justify-center shrink-0"
              :class="store.selectedNode?.status === 'online' ? 'bg-emerald-500/10' : store.selectedNode?.status === 'offline' ? 'bg-rose-500/10' : 'bg-amber-400/10'"
            >
              <Server class="size-4"
                :class="store.selectedNode?.status === 'online' ? 'text-emerald-500' : store.selectedNode?.status === 'offline' ? 'text-rose-500' : 'text-amber-400'"
              />
            </div>
            <div class="flex flex-col gap-1">
              <span class="text-base font-bold leading-none">{{ store.selectedNode?.name ?? store.selectedNode?.appId }}</span>
              <span v-if="store.selectedNode?.status"
                class="text-xs font-medium px-2 py-0.5 rounded-full w-fit"
                :class="statusBadge[store.selectedNode.status] ?? statusBadge.pending">
                {{ statusLabel[store.selectedNode.status] ?? store.selectedNode.status }}
              </span>
            </div>
          </div>
        </DialogTitle>
      </DialogHeader>

      <div v-if="store.selectedNode" class="space-y-5 pt-1 max-h-[65vh] overflow-y-auto pr-1">
        <!-- Identity -->
        <div>
          <h3 class="text-[11px] font-bold uppercase tracking-widest text-muted-foreground/50 mb-2 flex items-center gap-1.5">
            <KeyRound class="size-3" /> 身份信息
          </h3>
          <div class="rounded-lg border border-border overflow-hidden">
            <div class="flex items-center justify-between px-4 py-2.5 border-b border-border/60">
              <span class="text-xs text-muted-foreground">App ID</span>
              <span class="font-mono text-xs">{{ store.selectedNode.appId }}</span>
            </div>
            <div class="flex items-start justify-between px-4 py-2.5 gap-4">
              <span class="text-xs text-muted-foreground shrink-0">公钥</span>
              <span class="font-mono text-xs text-right break-all opacity-70">{{ store.selectedNode.publicKey || '—' }}</span>
            </div>
          </div>
        </div>

        <!-- Network -->
        <div v-if="store.selectedNode.region || store.selectedNode.network || store.selectedNode.address || store.selectedNode.namespace">
          <h3 class="text-[11px] font-bold uppercase tracking-widest text-muted-foreground/50 mb-2 flex items-center gap-1.5">
            <Network class="size-3" /> 网络信息
          </h3>
          <div class="rounded-lg border border-border overflow-hidden divide-y divide-border/60">
            <div v-if="store.selectedNode.region" class="flex items-center justify-between px-4 py-2.5">
              <span class="text-xs text-muted-foreground flex items-center gap-1"><MapPin class="size-3" /> 区域</span>
              <span class="text-xs">{{ regionFlag[store.selectedNode.region] ?? '🌐' }} {{ store.selectedNode.region }}</span>
            </div>
            <div v-if="store.selectedNode.network || store.selectedNode.namespace" class="flex items-center justify-between px-4 py-2.5">
              <span class="text-xs text-muted-foreground">命名空间</span>
              <span class="text-xs font-mono">{{ store.selectedNode.network ?? store.selectedNode.namespace }}</span>
            </div>
            <div v-if="store.selectedNode.address" class="flex items-center justify-between px-4 py-2.5">
              <span class="text-xs text-muted-foreground">IP 地址</span>
              <span class="font-mono text-xs">{{ store.selectedNode.address }}</span>
            </div>
            <div v-if="store.selectedNode.lastSeen" class="flex items-center justify-between px-4 py-2.5">
              <span class="text-xs text-muted-foreground flex items-center gap-1"><Clock class="size-3" /> 最后在线</span>
              <span class="text-xs" :class="store.selectedNode.status === 'offline' ? 'text-rose-500' : 'text-muted-foreground'">
                {{ store.selectedNode.lastSeen }}
              </span>
            </div>
          </div>
        </div>

        <!-- Labels -->
        <div>
          <h3 class="text-[11px] font-bold uppercase tracking-widest text-muted-foreground/50 mb-2 flex items-center gap-1.5">
            <Tag class="size-3" /> 标签
          </h3>
          <div class="flex flex-wrap gap-1.5 mb-2 min-h-6">
            <span
              v-for="(label, i) in store.selectedNode.labels" :key="i"
              class="flex items-center gap-1 text-[11px] font-medium px-2 py-0.5 rounded-full"
              :class="labelColor(label)"
            >
              {{ label }}
              <button
                v-if="store.drawerType === 'edit'"
                class="opacity-60 hover:opacity-100 hover:text-destructive transition-colors"
                type="button"
                @click="store.actions.removeLabel(i)"
              >
                <X class="size-3" />
              </button>
            </span>
            <span v-if="!store.selectedNode.labels.length" class="text-xs text-muted-foreground/40">暂无标签</span>
          </div>
          <div v-if="store.drawerType === 'edit'" class="flex gap-2">
            <Input
              v-model="store.newLabelInput"
              placeholder="key=value 后按 Enter..."
              class="h-8 text-xs"
              @keydown.enter="store.actions.addLabel()"
            />
            <Button size="sm" variant="outline" class="shrink-0" @click="store.actions.addLabel()">添加</Button>
          </div>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="store.isDrawerOpen = false">
          {{ store.drawerType === 'view' ? '关闭' : '取消' }}
        </Button>
        <Button
          v-if="store.drawerType === 'edit'"
          :disabled="store.isUpdating"
          @click="store.actions.handleSave()"
        >
          <RefreshCw v-if="store.isUpdating" class="size-3.5 animate-spin mr-2" />
          保存更改
        </Button>
        <Button
          v-else
          variant="secondary"
          @click="store.actions.openDrawer('edit', store.selectedNode)"
        >
          <Pencil class="size-3.5 mr-1.5" /> 编辑标签
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
