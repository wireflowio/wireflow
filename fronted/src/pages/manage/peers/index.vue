<script setup lang="ts">
import { ref, computed } from 'vue'
import { Plus, Search, UserCheck, Shield, Users, ChevronDown, Trash2 } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription,
} from '@/components/ui/sheet'
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

definePage({
  meta: { title: '对等连接', description: '管理网络成员与访问权限。' },
})

type Role = 'admin' | 'member' | 'viewer'
type AuthType = 'local' | 'oidc'

interface Peer {
  id: string
  name: string
  email: string
  role: Role
  authType: AuthType
  workspaces: string[]
  joinedAt: string
  status: 'active' | 'pending'
}

const peers = ref<Peer[]>([
  { id: '1', name: 'Alice Chen', email: 'alice@example.com', role: 'admin', authType: 'local', workspaces: ['production', 'dev-test'], joinedAt: '2024-01-10', status: 'active' },
  { id: '2', name: 'Bob Wang', email: 'bob@example.com', role: 'member', authType: 'oidc', workspaces: ['dev-test'], joinedAt: '2024-02-15', status: 'active' },
  { id: '3', name: 'Carol Liu', email: 'carol@example.com', role: 'member', authType: 'local', workspaces: ['production', 'dev-test', 'demo'], joinedAt: '2024-03-20', status: 'active' },
  { id: '4', name: 'Dave Zhang', email: 'dave@example.com', role: 'viewer', authType: 'oidc', workspaces: ['demo'], joinedAt: '2024-04-05', status: 'pending' },
  { id: '5', name: 'Eve Lin', email: 'eve@example.com', role: 'member', authType: 'local', workspaces: ['ci-pipeline', 'dev-test'], joinedAt: '2024-05-12', status: 'active' },
])

const search = ref('')
const inviteDrawerOpen = ref(false)
const inviteTab = ref<'local' | 'oidc'>('local')

const inviteForm = ref({
  email: '',
  role: 'member' as Role,
  workspaces: [] as string[],
  oidcProvider: '',
})

const allWorkspaces = ['production', 'dev-test', 'demo', 'ci-pipeline', 'archive']

const filtered = computed(() => peers.value.filter(p =>
  !search.value || p.name.toLowerCase().includes(search.value.toLowerCase()) || p.email.includes(search.value)
))

const roleBadge: Record<Role, string> = {
  admin: 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400',
  member: 'bg-primary/10 text-primary',
  viewer: 'bg-muted text-muted-foreground',
}

const roleIcon: Record<Role, typeof Shield> = {
  admin: Shield,
  member: Users,
  viewer: UserCheck,
}

function toggleWorkspace(ws: string) {
  const idx = inviteForm.value.workspaces.indexOf(ws)
  if (idx > -1) inviteForm.value.workspaces.splice(idx, 1)
  else inviteForm.value.workspaces.push(ws)
}

function sendInvite() {
  peers.value.push({
    id: String(Date.now()),
    name: inviteForm.value.email.split('@')[0],
    email: inviteForm.value.email,
    role: inviteForm.value.role,
    authType: inviteTab.value,
    workspaces: [...inviteForm.value.workspaces],
    joinedAt: new Date().toISOString().slice(0, 10),
    status: 'pending',
  })
  inviteDrawerOpen.value = false
  inviteForm.value = { email: '', role: 'member', workspaces: [], oidcProvider: '' }
}

function removePeer(id: string) {
  peers.value = peers.value.filter(p => p.id !== id)
}

function changeRole(peer: Peer, role: Role) {
  peer.role = role
}
</script>

<template>
  <div class="p-6 space-y-5">
    <!-- Toolbar -->
    <div class="flex flex-wrap items-center gap-3">
      <div class="relative flex-1 min-w-48">
        <Search class="absolute left-2.5 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
        <Input v-model="search" placeholder="搜索成员..." class="pl-8" />
      </div>
      <Button size="sm" class="gap-1.5" @click="inviteDrawerOpen = true">
        <Plus class="size-3.5" /> 邀请成员
      </Button>
    </div>

    <!-- Stats -->
    <div class="grid grid-cols-3 gap-3">
      <div class="bg-card border border-border rounded-xl p-4 text-center">
        <p class="text-2xl font-bold">{{ peers.filter(p => p.role === 'admin').length }}</p>
        <p class="text-xs text-muted-foreground mt-0.5">管理员</p>
      </div>
      <div class="bg-card border border-border rounded-xl p-4 text-center">
        <p class="text-2xl font-bold">{{ peers.filter(p => p.role === 'member').length }}</p>
        <p class="text-xs text-muted-foreground mt-0.5">成员</p>
      </div>
      <div class="bg-card border border-border rounded-xl p-4 text-center">
        <p class="text-2xl font-bold">{{ peers.filter(p => p.status === 'pending').length }}</p>
        <p class="text-xs text-muted-foreground mt-0.5">待接受</p>
      </div>
    </div>

    <!-- Table -->
    <div class="bg-card border border-border rounded-xl overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-border bg-muted/30">
            <th class="text-left px-4 py-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">成员</th>
            <th class="text-left px-4 py-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">角色</th>
            <th class="text-left px-4 py-3 text-xs font-medium uppercase tracking-wider text-muted-foreground hidden md:table-cell">认证方式</th>
            <th class="text-left px-4 py-3 text-xs font-medium uppercase tracking-wider text-muted-foreground hidden lg:table-cell">工作空间</th>
            <th class="text-left px-4 py-3 text-xs font-medium uppercase tracking-wider text-muted-foreground hidden sm:table-cell">加入时间</th>
            <th class="text-left px-4 py-3 text-xs font-medium uppercase tracking-wider text-muted-foreground">状态</th>
            <th class="px-4 py-3" />
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="peer in filtered"
            :key="peer.id"
            class="border-b border-border last:border-0 hover:bg-muted/20 transition-colors"
          >
            <td class="px-4 py-3">
              <div class="flex items-center gap-3">
                <div class="size-8 rounded-full bg-primary/10 flex items-center justify-center text-xs font-semibold text-primary">
                  {{ peer.name.split(' ').map(n => n[0]).join('') }}
                </div>
                <div>
                  <p class="font-medium">{{ peer.name }}</p>
                  <p class="text-xs text-muted-foreground">{{ peer.email }}</p>
                </div>
              </div>
            </td>
            <td class="px-4 py-3">
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <button class="flex items-center gap-1 text-xs rounded-full px-2.5 py-1 font-medium transition-colors" :class="roleBadge[peer.role]">
                    <component :is="roleIcon[peer.role]" class="size-3" />
                    {{ peer.role === 'admin' ? '管理员' : peer.role === 'member' ? '成员' : '访客' }}
                    <ChevronDown class="size-3 ml-0.5" />
                  </button>
                </DropdownMenuTrigger>
                <DropdownMenuContent>
                  <DropdownMenuItem @click="changeRole(peer, 'admin')">管理员</DropdownMenuItem>
                  <DropdownMenuItem @click="changeRole(peer, 'member')">成员</DropdownMenuItem>
                  <DropdownMenuItem @click="changeRole(peer, 'viewer')">访客</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </td>
            <td class="px-4 py-3 hidden md:table-cell">
              <span class="text-xs rounded px-2 py-0.5 bg-muted text-muted-foreground uppercase font-medium tracking-wider">
                {{ peer.authType }}
              </span>
            </td>
            <td class="px-4 py-3 hidden lg:table-cell">
              <div class="flex flex-wrap gap-1">
                <span v-for="ws in peer.workspaces" :key="ws" class="text-xs bg-muted rounded px-1.5 py-0.5 text-muted-foreground">
                  {{ ws }}
                </span>
              </div>
            </td>
            <td class="px-4 py-3 text-muted-foreground text-xs hidden sm:table-cell">{{ peer.joinedAt }}</td>
            <td class="px-4 py-3">
              <span
                class="text-xs rounded-full px-2 py-0.5 font-medium"
                :class="peer.status === 'active'
                  ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                  : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'"
              >
                {{ peer.status === 'active' ? '已接受' : '待接受' }}
              </span>
            </td>
            <td class="px-4 py-3">
              <Button variant="ghost" size="sm" class="size-8 p-0 text-muted-foreground hover:text-destructive" @click="removePeer(peer.id)">
                <Trash2 class="size-4" />
              </Button>
            </td>
          </tr>
          <tr v-if="filtered.length === 0">
            <td colspan="7" class="px-4 py-12 text-center text-muted-foreground">未找到成员</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Invite Drawer -->
    <Sheet v-model:open="inviteDrawerOpen">
      <SheetContent class="w-[420px] overflow-y-auto">
        <SheetHeader>
          <SheetTitle>邀请成员</SheetTitle>
          <SheetDescription>邀请新成员加入网络</SheetDescription>
        </SheetHeader>

        <div class="mt-6 space-y-5">
          <!-- Auth type tabs -->
          <div class="flex gap-1 bg-muted rounded-lg p-1">
            <button
              v-for="tab in (['local', 'oidc'] as const)"
              :key="tab"
              @click="inviteTab = tab"
              class="flex-1 py-1.5 rounded-md text-sm font-medium transition-colors"
              :class="inviteTab === tab ? 'bg-background shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground'"
            >
              {{ tab === 'local' ? '本地账号' : 'OIDC 账号' }}
            </button>
          </div>

          <div class="space-y-1.5">
            <label class="text-sm font-medium">邮箱地址</label>
            <Input v-model="inviteForm.email" type="email" placeholder="user@example.com" />
          </div>

          <div v-if="inviteTab === 'oidc'" class="space-y-1.5">
            <label class="text-sm font-medium">OIDC 提供商</label>
            <Input v-model="inviteForm.oidcProvider" placeholder="例如：https://accounts.google.com" />
          </div>

          <div class="space-y-1.5">
            <label class="text-sm font-medium">角色</label>
            <div class="flex gap-2">
              <button
                v-for="r in (['admin', 'member', 'viewer'] as const)"
                :key="r"
                @click="inviteForm.role = r"
                class="flex-1 py-2 rounded-lg border text-sm font-medium transition-colors"
                :class="inviteForm.role === r
                  ? 'border-primary bg-primary/5 text-primary'
                  : 'border-border text-muted-foreground hover:text-foreground hover:border-foreground/20'"
              >
                {{ r === 'admin' ? '管理员' : r === 'member' ? '成员' : '访客' }}
              </button>
            </div>
          </div>

          <div class="space-y-2">
            <label class="text-sm font-medium">授权工作空间</label>
            <div class="space-y-2">
              <label
                v-for="ws in allWorkspaces"
                :key="ws"
                class="flex items-center gap-3 p-2.5 rounded-lg border border-border cursor-pointer hover:bg-muted/30 transition-colors"
                :class="inviteForm.workspaces.includes(ws) ? 'border-primary bg-primary/5' : ''"
              >
                <input type="checkbox" :checked="inviteForm.workspaces.includes(ws)" @change="toggleWorkspace(ws)" class="accent-primary" />
                <span class="text-sm font-mono">{{ ws }}</span>
              </label>
            </div>
          </div>

          <div class="flex justify-end gap-2 pt-2">
            <Button variant="outline" @click="inviteDrawerOpen = false">取消</Button>
            <Button @click="sendInvite">发送邀请</Button>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  </div>
</template>
