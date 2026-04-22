<script setup lang="ts">
import { ref, computed, watch, onMounted, h } from 'vue'
import {
  useVueTable, getCoreRowModel, getPaginationRowModel,
  FlexRender, type ColumnDef,
} from '@tanstack/vue-table'
import {
  Users, Plus, RefreshCw, MoreHorizontal, Pencil,
  Trash2, Search,
  Shield, UserCheck, User, Eye, Clock, Mail,
  CheckCircle2, AlertCircle, XCircle,
  ArrowUpRight, ArrowDownRight,
} from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '@/components/ui/table'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle,
  DialogDescription, DialogFooter,
} from '@/components/ui/dialog'
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem,
  DropdownMenuSeparator, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import AppAlertDialog from '@/components/AlertDialog.vue'
import DataTablePagination from '@/components/DataTablePagination.vue'
import { listMembers, updateMemberRole, removeMember } from '@/api/member'
import { listInvitations, createInvitation, revokeInvitation } from '@/api/invitation'
import { useTable } from '@/composables/useApi'
import { toast } from 'vue-sonner'

definePage({
  meta: { title: '成员管理', description: '管理 Workspace 成员与邀请。' },
})

// ── Active tab ────────────────────────────────────────────────────
type Tab = 'members' | 'invitations'
const activeTab = ref<Tab>('members')

// ── Members API ───────────────────────────────────────────────────
const { rows: members, total: memberTotal, loading: memberLoading, refresh: refreshMembers } = useTable(listMembers)

// ── Invitations API ───────────────────────────────────────────────
const { rows: invitations, total: invTotal, loading: invLoading, refresh: refreshInvitations } = useTable(listInvitations)

onMounted(() => { refreshMembers(); refreshInvitations() })

// ── Style helpers ─────────────────────────────────────────────────
const roleStyle: Record<string, string> = {
  admin:  'bg-primary/10 text-primary ring-1 ring-primary/20',
  editor: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 ring-1 ring-emerald-500/20',
  member: 'bg-violet-500/10 text-violet-600 dark:text-violet-400 ring-1 ring-violet-500/20',
  viewer: 'bg-muted text-muted-foreground ring-1 ring-border',
}
const roleLabel: Record<string, string> = {
  admin: '管理员', editor: '编辑者', member: '成员', viewer: '访客',
}
const roleIcon: Record<string, any> = {
  admin: Shield, editor: UserCheck, member: User, viewer: Eye,
}
const providerStyle: Record<string, string> = {
  local: 'bg-blue-500/10 text-blue-600 dark:text-blue-400 ring-1 ring-blue-500/20',
  dex:   'bg-violet-500/10 text-violet-600 dark:text-violet-400 ring-1 ring-violet-500/20',
  ldap:  'bg-orange-500/10 text-orange-600 dark:text-orange-400 ring-1 ring-orange-500/20',
}
const invStatusStyle: Record<string, string> = {
  pending:  'bg-amber-400/10 text-amber-600 dark:text-amber-400 ring-1 ring-amber-400/20',
  accepted: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 ring-1 ring-emerald-500/20',
  expired:  'bg-red-500/10 text-red-500 ring-1 ring-red-500/20',
  revoked:  'bg-muted text-muted-foreground ring-1 ring-border',
}
const invStatusLabel: Record<string, string> = {
  pending: '待接受', accepted: '已接受', expired: '已过期', revoked: '已撤销',
}

const avatarColors = [
  'bg-blue-500', 'bg-violet-500', 'bg-emerald-500',
  'bg-orange-500', 'bg-rose-500', 'bg-cyan-500', 'bg-indigo-500',
]
function avatarColor(name: string) {
  let hash = 0
  for (const c of (name ?? '')) hash = (hash * 31 + c.charCodeAt(0)) & 0xff
  return avatarColors[hash % avatarColors.length]
}
function firstChar(name: string) {
  return name?.trim().charAt(0).toUpperCase() ?? '?'
}
function formatDate(iso?: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' })
}

// ── Stats ─────────────────────────────────────────────────────────
const stats = computed(() => {
  const all = members.value as any[]
  const adminCount  = all.filter(m => m.role === 'admin').length
  const activeCount = all.filter(m => m.status === 'active').length
  const pendingInvs = (invitations.value as any[]).filter(i => i.status === 'pending').length
  return {
    total: memberTotal.value || all.length,
    admins: adminCount,
    active: activeCount,
    pendingInvitations: pendingInvs,
    recentNames: all.slice(0, 4).map(m => m.name ?? '?'),
  }
})

// ── Search ────────────────────────────────────────────────────────
const searchValue = ref('')

const filteredMembers = computed(() => {
  const q = searchValue.value.toLowerCase().trim()
  if (!q) return members.value
  return (members.value as any[]).filter(m =>
    m.name?.toLowerCase().includes(q) || m.email?.toLowerCase().includes(q)
  )
})

// Reset to first page whenever the search filter changes.
watch(searchValue, () => memberTable.setPageIndex(0))

// ── Remove member ─────────────────────────────────────────────────
const removeTarget = ref<any>(null)
const removeDialogOpen = ref(false)

function promptRemove(m: any) { removeTarget.value = m; removeDialogOpen.value = true }
async function confirmRemove() {
  if (removeTarget.value) {
    try {
      await removeMember(removeTarget.value.userId)
      toast.success('成员已移除')
      refreshMembers()
    } catch { toast.error('移除失败') }
  }
  removeTarget.value = null
}

// ── Edit role ─────────────────────────────────────────────────────
const editTarget    = ref<any>(null)
const editRole      = ref('')
const editDialogOpen = ref(false)
const editLoading   = ref(false)

function openEditRole(m: any) {
  editTarget.value = m
  editRole.value = m.role
  editDialogOpen.value = true
}
async function confirmEditRole() {
  if (!editTarget.value) return
  editLoading.value = true
  try {
    await updateMemberRole(editTarget.value.userId, editRole.value)
    toast.success('角色已更新')
    editDialogOpen.value = false
    refreshMembers()
  } catch { toast.error('更新失败') }
  editLoading.value = false
}

// ── Invite member ─────────────────────────────────────────────────
const inviteDialogOpen = ref(false)
const inviteForm = ref({ email: '', role: 'member' as string })
const inviteLoading = ref(false)

function openInvite() {
  inviteForm.value = { email: '', role: 'member' }
  inviteDialogOpen.value = true
}
async function submitInvite() {
  if (!inviteForm.value.email) { toast.error('请填写邮箱'); return }
  inviteLoading.value = true
  try {
    await createInvitation(inviteForm.value)
    toast.success('邀请已发送')
    inviteDialogOpen.value = false
    refreshInvitations()
  } catch (e: any) {
    toast.error(e?.response?.data?.message ?? '邀请失败')
  }
  inviteLoading.value = false
}

// ── Revoke invitation ─────────────────────────────────────────────
const revokeTarget = ref<any>(null)
const revokeDialogOpen = ref(false)

function promptRevoke(inv: any) { revokeTarget.value = inv; revokeDialogOpen.value = true }
async function confirmRevoke() {
  if (revokeTarget.value) {
    try {
      await revokeInvitation(revokeTarget.value.id)
      toast.success('邀请已撤销')
      refreshInvitations()
    } catch { toast.error('撤销失败') }
  }
  revokeTarget.value = null
}

// ── Member columns ────────────────────────────────────────────────
type MemberRow = (typeof members.value)[number]

const memberColumns: ColumnDef<MemberRow>[] = [
  {
    id: 'member',
    header: '成员',
    cell: ({ row }) => {
      const m = row.original as any
      return h('div', { class: 'flex items-center gap-3' }, [
        m.avatar
          ? h('img', { src: m.avatar, class: 'size-9 rounded-xl object-cover shrink-0' })
          : h('div', {
              class: `size-9 rounded-xl flex items-center justify-center text-white text-xs font-black shrink-0 ${avatarColor(m.name)}`,
            }, firstChar(m.name)),
        h('div', { class: 'min-w-0' }, [
          h('p', { class: 'font-semibold text-sm leading-none' }, m.name || '—'),
          h('p', { class: 'font-mono text-[11px] text-muted-foreground/60 mt-1 truncate max-w-48' }, m.email),
        ]),
      ])
    },
  },
  {
    accessorKey: 'role',
    header: '角色',
    cell: ({ row }) => {
      const role: string = (row.original as any).role ?? 'viewer'
      const icon = roleIcon[role] ?? User
      return h('span', {
        class: `text-[11px] font-bold px-2.5 py-1 rounded-full flex items-center gap-1.5 w-fit ${roleStyle[role] ?? roleStyle.viewer}`,
      }, [h(icon, { class: 'size-3' }), roleLabel[role] ?? role])
    },
  },
  {
    accessorKey: 'provider',
    header: '来源',
    cell: ({ row }) => {
      const provider: string = (row.original as any).provider ?? 'local'
      return h('span', {
        class: `text-[10px] font-bold px-2 py-0.5 rounded uppercase tracking-wider ${providerStyle[provider] ?? providerStyle.local}`,
      }, provider)
    },
  },
  {
    accessorKey: 'joinedAt',
    header: '加入时间',
    cell: ({ row }) => {
      const t = (row.original as any).joinedAt
      return h('div', { class: 'flex items-center gap-1.5 text-xs text-muted-foreground' }, [
        h(Clock, { class: 'size-3 shrink-0' }),
        formatDate(t),
      ])
    },
  },
  {
    id: 'actions',
    header: '',
    cell: ({ row }) => {
      const m = row.original as any
      return h(DropdownMenu, {}, {
        default: () => [
          h(DropdownMenuTrigger, { asChild: true }, () =>
            h(Button, { variant: 'ghost', size: 'sm', class: 'size-8 p-0' }, () =>
              h(MoreHorizontal, { class: 'size-4' })
            )
          ),
          h(DropdownMenuContent, { align: 'end', class: 'w-36' }, () => [
            h(DropdownMenuItem, { onClick: () => openEditRole(m) }, () => [
              h(Pencil, { class: 'mr-2 size-3.5' }), '修改角色',
            ]),
            h(DropdownMenuSeparator),
            h(DropdownMenuItem, {
              class: 'text-destructive focus:text-destructive',
              onClick: () => promptRemove(m),
            }, () => [h(Trash2, { class: 'mr-2 size-3.5' }), '移除成员']),
          ]),
        ],
      })
    },
  },
]

// ── Invitation columns ────────────────────────────────────────────
type InvRow = (typeof invitations.value)[number]

const invColumns: ColumnDef<InvRow>[] = [
  {
    id: 'email',
    header: '邮箱',
    cell: ({ row }) => {
      const inv = row.original as any
      return h('div', { class: 'flex items-center gap-2' }, [
        h('div', { class: 'size-8 rounded-lg bg-muted flex items-center justify-center shrink-0' },
          h(Mail, { class: 'size-3.5 text-muted-foreground' })
        ),
        h('span', { class: 'font-mono text-sm' }, inv.email),
      ])
    },
  },
  {
    accessorKey: 'role',
    header: '角色',
    cell: ({ row }) => {
      const role: string = (row.original as any).role ?? 'member'
      const icon = roleIcon[role] ?? User
      return h('span', {
        class: `text-[11px] font-bold px-2.5 py-1 rounded-full flex items-center gap-1.5 w-fit ${roleStyle[role] ?? roleStyle.member}`,
      }, [h(icon, { class: 'size-3' }), roleLabel[role] ?? role])
    },
  },
  {
    accessorKey: 'status',
    header: '状态',
    cell: ({ row }) => {
      const status: string = (row.original as any).status ?? 'pending'
      return h('span', {
        class: `text-[11px] font-bold px-2.5 py-1 rounded-full w-fit ${invStatusStyle[status] ?? invStatusStyle.pending}`,
      }, invStatusLabel[status] ?? status)
    },
  },
  {
    accessorKey: 'expiresAt',
    header: '过期时间',
    cell: ({ row }) => {
      const inv = row.original as any
      const expired = inv.status === 'expired' || new Date(inv.expiresAt) < new Date()
      return h('div', { class: `flex items-center gap-1.5 text-xs ${expired ? 'text-red-500' : 'text-muted-foreground'}` }, [
        h(Clock, { class: 'size-3 shrink-0' }),
        formatDate(inv.expiresAt),
      ])
    },
  },
  {
    id: 'actions',
    header: '',
    cell: ({ row }) => {
      const inv = row.original as any
      if (inv.status !== 'pending') return h('span')
      return h(Button, {
        variant: 'ghost',
        size: 'sm',
        class: 'text-destructive hover:text-destructive h-8 px-2 text-xs gap-1',
        onClick: () => promptRevoke(inv),
      }, () => [h(XCircle, { class: 'size-3.5' }), '撤销'])
    },
  },
]

// ── TanStack Tables ───────────────────────────────────────────────
const memberTable = useVueTable({
  get data() { return filteredMembers.value as any[] },
  columns: memberColumns,
  getCoreRowModel: getCoreRowModel(),
  getPaginationRowModel: getPaginationRowModel(),
  initialState: { pagination: { pageSize: 10 } },
})

const invTable = useVueTable({
  get data() { return invitations.value as any[] },
  columns: invColumns,
  getCoreRowModel: getCoreRowModel(),
  getPaginationRowModel: getPaginationRowModel(),
  initialState: { pagination: { pageSize: 10 } },
})
</script>

<template>
  <div class="flex flex-col gap-5 p-6 animate-in fade-in duration-300">

    <!-- ── Stat cards ──────────────────────────────────────────────── -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">

      <!-- 全部成员 -->
      <div class="border-border bg-card text-card-foreground rounded-xl border p-5 shadow-sm">
        <div class="flex items-start justify-between">
          <div class="flex flex-col gap-1">
            <span class="text-muted-foreground text-sm font-medium">全部成员</span>
            <span class="text-2xl font-bold tracking-tight">{{ stats.total }}</span>
          </div>
          <div class="bg-muted rounded-lg p-2">
            <Users class="text-muted-foreground size-4" />
          </div>
        </div>
        <div class="mt-3 flex items-center gap-1 text-sm">
          <div class="flex -space-x-1.5">
            <div
              v-for="(name, i) in stats.recentNames" :key="i"
              class="size-5 rounded-full ring-2 ring-card flex items-center justify-center text-[8px] font-black text-white shrink-0"
              :class="avatarColor(name)"
            >{{ firstChar(name) }}</div>
          </div>
          <span v-if="stats.total > 4" class="text-muted-foreground text-xs ml-1">
            +{{ stats.total - 4 }} 人
          </span>
        </div>
      </div>

      <!-- 管理员 -->
      <div class="border-border bg-card text-card-foreground rounded-xl border p-5 shadow-sm">
        <div class="flex items-start justify-between">
          <div class="flex flex-col gap-1">
            <span class="text-muted-foreground text-sm font-medium">管理员</span>
            <span class="text-2xl font-bold tracking-tight">{{ stats.admins }}</span>
          </div>
          <div class="bg-muted rounded-lg p-2">
            <Shield class="text-muted-foreground size-4" />
          </div>
        </div>
        <div class="mt-3 flex items-center gap-1 text-sm">
          <ArrowUpRight class="text-muted-foreground size-4 shrink-0" />
          <span class="text-muted-foreground">共 <span class="font-semibold text-foreground">{{ stats.admins }}</span> 人拥有管理权限</span>
        </div>
      </div>

      <!-- 活跃 -->
      <div class="border-border bg-card text-card-foreground rounded-xl border p-5 shadow-sm">
        <div class="flex items-start justify-between">
          <div class="flex flex-col gap-1">
            <span class="text-muted-foreground text-sm font-medium">活跃成员</span>
            <span class="text-2xl font-bold tracking-tight">{{ stats.active }}</span>
          </div>
          <div class="bg-muted rounded-lg p-2">
            <UserCheck class="text-muted-foreground size-4" />
          </div>
        </div>
        <div class="mt-3 flex items-center gap-1 text-sm">
          <ArrowUpRight class="text-emerald-600 size-4 shrink-0" />
          <span class="text-emerald-600 font-semibold">
            {{ stats.total ? Math.round((stats.active / stats.total) * 100) : 0 }}%
          </span>
          <span class="text-muted-foreground">活跃率</span>
        </div>
      </div>

      <!-- 待接受邀请 -->
      <button
        class="border-border bg-card text-card-foreground rounded-xl border p-5 shadow-sm text-left hover:shadow-md transition-shadow"
        :class="activeTab === 'invitations' ? 'ring-2 ring-amber-400/20 border-amber-400/30' : ''"
        @click="activeTab = 'invitations'"
      >
        <div class="flex items-start justify-between">
          <div class="flex flex-col gap-1">
            <span class="text-muted-foreground text-sm font-medium">待接受邀请</span>
            <span class="text-2xl font-bold tracking-tight">{{ stats.pendingInvitations }}</span>
          </div>
          <div class="bg-muted rounded-lg p-2">
            <Clock class="text-muted-foreground size-4" />
          </div>
        </div>
        <div class="mt-3 flex items-center gap-1 text-sm">
          <component
            :is="stats.pendingInvitations === 0 ? ArrowUpRight : ArrowDownRight"
            :class="stats.pendingInvitations === 0 ? 'text-emerald-600' : 'text-amber-500'"
            class="size-4 shrink-0"
          />
          <span :class="stats.pendingInvitations === 0 ? 'text-emerald-600 font-semibold' : 'text-amber-500 font-semibold'">
            {{ stats.pendingInvitations === 0 ? '全部已接受' : stats.pendingInvitations + ' 条待处理' }}
          </span>
          <span class="text-muted-foreground">{{ stats.pendingInvitations === 0 ? '无需处理' : '等待接受' }}</span>
        </div>
      </button>

    </div>

    <!-- ── Tabs + Toolbar ──────────────────────────────────────────── -->
    <div class="flex items-center gap-2">
      <!-- Tabs -->
      <div class="flex bg-muted/50 rounded-lg p-1 border border-border gap-1">
        <button
          class="px-4 py-1.5 rounded-md text-xs font-semibold transition-all"
          :class="activeTab === 'members'
            ? 'bg-background text-foreground shadow-sm ring-1 ring-border'
            : 'text-muted-foreground hover:text-foreground'"
          @click="activeTab = 'members'"
        >
          成员
          <span class="ml-1.5 tabular-nums text-[10px] opacity-60">{{ stats.total }}</span>
        </button>
        <button
          class="px-4 py-1.5 rounded-md text-xs font-semibold transition-all"
          :class="activeTab === 'invitations'
            ? 'bg-background text-foreground shadow-sm ring-1 ring-border'
            : 'text-muted-foreground hover:text-foreground'"
          @click="activeTab = 'invitations'"
        >
          邀请
          <span
            class="ml-1.5 tabular-nums text-[10px]"
            :class="stats.pendingInvitations > 0 ? 'text-amber-500 font-bold' : 'opacity-60'"
          >{{ invTotal }}</span>
        </button>
      </div>

      <!-- Toolbar right -->
      <div class="ml-auto flex items-center gap-2">
        <div v-if="activeTab === 'members'" class="relative w-72">
          <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
          <Input
            v-model="searchValue"
            placeholder="搜索名称或邮箱..."
            class="pl-8 h-9"
            @input="onSearchInput"
          />
        </div>
        <Button
          variant="outline" size="sm" class="gap-1.5"
          :disabled="activeTab === 'members' ? memberLoading : invLoading"
          @click="activeTab === 'members' ? refreshMembers() : refreshInvitations()"
        >
          <RefreshCw
            class="size-3.5"
            :class="(activeTab === 'members' ? memberLoading : invLoading) ? 'animate-spin' : ''"
          />
          刷新
        </Button>
        <Button size="sm" class="gap-1.5" @click="openInvite">
          <Plus class="size-3.5" /> 邀请成员
        </Button>
      </div>
    </div>

    <!-- ── Members Table ───────────────────────────────────────────── -->
    <div v-if="activeTab === 'members'" class="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow v-for="hg in memberTable.getHeaderGroups()" :key="hg.id">
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
          <template v-if="memberTable.getRowModel().rows.length">
            <TableRow v-for="row in memberTable.getRowModel().rows" :key="row.id">
              <TableCell v-for="cell in row.getVisibleCells()" :key="cell.id">
                <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
              </TableCell>
            </TableRow>
          </template>
          <TableRow v-else>
            <TableCell :colspan="memberColumns.length" class="h-32 text-center text-muted-foreground">
              {{ memberLoading ? '加载中...' : '暂无成员' }}
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
      <DataTablePagination :table="memberTable" />
    </div>

    <!-- ── Invitations Table ───────────────────────────────────────── -->
    <div v-if="activeTab === 'invitations'" class="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow v-for="hg in invTable.getHeaderGroups()" :key="hg.id">
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
          <template v-if="invTable.getRowModel().rows.length">
            <TableRow v-for="row in invTable.getRowModel().rows" :key="row.id">
              <TableCell v-for="cell in row.getVisibleCells()" :key="cell.id">
                <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
              </TableCell>
            </TableRow>
          </template>
          <TableRow v-else>
            <TableCell :colspan="invColumns.length" class="h-32 text-center text-muted-foreground">
              {{ invLoading ? '加载中...' : '暂无邀请记录' }}
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
      <DataTablePagination :table="invTable" />
    </div>

    <!-- ── Remove confirm ─────────────────────────────────────────── -->
    <AppAlertDialog
      v-model:open="removeDialogOpen"
      title="移除成员"
      :description="`确认将「${removeTarget?.name}」从 Workspace 中移除？`"
      confirm-text="确认移除"
      variant="destructive"
      @confirm="confirmRemove"
      @cancel="removeTarget = null"
    />

    <!-- ── Revoke confirm ──────────────────────────────────────────── -->
    <AppAlertDialog
      v-model:open="revokeDialogOpen"
      title="撤销邀请"
      :description="`确认撤销发送给「${revokeTarget?.email}」的邀请？`"
      confirm-text="确认撤销"
      variant="destructive"
      @confirm="confirmRevoke"
      @cancel="revokeTarget = null"
    />

  </div>

  <!-- ── Invite Dialog ───────────────────────────────────────────── -->
  <Dialog v-model:open="inviteDialogOpen">
    <DialogContent class="sm:max-w-sm">
      <DialogHeader>
        <DialogTitle>邀请成员</DialogTitle>
        <DialogDescription>通过邮件邀请新成员加入当前 Workspace，链接 7 天内有效。</DialogDescription>
      </DialogHeader>

      <div class="space-y-4 py-2">
        <div class="space-y-1.5">
          <label class="text-xs font-medium">邮箱地址</label>
          <Input
            v-model="inviteForm.email"
            type="email"
            placeholder="ops@example.com"
            @keyup.enter="submitInvite"
          />
        </div>

        <div class="space-y-1.5">
          <label class="text-xs font-medium">角色</label>
          <select
            v-model="inviteForm.role"
            class="w-full h-9 rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 transition-[color,box-shadow]"
          >
            <option value="admin">管理员 — 管理成员与空间配置</option>
            <option value="editor">编辑者 — 管理节点与策略</option>
            <option value="member">成员 — 查看与使用资源</option>
            <option value="viewer">访客 — 只读访问</option>
          </select>
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="inviteDialogOpen = false">取消</Button>
        <Button :disabled="inviteLoading" @click="submitInvite">
          {{ inviteLoading ? '发送中...' : '发送邀请' }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>

  <!-- ── Edit Role Dialog ────────────────────────────────────────── -->
  <Dialog v-model:open="editDialogOpen">
    <DialogContent class="sm:max-w-sm">
      <DialogHeader>
        <DialogTitle>修改角色</DialogTitle>
        <DialogDescription>
          修改「{{ editTarget?.name }}」在当前 Workspace 的角色。
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-1.5 py-2">
        <label class="text-xs font-medium">新角色</label>
        <select
          v-model="editRole"
          class="w-full h-9 rounded-md border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 transition-[color,box-shadow]"
        >
          <option value="admin">管理员</option>
          <option value="editor">编辑者</option>
          <option value="member">成员</option>
          <option value="viewer">访客</option>
        </select>
      </div>

      <DialogFooter>
        <Button variant="outline" @click="editDialogOpen = false">取消</Button>
        <Button :disabled="editLoading" @click="confirmEditRole">
          {{ editLoading ? '保存中...' : '保存' }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
