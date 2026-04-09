<script setup lang="ts">
import { ref } from 'vue'
import { add, listUser, deleteUser } from '@/api/user'
import { listWs } from '@/api/workspace'
import { useTable, useAction } from '@/composables/useApi'
import { useConfirm } from '@/composables/useConfirm'
import {
  Users, Plus, RefreshCw, Trash2, MoreHorizontal,
  Pencil, Key, Clock, Server,
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

definePage({
  meta: { title: '用户管理', description: '管理平台成员与 RBAC 权限。' },
})

const { confirm } = useConfirm()

// ── API ───────────────────────────────────────────────────────────
const { rows: members, total, loading, params, refresh } = useTable(listUser)
const { rows: workspaces } = useTable(listWs)

const { loading: addLoading, execute: runAdd } = useAction(add, {
  successMsg: '成员添加成功',
  onSuccess: () => { dialogOpen.value = false; refresh() },
})

// ── 状态 ──────────────────────────────────────────────────────────
const dialogOpen = ref(false)
const dialogType = ref<'invite' | 'config'>('invite')
const selectedMember = ref<any>(null)

const form = ref({
  username: '', password: '', role: 'viewer', namespace: '',
  provider: 'local' as 'local' | 'dex',
})

function openInvite() {
  dialogType.value = 'invite'
  form.value = { username: '', password: '', role: 'viewer', namespace: '', provider: 'local' }
  dialogOpen.value = true
}

function openConfig(m: any) {
  selectedMember.value = JSON.parse(JSON.stringify(m))
  dialogType.value = 'config'
  dialogOpen.value = true
}

async function handleDelete(m: any) {
  const ok = await confirm({
    title: '确认移除成员？',
    message: `确认从团队中移除 ${m.name}？此操作将同步撤销其 RBAC 绑定。`,
    type: 'danger',
    confirmText: '确认移除',
  })
  if (ok) { await deleteUser(m.id); refresh() }
}

// ── 样式辅助 ──────────────────────────────────────────────────────
const roleStyle: Record<string, string> = {
  admin:  'bg-primary/10 text-primary ring-1 ring-primary/20',
  editor: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 ring-1 ring-emerald-500/20',
  viewer: 'bg-muted text-muted-foreground ring-1 ring-border',
}
const statusStyle: Record<string, string> = {
  active:  'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 ring-1 ring-emerald-500/20',
  pending: 'bg-amber-400/10 text-amber-600 dark:text-amber-400 ring-1 ring-amber-400/20',
}

const avatarColors = [
  'bg-blue-500', 'bg-violet-500', 'bg-emerald-500',
  'bg-orange-500', 'bg-rose-500', 'bg-cyan-500', 'bg-indigo-500',
]
function avatarColor(name: string) {
  let h = 0
  for (const c of (name ?? '')) h = (h * 31 + c.charCodeAt(0)) & 0xff
  return avatarColors[h % avatarColors.length]
}
function firstChar(name: string) {
  return name?.trim().charAt(0).toUpperCase() ?? '?'
}
function nsBadgeStyle(name: string) {
  const hues = [210, 160, 260, 40, 0, 190, 230]
  let h = 0
  for (const c of (name ?? '')) h = (h * 31 + c.charCodeAt(0)) & 0xff
  const hue = hues[h % hues.length]
  return {
    backgroundColor: `hsla(${hue}, 70%, 50%, 0.12)`,
    color: `hsla(${hue}, 80%, 60%, 1)`,
    outline: `1px solid hsla(${hue}, 70%, 50%, 0.2)`,
  }
}

const totalPages = () => Math.ceil(total.value / params.pageSize)
</script>

<template>
  <div class="p-6 space-y-5 animate-in fade-in duration-300">

    <!-- ── Toolbar ────────────────────────────────────────────────── -->
    <div class="flex items-center justify-between">
      <p class="text-sm text-muted-foreground">
        共 <span class="font-semibold text-foreground">{{ total }}</span> 位成员
      </p>
      <div class="flex items-center gap-2">
        <Button variant="outline" size="sm" class="gap-1.5" :disabled="loading" @click="refresh">
          <RefreshCw class="size-3.5" :class="loading ? 'animate-spin' : ''" />
          刷新
        </Button>
        <Button size="sm" class="gap-1.5" @click="openInvite">
          <Plus class="size-3.5" /> 添加成员
        </Button>
      </div>
    </div>

    <!-- ── Skeleton ───────────────────────────────────────────────── -->
    <div v-if="loading && members.length === 0" class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <div v-for="i in 3" :key="i" class="h-52 rounded-xl bg-muted/50 animate-pulse" />
    </div>

    <!-- ── Member cards ───────────────────────────────────────────── -->
    <div v-else-if="members.length > 0" class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <div
        v-for="m in members" :key="m.id"
        class="group bg-card border border-border rounded-xl flex flex-col overflow-hidden hover:shadow-md hover:border-primary/20 transition-all duration-200"
      >
        <!-- Card header -->
        <div class="p-5 flex items-start justify-between gap-3">
          <div class="flex items-center gap-3 min-w-0">
            <div
              class="size-10 rounded-xl flex items-center justify-center text-white text-sm font-black shrink-0 transition-transform group-hover:scale-105"
              :class="avatarColor(m.name)"
            >
              {{ firstChar(m.name) }}
            </div>
            <div class="min-w-0">
              <p class="text-sm font-bold truncate group-hover:text-primary transition-colors">{{ m.name }}</p>
              <p class="text-[11px] font-mono text-muted-foreground/60 truncate">{{ m.email }}</p>
            </div>
          </div>

          <div class="flex items-center gap-1.5 shrink-0">
            <span
              class="text-[10px] font-semibold px-2 py-0.5 rounded-full flex items-center gap-1"
              :class="statusStyle[m.status] ?? statusStyle.pending"
            >
              <span
                class="size-1.5 rounded-full"
                :class="m.status === 'active' ? 'bg-emerald-500 animate-pulse' : 'bg-amber-400'"
              />
              {{ m.status === 'active' ? '活跃' : '待激活' }}
            </span>

            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button variant="ghost" size="sm" class="size-7 p-0 opacity-0 group-hover:opacity-100 transition-opacity">
                  <MoreHorizontal class="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" class="w-36">
                <DropdownMenuItem @click="openConfig(m)">
                  <Pencil class="size-3.5 mr-2" /> 编辑权限
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem class="text-destructive focus:text-destructive" @click="handleDelete(m)">
                  <Trash2 class="size-3.5 mr-2" /> 移除
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>

        <!-- Stats strip -->
        <div class="flex items-center border-y border-border bg-muted/20 px-5 py-2.5 gap-4">
          <div class="flex-1 text-center">
            <p class="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/50 mb-0.5">角色</p>
            <span class="text-[10px] font-black uppercase px-2 py-0.5 rounded-md" :class="roleStyle[m.role] ?? roleStyle.viewer">
              {{ m.role ?? 'viewer' }}
            </span>
          </div>
          <div class="w-px h-6 bg-border" />
          <div class="flex-1 text-center">
            <p class="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/50 mb-0.5">空间数</p>
            <p class="text-sm font-black">{{ m.bindings?.length ?? 0 }}</p>
          </div>
          <div class="w-px h-6 bg-border" />
          <div class="flex-1 text-center">
            <p class="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/50 mb-0.5">Provider</p>
            <p class="text-[10px] font-black uppercase">{{ m.provider ?? 'local' }}</p>
          </div>
        </div>

        <!-- Namespace bindings -->
        <div class="px-5 py-4 flex-1">
          <p class="text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/40 mb-2">Namespace Bindings</p>
          <div class="flex flex-wrap gap-1.5 min-h-5">
            <span
              v-for="b in m.bindings" :key="b.ns"
              :style="nsBadgeStyle(b.ns)"
              class="px-2 py-0.5 rounded-md text-[10px] font-bold"
            >
              {{ b.ns }}
            </span>
            <span v-if="!m.bindings?.length" class="text-[11px] italic text-muted-foreground/30">Unassigned</span>
          </div>
        </div>

        <!-- Card footer -->
        <div class="px-5 py-3.5 border-t border-border bg-muted/10 flex items-center justify-between">
          <p class="text-[11px] text-muted-foreground/50 flex items-center gap-1">
            <Clock class="size-3" /> {{ m.lastActive ?? '从未登录' }}
          </p>
          <Button
            size="sm"
            class="h-7 text-xs gap-1 px-3 opacity-0 group-hover:opacity-100 transition-opacity"
            @click="openConfig(m)"
          >
            编辑权限
          </Button>
        </div>
      </div>
    </div>

    <!-- ── Empty state ────────────────────────────────────────────── -->
    <div v-else class="flex flex-col items-center justify-center py-28 text-center">
      <div class="size-16 rounded-2xl bg-muted/40 flex items-center justify-center mb-4">
        <Users class="size-7 text-muted-foreground/30" />
      </div>
      <p class="text-sm font-semibold text-muted-foreground">暂无团队成员</p>
      <p class="text-xs text-muted-foreground/50 mt-1">点击添加按钮邀请第一位成员</p>
      <Button size="sm" class="mt-4 gap-1.5" @click="openInvite">
        <Plus class="size-3.5" /> 添加成员
      </Button>
    </div>

    <!-- ── Pagination ─────────────────────────────────────────────── -->
    <div v-if="total > params.pageSize" class="flex items-center justify-end gap-1">
      <Button variant="outline" size="sm" class="h-7 text-xs px-2.5" :disabled="params.page <= 1" @click="params.page--">上一页</Button>
      <span class="text-xs text-muted-foreground px-2">{{ params.page }} / {{ totalPages() || 1 }}</span>
      <Button variant="outline" size="sm" class="h-7 text-xs px-2.5" :disabled="params.page >= totalPages()" @click="params.page++">下一页</Button>
    </div>

  </div>

  <!-- ── Dialog ─────────────────────────────────────────────────── -->
  <Dialog v-model:open="dialogOpen">
    <DialogContent class="sm:max-w-md">

      <!-- Invite -->
      <template v-if="dialogType === 'invite'">
        <DialogHeader>
          <DialogTitle>添加成员</DialogTitle>
          <DialogDescription>向平台添加新用户并分配初始权限</DialogDescription>
        </DialogHeader>

        <div class="space-y-4 py-2">
          <!-- Provider toggle -->
          <div class="flex bg-muted/50 rounded-lg p-1 border border-border">
            <button
              class="flex-1 py-1.5 rounded-md text-xs font-bold uppercase tracking-widest transition-all"
              :class="form.provider === 'local' ? 'bg-background text-primary shadow-sm' : 'text-muted-foreground'"
              @click="form.provider = 'local'"
            >
              Local
            </button>
            <button
              class="flex-1 py-1.5 rounded-md text-xs font-bold uppercase tracking-widest transition-all"
              :class="form.provider === 'dex' ? 'bg-background text-primary shadow-sm' : 'text-muted-foreground'"
              @click="form.provider = 'dex'"
            >
              OIDC / Dex
            </button>
          </div>

          <div class="space-y-1.5">
            <label class="text-xs font-medium">用户名 / Email</label>
            <Input v-model="form.username" placeholder="ops@example.com" />
          </div>

          <div v-if="form.provider === 'local'" class="space-y-1.5">
            <label class="text-xs font-medium">初始密码</label>
            <Input v-model="form.password" type="password" />
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div class="space-y-1.5">
              <label class="text-xs font-medium">系统角色</label>
              <select
                v-model="form.role"
                class="w-full h-9 rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 transition-[color,box-shadow]"
              >
                <option value="admin">Admin</option>
                <option value="editor">Editor</option>
                <option value="viewer">Viewer</option>
              </select>
            </div>
            <div class="space-y-1.5">
              <label class="text-xs font-medium">初始空间</label>
              <select
                v-model="form.namespace"
                class="w-full h-9 rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 transition-[color,box-shadow]"
              >
                <option value="">不指定</option>
                <option v-for="ws in workspaces" :key="ws.id" :value="ws.id">{{ ws.displayName }}</option>
              </select>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="dialogOpen = false">取消</Button>
          <Button :disabled="addLoading" @click="runAdd(form)">
            <RefreshCw v-if="addLoading" class="size-3.5 animate-spin mr-2" />
            添加成员
          </Button>
        </DialogFooter>
      </template>

      <!-- Config -->
      <template v-else-if="selectedMember">
        <DialogHeader>
          <DialogTitle>编辑权限</DialogTitle>
          <DialogDescription>管理 {{ selectedMember.name }} 的命名空间绑定与角色</DialogDescription>
        </DialogHeader>

        <div class="space-y-4 py-2">
          <!-- Member info -->
          <div class="flex items-center gap-3 p-3 bg-muted/30 rounded-lg border border-border">
            <div class="size-10 rounded-xl flex items-center justify-center text-white text-sm font-black shrink-0" :class="avatarColor(selectedMember.name)">
              {{ firstChar(selectedMember.name) }}
            </div>
            <div>
              <p class="text-sm font-bold">{{ selectedMember.name }}</p>
              <p class="text-xs font-mono text-muted-foreground/60">{{ selectedMember.email }}</p>
            </div>
          </div>

          <!-- Role -->
          <div class="space-y-1.5">
            <label class="text-xs font-medium">系统角色</label>
            <select
              v-model="selectedMember.role"
              class="w-full h-9 rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 transition-[color,box-shadow]"
            >
              <option value="admin">Admin</option>
              <option value="editor">Editor</option>
              <option value="viewer">Viewer</option>
            </select>
          </div>

          <!-- Bindings -->
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <label class="text-xs font-medium">Namespace Bindings</label>
            </div>
            <div class="space-y-2 max-h-44 overflow-y-auto">
              <div
                v-for="(b, i) in selectedMember.bindings" :key="i"
                class="flex items-center justify-between p-3 rounded-lg border border-border bg-muted/20"
              >
                <div class="flex items-center gap-2">
                  <Server class="size-3.5 text-muted-foreground/50" />
                  <span class="text-xs font-bold">{{ b.ns }}</span>
                </div>
                <select class="h-7 rounded-md border border-input bg-background px-2 text-xs focus-visible:outline-none w-28">
                  <option>Admin</option>
                  <option>Editor</option>
                  <option>Viewer</option>
                </select>
              </div>
              <p v-if="!selectedMember.bindings?.length" class="text-xs text-muted-foreground/40 italic py-1 text-center">暂无命名空间绑定</p>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" @click="dialogOpen = false">取消</Button>
          <Button @click="dialogOpen = false">
            <Key class="size-3.5 mr-2" /> 保存权限
          </Button>
        </DialogFooter>
      </template>

    </DialogContent>
  </Dialog>
</template>
