<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  ArrowLeftRight, Plus, RefreshCw, MoreHorizontal, Trash2,
  CheckCircle2, XCircle, Clock, AlertTriangle, Info,
  ChevronLeft, ChevronRight, Search, Zap,
  Activity, Network,
} from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle,
  DialogDescription, DialogFooter,
} from '@/components/ui/dialog'
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem,
  DropdownMenuSeparator, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import AppAlertDialog from '@/components/AlertDialog.vue'
import { listPeerings, createPeering, deletePeering } from '@/api/user'
import { useWorkspaceStore } from '@/stores/workspace'
import { toast } from 'vue-sonner'

definePage({
  meta: { title: '对等连接', description: '管理跨空间网段互通的对等连接通道。' },
})

// ── Types ─────────────────────────────────────────────────────────
type PeeringStatus = 'active' | 'pending' | 'failed'
type PeeringMode   = 'gateway' | 'mesh'

interface WorkspaceEndpoint {
  name:      string
  namespace: string
  cidr:      string
  nodeCount: number
}

interface PeeringConnection {
  name:        string
  local:       WorkspaceEndpoint
  remote:      WorkspaceEndpoint
  status:      PeeringStatus
  peeringMode: PeeringMode
  createdAt:   string
}

// ── Data ──────────────────────────────────────────────────────────
const workspaceStore = useWorkspaceStore()
const connections    = ref<PeeringConnection[]>([])
const loading        = ref(false)

async function fetchList() {
  loading.value = true
  try {
    const res = await listPeerings()
    connections.value = (res.data?.data ?? res.data ?? []) as PeeringConnection[]
  } catch (e: any) {
    toast('加载失败', { description: e?.message })
  } finally {
    loading.value = false
  }
}

onMounted(fetchList)

// ── Style maps ────────────────────────────────────────────────────
const statusConfig: Record<PeeringStatus, { label: string; badge: string; icon: any; dot: string }> = {
  active:  { label: '已连接',   badge: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 ring-1 ring-emerald-500/20', icon: CheckCircle2, dot: 'bg-emerald-500' },
  pending: { label: '建立中',   badge: 'bg-amber-400/10 text-amber-600 dark:text-amber-400 ring-1 ring-amber-400/20',         icon: Clock,        dot: 'bg-amber-400' },
  failed:  { label: '连接失败', badge: 'bg-rose-500/10 text-rose-600 dark:text-rose-400 ring-1 ring-rose-500/20',             icon: XCircle,      dot: 'bg-rose-500' },
}

const modeConfig: Record<PeeringMode, { label: string; badge: string; tip: string }> = {
  gateway: { label: 'Gateway', badge: 'bg-blue-500/10 text-blue-600 dark:text-blue-400 ring-1 ring-blue-500/20',     tip: '通过网关节点中转，推荐方式' },
  mesh:    { label: 'Mesh',    badge: 'bg-violet-500/10 text-violet-600 dark:text-violet-400 ring-1 ring-violet-500/20', tip: '全互联，适合小规模部署' },
}

// ── Stats ─────────────────────────────────────────────────────────
type StatusFilter = PeeringStatus | 'all'
const statusFilter = ref<StatusFilter>('all')
const searchValue  = ref('')

const stats = computed(() => ({
  total:   connections.value.length,
  active:  connections.value.filter(c => c.status === 'active').length,
  pending: connections.value.filter(c => c.status === 'pending').length,
  failed:  connections.value.filter(c => c.status === 'failed').length,
}))

// ── Filter / pagination ───────────────────────────────────────────
const filtered = computed(() => {
  const q = searchValue.value.toLowerCase().trim()
  return connections.value.filter(c => {
    const matchSearch = !q
      || c.name.toLowerCase().includes(q)
      || c.local.name.toLowerCase().includes(q)
      || c.remote.name.toLowerCase().includes(q)
      || c.local.namespace.toLowerCase().includes(q)
      || c.remote.namespace.toLowerCase().includes(q)
    const matchStatus = statusFilter.value === 'all' || c.status === statusFilter.value
    return matchSearch && matchStatus
  })
})

const PAGE_SIZE    = 10
const currentPage  = ref(1)
const totalPages   = computed(() => Math.max(1, Math.ceil(filtered.value.length / PAGE_SIZE)))
const visiblePages = computed(() => {
  const cur   = currentPage.value
  const total = totalPages.value
  const start = Math.max(1, Math.min(cur - 1, total - 2))
  const end   = Math.min(total, start + 2)
  return Array.from({ length: end - start + 1 }, (_, i) => start + i)
})
const paginated = computed(() => {
  const start = (currentPage.value - 1) * PAGE_SIZE
  return filtered.value.slice(start, start + PAGE_SIZE)
})

function setFilter(val: StatusFilter) {
  statusFilter.value = val
  searchValue.value  = ''
  currentPage.value  = 1
}

// ── Detail dialog ─────────────────────────────────────────────────
const detailOpen = ref(false)
const selected   = ref<PeeringConnection | null>(null)

function openDetail(conn: PeeringConnection) {
  selected.value  = conn
  detailOpen.value = true
}

// ── Create dialog ─────────────────────────────────────────────────
const createOpen    = ref(false)
const createLoading = ref(false)
const createForm    = ref({
  name:        '',
  namespaceB:  '',
  networkB:    '',
  peeringMode: 'gateway' as PeeringMode,
})

function openCreate() {
  createForm.value = { name: '', namespaceB: '', networkB: '', peeringMode: 'gateway' }
  createOpen.value = true
}

async function handleCreate() {
  if (!createForm.value.namespaceB.trim()) {
    toast('请填写对端命名空间')
    return
  }
  createLoading.value = true
  try {
    await createPeering({
      name:        createForm.value.name.trim() || undefined,
      namespaceB:  createForm.value.namespaceB.trim(),
      networkB:    createForm.value.networkB.trim() || undefined,
      peeringMode: createForm.value.peeringMode,
    })
    toast('连接已发起，等待 Controller 建立通道')
    createOpen.value = false
    fetchList()
  } catch (e: any) {
    toast('创建失败', { description: e?.message })
  } finally {
    createLoading.value = false
  }
}

// ── Delete ────────────────────────────────────────────────────────
const deleteTarget     = ref<PeeringConnection | null>(null)
const deleteDialogOpen = ref(false)

function promptDelete(conn: PeeringConnection) {
  deleteTarget.value     = conn
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  try {
    await deletePeering(deleteTarget.value.name)
    toast('对等连接已删除')
    fetchList()
  } catch (e: any) {
    toast('删除失败', { description: e?.message })
  } finally {
    deleteTarget.value = null
  }
}
</script>

<template>
  <div class="flex flex-col gap-5 p-6 animate-in fade-in duration-300">

    <!-- ── Stat cards ─────────────────────────────────────────────── -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
      <button
        class="bg-card border border-border rounded-xl p-4 text-left hover:border-primary/30 transition-colors group"
        :class="statusFilter === 'all' ? 'border-primary/40 ring-1 ring-primary/10' : ''"
        @click="setFilter('all')"
      >
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs font-medium text-muted-foreground">全部连接</span>
          <ArrowLeftRight class="size-4 text-muted-foreground/50 group-hover:text-muted-foreground transition-colors" />
        </div>
        <p class="text-2xl font-black tracking-tighter">{{ stats.total }}</p>
      </button>

      <button
        class="bg-card border border-border rounded-xl p-4 text-left hover:border-emerald-500/30 transition-colors group"
        :class="statusFilter === 'active' ? 'border-emerald-500/40 ring-1 ring-emerald-500/10' : ''"
        @click="setFilter('active')"
      >
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs font-medium text-muted-foreground">已连接</span>
          <CheckCircle2 class="size-4 text-emerald-500/60 group-hover:text-emerald-500 transition-colors" />
        </div>
        <p class="text-2xl font-black tracking-tighter text-emerald-500">{{ stats.active }}</p>
      </button>

      <button
        class="bg-card border border-border rounded-xl p-4 text-left hover:border-amber-400/30 transition-colors group"
        :class="statusFilter === 'pending' ? 'border-amber-400/40 ring-1 ring-amber-400/10' : ''"
        @click="setFilter('pending')"
      >
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs font-medium text-muted-foreground">建立中</span>
          <Clock class="size-4 text-amber-400/60 group-hover:text-amber-400 transition-colors" />
        </div>
        <p class="text-2xl font-black tracking-tighter text-amber-400">{{ stats.pending }}</p>
      </button>

      <button
        class="bg-card border border-border rounded-xl p-4 text-left hover:border-rose-500/30 transition-colors group"
        :class="statusFilter === 'failed' ? 'border-rose-500/40 ring-1 ring-rose-500/10' : ''"
        @click="setFilter('failed')"
      >
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs font-medium text-muted-foreground">连接失败</span>
          <XCircle class="size-4 text-rose-500/60 group-hover:text-rose-500 transition-colors" />
        </div>
        <p class="text-2xl font-black tracking-tighter text-rose-500">{{ stats.failed }}</p>
      </button>
    </div>

    <!-- ── Toolbar ────────────────────────────────────────────────── -->
    <div class="flex items-center gap-2">
      <div class="relative w-72">
        <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
        <Input v-model="searchValue" placeholder="搜索连接名称、空间、命名空间..." class="pl-8 h-9" />
      </div>
      <div class="ml-auto flex items-center gap-2">
        <Button variant="outline" size="sm" class="gap-1.5" :disabled="loading" @click="fetchList">
          <RefreshCw class="size-3.5" :class="loading ? 'animate-spin' : ''" />
          刷新
        </Button>
        <Button size="sm" class="gap-1.5" @click="openCreate">
          <Plus class="size-3.5" /> 新建连接
        </Button>
      </div>
    </div>

    <!-- ── Connection cards ───────────────────────────────────────── -->
    <div v-if="loading" class="flex items-center justify-center py-28 text-muted-foreground text-sm">
      <RefreshCw class="size-4 animate-spin mr-2" /> 加载中...
    </div>

    <div v-else-if="paginated.length" class="grid gap-3 lg:grid-cols-2">
      <div
        v-for="conn in paginated"
        :key="conn.name"
        class="group bg-card border border-border rounded-xl overflow-hidden hover:shadow-md hover:border-primary/20 transition-all cursor-pointer"
        @click="openDetail(conn)"
      >
        <!-- Card header -->
        <div class="flex items-start justify-between px-4 pt-4 pb-3 gap-3">
          <div class="flex items-center gap-3 min-w-0">
            <div class="relative shrink-0 size-9 rounded-xl flex items-center justify-center"
              :class="conn.status === 'active' ? 'bg-emerald-500/10' : conn.status === 'pending' ? 'bg-amber-400/10' : 'bg-rose-500/10'">
              <component :is="statusConfig[conn.status]?.icon ?? AlertTriangle"
                class="size-4"
                :class="conn.status === 'active' ? 'text-emerald-500' : conn.status === 'pending' ? 'text-amber-400' : 'text-rose-500'"
              />
            </div>
            <div class="min-w-0">
              <p class="font-bold text-sm leading-none truncate">{{ conn.name }}</p>
              <p class="text-[11px] text-muted-foreground/60 mt-0.5 font-mono truncate">
                {{ conn.local.namespace }} ↔ {{ conn.remote.namespace }}
              </p>
            </div>
          </div>

          <div class="flex items-center gap-1.5 shrink-0" @click.stop>
            <span class="text-[10px] font-semibold px-2 py-0.5 rounded-full flex items-center gap-1"
              :class="statusConfig[conn.status]?.badge">
              <span class="size-1.5 rounded-full" :class="statusConfig[conn.status]?.dot" />
              {{ statusConfig[conn.status]?.label }}
            </span>
            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button variant="ghost" size="sm" class="size-7 p-0 opacity-0 group-hover:opacity-100 transition-opacity">
                  <MoreHorizontal class="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" class="w-36">
                <DropdownMenuItem @click="openDetail(conn)">
                  <Info class="mr-2 size-3.5" /> 查看详情
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem class="text-destructive focus:text-destructive" @click="promptDelete(conn)">
                  <Trash2 class="mr-2 size-3.5" /> 删除
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>

        <!-- Workspace path -->
        <div class="flex items-center gap-2 px-4 py-3 bg-muted/30 border-y border-border/60">
          <div class="flex-1 min-w-0">
            <p class="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/50 mb-1">本端空间</p>
            <p class="text-xs font-bold truncate">{{ conn.local.name }}</p>
            <p class="font-mono text-[10px] text-muted-foreground/60">{{ conn.local.cidr || '—' }}</p>
          </div>
          <div class="flex flex-col items-center gap-0.5 shrink-0">
            <span class="text-[10px] font-bold px-1.5 py-0.5 rounded" :class="modeConfig[conn.peeringMode]?.badge ?? modeConfig.gateway.badge">
              {{ modeConfig[conn.peeringMode]?.label ?? conn.peeringMode }}
            </span>
            <ArrowLeftRight class="size-3.5 text-muted-foreground/40" />
          </div>
          <div class="flex-1 min-w-0 text-right">
            <p class="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/50 mb-1">对端空间</p>
            <p class="text-xs font-bold truncate">{{ conn.remote.name }}</p>
            <p class="font-mono text-[10px] text-muted-foreground/60">{{ conn.remote.cidr || '—' }}</p>
          </div>
        </div>

        <!-- Footer stats -->
        <div class="flex items-center divide-x divide-border/60 px-4 py-2.5 text-center">
          <div class="flex-1 flex items-center justify-center gap-1.5 text-xs text-muted-foreground">
            <Network class="size-3 shrink-0" />
            {{ conn.local.nodeCount + conn.remote.nodeCount }} 节点
          </div>
          <div class="flex-1 flex items-center justify-center gap-1.5 text-xs text-muted-foreground">
            <Activity class="size-3 shrink-0" />
            <span class="font-mono text-[10px]">{{ conn.local.cidr || '分配中' }}</span>
          </div>
          <div class="flex-1 flex items-center justify-center gap-1.5 text-xs text-muted-foreground">
            <span class="text-[10px]">{{ new Date(conn.createdAt).toLocaleDateString('zh-CN') }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <div v-else class="flex flex-col items-center justify-center py-28 text-center">
      <div class="size-16 rounded-2xl bg-muted/40 flex items-center justify-center mb-4">
        <ArrowLeftRight class="size-7 text-muted-foreground/30" />
      </div>
      <p class="text-sm font-semibold text-muted-foreground">暂无对等连接</p>
      <p class="text-xs text-muted-foreground/50 mt-1">创建连接以打通不同工作空间之间的网络</p>
      <Button size="sm" class="mt-4 gap-1.5" @click="openCreate">
        <Plus class="size-3.5" /> 新建连接
      </Button>
    </div>

    <!-- ── Pagination ─────────────────────────────────────────────── -->
    <div v-if="totalPages > 1" class="flex items-center justify-between text-sm text-muted-foreground">
      <span>共 {{ filtered.length }} 条 · 第 {{ currentPage }} / {{ totalPages }} 页</span>
      <div class="flex items-center gap-1">
        <Button variant="outline" size="sm" class="size-8 p-0" :disabled="currentPage <= 1" @click="currentPage--">
          <ChevronLeft class="size-4" />
        </Button>
        <Button
          v-for="p in visiblePages" :key="p"
          variant="outline" size="sm" class="size-8 p-0 text-xs"
          :class="p === currentPage ? 'bg-primary text-primary-foreground border-primary' : ''"
          @click="currentPage = p"
        >{{ p }}</Button>
        <Button variant="outline" size="sm" class="size-8 p-0" :disabled="currentPage >= totalPages" @click="currentPage++">
          <ChevronRight class="size-4" />
        </Button>
      </div>
    </div>

    <!-- ── Delete confirm ─────────────────────────────────────────── -->
    <AppAlertDialog
      v-model:open="deleteDialogOpen"
      title="删除对等连接"
      :description="`确认删除「${deleteTarget?.name}」？双端路由规则将被清除，通信立即中断。`"
      confirm-text="删除"
      variant="destructive"
      @confirm="confirmDelete"
      @cancel="deleteTarget = null"
    />
  </div>

  <!-- ── Detail Dialog ───────────────────────────────────────────── -->
  <Dialog v-model:open="detailOpen">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2">
          <ArrowLeftRight class="size-4" />
          {{ selected?.name }}
        </DialogTitle>
        <DialogDescription>{{ selected?.local.namespace }} ↔ {{ selected?.remote.namespace }}</DialogDescription>
      </DialogHeader>

      <div v-if="selected" class="space-y-4 pt-1 max-h-[65vh] overflow-y-auto pr-1">

        <!-- Status + mode -->
        <div class="flex items-center gap-2 flex-wrap">
          <span class="text-xs font-semibold px-2.5 py-1 rounded-full flex items-center gap-1.5"
            :class="statusConfig[selected.status]?.badge">
            <component :is="statusConfig[selected.status]?.icon" class="size-3" />
            {{ statusConfig[selected.status]?.label }}
          </span>
          <span class="text-xs font-semibold px-2.5 py-1 rounded-full flex items-center gap-1.5"
            :class="modeConfig[selected.peeringMode]?.badge ?? modeConfig.gateway.badge">
            {{ modeConfig[selected.peeringMode]?.label ?? selected.peeringMode }}
            — {{ modeConfig[selected.peeringMode]?.tip ?? '' }}
          </span>
        </div>

        <!-- Endpoints -->
        <div class="rounded-lg border border-border overflow-hidden">
          <div class="grid grid-cols-2 divide-x divide-border/60">
            <div class="p-4 space-y-2">
              <p class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/50">本端空间</p>
              <p class="font-bold text-sm">{{ selected.local.name }}</p>
              <div class="space-y-1 text-xs text-muted-foreground">
                <div class="flex items-center justify-between">
                  <span>Namespace</span>
                  <span class="font-mono text-[11px]">{{ selected.local.namespace }}</span>
                </div>
                <div class="flex items-center justify-between">
                  <span>CIDR</span>
                  <span class="font-mono text-[11px] font-semibold text-foreground">{{ selected.local.cidr || '分配中' }}</span>
                </div>
                <div class="flex items-center justify-between">
                  <span>节点数</span>
                  <span class="font-semibold text-foreground">{{ selected.local.nodeCount }}</span>
                </div>
              </div>
            </div>
            <div class="p-4 space-y-2">
              <p class="text-[10px] font-bold uppercase tracking-widest text-muted-foreground/50">对端空间</p>
              <p class="font-bold text-sm">{{ selected.remote.name }}</p>
              <div class="space-y-1 text-xs text-muted-foreground">
                <div class="flex items-center justify-between">
                  <span>Namespace</span>
                  <span class="font-mono text-[11px]">{{ selected.remote.namespace }}</span>
                </div>
                <div class="flex items-center justify-between">
                  <span>CIDR</span>
                  <span class="font-mono text-[11px] font-semibold text-foreground">{{ selected.remote.cidr || '分配中' }}</span>
                </div>
                <div class="flex items-center justify-between">
                  <span>节点数</span>
                  <span class="font-semibold text-foreground">{{ selected.remote.nodeCount }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Metadata -->
        <div class="rounded-lg border border-border overflow-hidden divide-y divide-border/60">
          <div class="flex items-center justify-between px-4 py-2.5">
            <span class="text-xs text-muted-foreground">创建时间</span>
            <span class="text-xs">{{ new Date(selected.createdAt).toLocaleString('zh-CN') }}</span>
          </div>
        </div>

        <!-- Pending tip -->
        <div v-if="selected.status === 'pending'"
          class="flex gap-2 rounded-lg bg-amber-400/5 border border-amber-400/20 p-3">
          <Clock class="size-4 text-amber-400 shrink-0 mt-0.5" />
          <p class="text-xs text-muted-foreground leading-relaxed">
            Controller 正在建立通道：查找两端 Gateway 节点 → 配置路由 annotation → 创建 Shadow Peer → 应用访问策略。就绪后状态变为「已连接」。
          </p>
        </div>

        <!-- Failed tip -->
        <div v-if="selected.status === 'failed'"
          class="flex gap-2 rounded-lg bg-rose-500/5 border border-rose-500/20 p-3">
          <XCircle class="size-4 text-rose-500 shrink-0 mt-0.5" />
          <p class="text-xs text-muted-foreground leading-relaxed">
            连接建立失败，常见原因：两端网络尚未 Ready、未找到 Gateway 节点（需打标签 <code class="font-mono">wireflow.run/gateway=true</code>）。
          </p>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="detailOpen = false">关闭</Button>
        <Button variant="destructive" size="sm" @click="() => { detailOpen = false; promptDelete(selected!) }">
          <Trash2 class="size-3.5 mr-1.5" /> 删除连接
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <!-- ── Create Dialog ───────────────────────────────────────────── -->
  <Dialog v-model:open="createOpen">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>新建对等连接</DialogTitle>
        <DialogDescription>将当前工作空间与另一个空间通过加密隧道互通</DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-2">

        <!-- Local (read-only) -->
        <div class="space-y-1.5">
          <label class="text-xs font-medium">本端空间（当前）</label>
          <div class="h-9 rounded-md border border-input bg-muted/40 px-3 flex items-center text-sm text-muted-foreground">
            {{ workspaceStore.currentWorkspace?.displayName || workspaceStore.currentWorkspace?.namespace || '—' }}
            <span v-if="workspaceStore.currentWorkspace?.namespace" class="ml-2 font-mono text-[11px] text-muted-foreground/60">
              ({{ workspaceStore.currentWorkspace.namespace }})
            </span>
          </div>
        </div>

        <!-- Remote namespace -->
        <div class="space-y-1.5">
          <label class="text-xs font-medium">对端命名空间 <span class="text-destructive">*</span></label>
          <Input
            v-model="createForm.namespaceB"
            placeholder="例如：wf-workspace-b"
            class="font-mono text-xs h-9"
          />
          <p class="text-[10px] text-muted-foreground/60">对端工作空间的 K8s namespace</p>
        </div>

        <!-- Remote network (optional) -->
        <div class="space-y-1.5">
          <label class="text-xs font-medium">对端网络名称 <span class="text-muted-foreground font-normal">(可选)</span></label>
          <Input
            v-model="createForm.networkB"
            placeholder="留空则使用 wireflow-default-net"
            class="font-mono text-xs h-9"
          />
        </div>

        <!-- Connection name (optional) -->
        <div class="space-y-1.5">
          <label class="text-xs font-medium">连接名称 <span class="text-muted-foreground font-normal">(可选)</span></label>
          <Input v-model="createForm.name" placeholder="留空则自动生成" class="font-mono text-xs h-9" />
        </div>

        <!-- Mode selector -->
        <div class="space-y-2">
          <label class="text-xs font-medium">路由模式</label>
          <div class="grid grid-cols-2 gap-2">
            <button
              v-for="mode in (['gateway', 'mesh'] as PeeringMode[])" :key="mode"
              class="p-3 rounded-lg border text-left transition-all"
              :class="createForm.peeringMode === mode
                ? 'border-primary bg-primary/5'
                : 'border-border hover:border-primary/30'"
              @click="createForm.peeringMode = mode"
            >
              <p class="text-xs font-bold" :class="createForm.peeringMode === mode ? 'text-primary' : ''">
                {{ modeConfig[mode].label }}
              </p>
              <p class="text-[10px] text-muted-foreground/60 mt-0.5 leading-tight">{{ modeConfig[mode].tip }}</p>
            </button>
          </div>
        </div>

        <!-- Info tip -->
        <div class="flex gap-2 rounded-lg bg-primary/5 border border-primary/10 p-3">
          <Zap class="size-4 text-primary shrink-0 mt-0.5" />
          <p class="text-xs text-muted-foreground leading-relaxed">
            提交后 Controller 自动配置路由 annotation、Shadow Peer 和访问策略。需要两端各有一个打了
            <code class="font-mono">wireflow.run/gateway=true</code> 标签的 Gateway 节点。
          </p>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="createOpen = false">取消</Button>
        <Button
          :disabled="!createForm.namespaceB.trim() || createLoading"
          @click="handleCreate"
        >
          <RefreshCw v-if="createLoading" class="size-3.5 mr-1.5 animate-spin" />
          <ArrowLeftRight v-else class="size-3.5 mr-1.5" />
          发起连接
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
