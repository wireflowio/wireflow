<script setup lang="ts">
import { onMounted } from 'vue'
import {
  Shield, Plus, RefreshCw, Pencil, Trash2,
  ArrowDown, ArrowUp, MoreHorizontal, Info,
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
import { toast } from 'vue-sonner'
import { usePolicyPageStore } from '@/stores/usePolicyPageStore'

definePage({
  meta: { title: '策略管理', description: '管理网络访问控制策略。' },
})

const store = usePolicyPageStore()
onMounted(() => store.actions.refresh())

// ── 样式辅助 ──────────────────────────────────────────────────────
const actionBadge: Record<string, string> = {
  Allow: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 ring-1 ring-emerald-500/20',
  Deny:  'bg-rose-500/10 text-rose-600 dark:text-rose-400 ring-1 ring-rose-500/20',
}
const actionIcon: Record<string, string> = {
  Allow: 'text-emerald-500', Deny: 'text-rose-500',
}
const typeBadge: Record<string, string> = {
  Ingress: 'bg-blue-500/10 text-blue-600 dark:text-blue-400 ring-1 ring-blue-500/20',
  Egress:  'bg-violet-500/10 text-violet-600 dark:text-violet-400 ring-1 ring-violet-500/20',
}

// 快速模板
const templates = [
  { key: 'isolate',  label: '全隔离',    desc: 'Deny All In/Out' },
  { key: 'db',       label: '数据库保护', desc: 'Postgres Ingress' },
  { key: 'internet', label: '放通出口',   desc: 'Allow HTTPS Out' },
]

// 分页
const totalPages = () => Math.ceil(store.total / store.params.pageSize)
</script>

<template>
  <div class="p-6 space-y-5 animate-in fade-in duration-300">

    <!-- ── Toolbar ────────────────────────────────────────────────── -->
    <div class="flex items-center justify-between">
      <p class="text-sm text-muted-foreground">
        共 <span class="font-semibold text-foreground">{{ store.total }}</span> 条策略
      </p>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="sm" class="gap-1.5" :disabled="store.loading" @click="store.actions.refresh()">
          <RefreshCw class="size-3.5" :class="store.loading ? 'animate-spin' : ''" />
          刷新
        </Button>
        <Button size="sm" class="gap-1.5" @click="store.actions.openDrawer('create')">
          <Plus class="size-3.5" /> 新建策略
        </Button>
      </div>
    </div>

    <!-- ── Skeleton ───────────────────────────────────────────────── -->
    <div v-if="store.loading && store.rows.length === 0" class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <div v-for="i in 3" :key="i" class="h-48 rounded-xl bg-muted/50 animate-pulse" />
    </div>

    <!-- ── Policy cards ───────────────────────────────────────────── -->
    <div v-else-if="store.rows.length > 0" class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <div
        v-for="policy in store.rows" :key="policy.name"
        class="group bg-card border border-border rounded-xl flex flex-col overflow-hidden hover:shadow-md hover:border-primary/20 transition-all duration-200"
      >
        <!-- Card header -->
        <div class="p-5 flex items-start justify-between gap-3">
          <div class="flex items-center gap-3 min-w-0">
            <div
              class="size-10 rounded-lg flex items-center justify-center shrink-0"
              :class="policy.action === 'Deny' ? 'bg-rose-500/10' : 'bg-emerald-500/10'"
            >
              <Shield class="size-5" :class="actionIcon[policy.action] ?? actionIcon.Allow" />
            </div>
            <div class="min-w-0">
              <p class="text-sm font-bold truncate">{{ policy.name }}</p>
              <p class="text-[11px] text-muted-foreground/60 truncate mt-0.5">
                {{ policy.description || '无描述' }}
              </p>
            </div>
          </div>

          <div class="flex items-center gap-1.5 shrink-0">
            <span class="text-[10px] font-semibold px-2 py-0.5 rounded-full" :class="actionBadge[policy.action] ?? actionBadge.Allow">
              {{ policy.action ?? 'Allow' }}
            </span>
            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button variant="ghost" size="sm" class="size-7 p-0 opacity-0 group-hover:opacity-100 transition-opacity">
                  <MoreHorizontal class="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" class="w-36">
                <DropdownMenuItem @click="store.actions.openDrawer('edit', policy)">
                  <Pencil class="size-3.5 mr-2" /> 编辑
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  class="text-destructive focus:text-destructive"
                  @click="store.actions.handleDelete(policy, toast)"
                >
                  <Trash2 class="size-3.5 mr-2" /> 删除
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>

        <!-- Stats strip -->
        <div class="flex items-center border-y border-border bg-muted/20 px-5 py-2.5 gap-4">
          <div class="flex-1 text-center">
            <p class="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/50 mb-0.5">Ingress</p>
            <p class="text-sm font-black">{{ policy.ingress?.length ?? 0 }}</p>
          </div>
          <div class="w-px h-6 bg-border" />
          <div class="flex-1 text-center">
            <p class="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/50 mb-0.5">Egress</p>
            <p class="text-sm font-black">{{ policy.egress?.length ?? 0 }}</p>
          </div>
          <div class="w-px h-6 bg-border" />
          <div class="flex-1 text-center">
            <p class="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/50 mb-0.5">规则数</p>
            <p class="text-sm font-black">{{ (policy.ingress?.length ?? 0) + (policy.egress?.length ?? 0) }}</p>
          </div>
        </div>

        <!-- Target & types -->
        <div class="px-5 py-4 space-y-3 flex-1">
          <!-- Target selector labels -->
          <div>
            <p class="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/40 mb-1.5">目标选择器</p>
            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="(val, key) in policy.peerSelector?.matchLabels" :key="key"
                class="px-2 py-0.5 rounded-md text-[10px] font-mono font-medium bg-muted/60 text-muted-foreground ring-1 ring-border"
              >
                {{ key }}={{ val }}
              </span>
              <span v-if="!Object.keys(policy.peerSelector?.matchLabels ?? {}).length" class="text-[11px] text-muted-foreground/30 italic">未设置</span>
            </div>
          </div>

          <!-- Policy type badges -->
          <div class="flex gap-1.5">
            <span
              v-for="t in policy.policyTypes" :key="t"
              class="flex items-center gap-1 px-2 py-0.5 rounded-md text-[10px] font-bold"
              :class="typeBadge[t] ?? 'bg-muted text-muted-foreground'"
            >
              <ArrowDown v-if="t === 'Ingress'" class="size-3" />
              <ArrowUp v-else class="size-3" />
              {{ t }}
            </span>
          </div>
        </div>

        <!-- Card footer -->
        <div class="px-5 py-3.5 border-t border-border bg-muted/10 flex items-center justify-between">
          <p class="text-[11px] text-muted-foreground/50 font-mono">{{ policy.name }}</p>
          <Button
            variant="ghost" size="sm"
            class="h-7 text-xs text-primary hover:text-primary hover:bg-primary/10 gap-1 opacity-0 group-hover:opacity-100 transition-opacity"
            @click="store.actions.openDrawer('edit', policy)"
          >
            <Pencil class="size-3" /> 编辑
          </Button>
        </div>
      </div>
    </div>

    <!-- ── Empty state ────────────────────────────────────────────── -->
    <div v-else class="flex flex-col items-center justify-center py-28 text-center">
      <div class="size-16 rounded-2xl bg-muted/40 flex items-center justify-center mb-4">
        <Shield class="size-7 text-muted-foreground/30" />
      </div>
      <p class="text-sm font-semibold text-muted-foreground">暂无访问控制策略</p>
      <p class="text-xs text-muted-foreground/50 mt-1">点击新建按钮创建第一条网络策略</p>
      <Button size="sm" class="mt-4 gap-1.5" @click="store.actions.openDrawer('create')">
        <Plus class="size-3.5" /> 新建策略
      </Button>
    </div>

    <!-- ── Pagination ─────────────────────────────────────────────── -->
    <div v-if="store.total > store.params.pageSize" class="flex items-center justify-end gap-1">
      <Button variant="outline" size="sm" class="h-7 text-xs px-2.5" :disabled="store.params.page <= 1" @click="store.params.page--">上一页</Button>
      <span class="text-xs text-muted-foreground px-2">{{ store.params.page }} / {{ totalPages() }}</span>
      <Button variant="outline" size="sm" class="h-7 text-xs px-2.5" :disabled="store.params.page >= totalPages()" @click="store.params.page++">下一页</Button>
    </div>

  </div>

  <!-- ── Create / Edit Dialog ───────────────────────────────────── -->
  <Dialog :open="store.ui.isDrawerOpen" @update:open="v => { if (!v) store.ui.isDrawerOpen = false }">
    <DialogContent class="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{{ store.ui.drawerType === 'create' ? '新建策略' : '编辑策略' }}</DialogTitle>
        <DialogDescription>
          {{ store.ui.drawerType === 'create' ? '定义一条网络访问控制规则' : '修改策略配置' }}
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-1">

        <!-- Quick templates -->
        <div v-if="store.ui.drawerType === 'create'" class="grid grid-cols-3 gap-2">
          <button
            v-for="tpl in templates" :key="tpl.key"
            class="p-2.5 rounded-lg border border-border bg-muted/20 hover:border-primary/40 hover:bg-primary/5 transition-all text-left"
            @click="store.actions.applyTemplate(tpl.key)"
          >
            <p class="text-xs font-bold">{{ tpl.label }}</p>
            <p class="text-[10px] text-muted-foreground/60 mt-0.5">{{ tpl.desc }}</p>
          </button>
        </div>

        <div class="grid grid-cols-2 gap-3">
          <!-- Name -->
          <div class="space-y-1.5 col-span-2">
            <label class="text-xs font-medium">策略名称</label>
            <Input v-model="store.form.name" placeholder="例如：deny-all-egress" class="font-mono text-xs" />
          </div>

          <!-- Target label -->
          <div class="space-y-1.5 col-span-2">
            <label class="text-xs font-medium">
              目标选择器
              <span class="text-muted-foreground font-normal ml-1 font-mono text-[10px]">key=value</span>
            </label>
            <Input v-model="store.form._targetLabel" placeholder="app=web" class="font-mono text-xs" />
          </div>

          <!-- Action -->
          <div class="space-y-1.5">
            <label class="text-xs font-medium">动作</label>
            <select
              v-model="store.form.action"
              class="w-full h-9 rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 transition-[color,box-shadow]"
            >
              <option value="Allow">Allow</option>
              <option value="Deny">Deny</option>
            </select>
          </div>

          <!-- Policy types -->
          <div class="space-y-1.5">
            <label class="text-xs font-medium">策略方向</label>
            <div class="flex gap-2 h-9 items-center">
              <label
                v-for="t in ['Ingress', 'Egress']" :key="t"
                class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border cursor-pointer transition-all select-none text-xs font-semibold"
                :class="store.form.policyTypes.includes(t)
                  ? (t === 'Ingress' ? 'border-blue-500/50 bg-blue-500/8 text-blue-600 dark:text-blue-400' : 'border-violet-500/50 bg-violet-500/8 text-violet-600 dark:text-violet-400')
                  : 'border-border text-muted-foreground'"
              >
                <input
                  type="checkbox"
                  :checked="store.form.policyTypes.includes(t)"
                  class="sr-only"
                  @change="store.form.policyTypes.includes(t)
                    ? store.form.policyTypes.splice(store.form.policyTypes.indexOf(t), 1)
                    : store.form.policyTypes.push(t)"
                />
                <ArrowDown v-if="t === 'Ingress'" class="size-3.5" />
                <ArrowUp v-else class="size-3.5" />
                {{ t }}
              </label>
            </div>
          </div>

          <!-- Description -->
          <div class="space-y-1.5 col-span-2">
            <label class="text-xs font-medium">描述 <span class="text-muted-foreground font-normal">(可选)</span></label>
            <Input v-model="store.form.description" placeholder="简要说明此策略的用途..." />
          </div>
        </div>

        <!-- Ingress rules -->
        <div v-if="store.form.policyTypes.includes('Ingress')" class="space-y-2">
          <div class="flex items-center justify-between">
            <p class="text-xs font-semibold flex items-center gap-1.5 text-blue-600 dark:text-blue-400">
              <ArrowDown class="size-3.5" /> Ingress 规则
            </p>
            <Button variant="ghost" size="sm" class="h-6 text-[11px] text-primary font-bold px-2" @click="store.actions.addRule('ingress')">
              + 添加
            </Button>
          </div>
          <div v-for="(rule, i) in store.form.ingress" :key="i" class="grid grid-cols-2 gap-2 p-3 rounded-lg border border-border bg-muted/20">
            <div class="space-y-1">
              <p class="text-[10px] text-muted-foreground/50 uppercase font-semibold">来源选择器</p>
              <Input v-model="rule._rawLabel" placeholder="app=frontend" class="h-7 text-xs font-mono" />
            </div>
            <div class="space-y-1">
              <p class="text-[10px] text-muted-foreground/50 uppercase font-semibold">端口</p>
              <Input v-model="rule.ports[0].port" placeholder="80" class="h-7 text-xs font-mono" />
            </div>
          </div>
          <p v-if="!store.form.ingress.length" class="text-xs text-muted-foreground/40 italic">无规则 — 拒绝所有入站</p>
        </div>

        <!-- Egress rules -->
        <div v-if="store.form.policyTypes.includes('Egress')" class="space-y-2">
          <div class="flex items-center justify-between">
            <p class="text-xs font-semibold flex items-center gap-1.5 text-violet-600 dark:text-violet-400">
              <ArrowUp class="size-3.5" /> Egress 规则
            </p>
            <Button variant="ghost" size="sm" class="h-6 text-[11px] text-primary font-bold px-2" @click="store.actions.addRule('egress')">
              + 添加
            </Button>
          </div>
          <div v-for="(rule, i) in store.form.egress" :key="i" class="grid grid-cols-2 gap-2 p-3 rounded-lg border border-border bg-muted/20">
            <div class="space-y-1">
              <p class="text-[10px] text-muted-foreground/50 uppercase font-semibold">目标选择器</p>
              <Input v-model="rule._rawLabel" placeholder="app=db" class="h-7 text-xs font-mono" />
            </div>
            <div class="space-y-1">
              <p class="text-[10px] text-muted-foreground/50 uppercase font-semibold">端口</p>
              <Input v-model="rule.ports[0].port" placeholder="5432" class="h-7 text-xs font-mono" />
            </div>
          </div>
          <p v-if="!store.form.egress.length" class="text-xs text-muted-foreground/40 italic">无规则 — 拒绝所有出站</p>
        </div>

        <!-- Hint -->
        <div class="flex gap-2 rounded-lg bg-primary/5 border border-primary/10 p-3">
          <Info class="size-4 text-primary shrink-0 mt-0.5" />
          <p class="text-xs text-muted-foreground leading-relaxed">
            策略将以 <code class="font-mono text-xs">WireflowPolicy</code> CRD 形式同步至集群，生效可能需要数秒。
          </p>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="store.ui.isDrawerOpen = false">取消</Button>
        <Button :disabled="store.loading" @click="store.actions.handleCreateOrUpdate(toast)">
          <RefreshCw v-if="store.loading" class="size-3.5 animate-spin mr-2" />
          {{ store.ui.drawerType === 'create' ? '发布策略' : '保存更改' }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
