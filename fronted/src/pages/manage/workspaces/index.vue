<script setup lang="ts">
import { ref, computed } from 'vue'
import { Plus, Server, ArrowRight, MoreHorizontal, Pencil, Trash2 } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
} from '@/components/ui/dialog'
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

definePage({
  meta: { title: '空间管理', description: '管理网络隔离工作空间。' },
})

interface Workspace {
  id: string
  displayName: string
  slug: string
  nodeCount: number
  maxNodeCount: number
  status: 'active' | 'inactive'
  createdAt: string
}

const workspaces = ref<Workspace[]>([
  { id: '1', displayName: '生产环境', slug: 'production', nodeCount: 18, maxNodeCount: 50, status: 'active', createdAt: '2024-01-10' },
  { id: '2', displayName: '开发测试', slug: 'dev-test', nodeCount: 6, maxNodeCount: 20, status: 'active', createdAt: '2024-02-14' },
  { id: '3', displayName: '演示环境', slug: 'demo', nodeCount: 3, maxNodeCount: 10, status: 'active', createdAt: '2024-03-05' },
  { id: '4', displayName: '归档空间', slug: 'archive', nodeCount: 0, maxNodeCount: 10, status: 'inactive', createdAt: '2023-11-20' },
  { id: '5', displayName: 'CI 流水线', slug: 'ci-pipeline', nodeCount: 12, maxNodeCount: 30, status: 'active', createdAt: '2024-04-18' },
])

const drawerOpen = ref(false)
const editingWorkspace = ref<Workspace | null>(null)
const form = ref({ displayName: '', slug: '', maxNodeCount: 20 })

function openCreate() {
  editingWorkspace.value = null
  form.value = { displayName: '', slug: '', maxNodeCount: 20 }
  drawerOpen.value = true
}

function openEdit(ws: Workspace) {
  editingWorkspace.value = ws
  form.value = { displayName: ws.displayName, slug: ws.slug, maxNodeCount: ws.maxNodeCount }
  drawerOpen.value = true
}

function saveWorkspace() {
  if (editingWorkspace.value) {
    const ws = workspaces.value.find(w => w.id === editingWorkspace.value!.id)
    if (ws) {
      ws.displayName = form.value.displayName
      ws.slug = form.value.slug
      ws.maxNodeCount = form.value.maxNodeCount
    }
  } else {
    workspaces.value.push({
      id: String(Date.now()),
      displayName: form.value.displayName,
      slug: form.value.slug,
      nodeCount: 0,
      maxNodeCount: form.value.maxNodeCount,
      status: 'active',
      createdAt: new Date().toISOString().slice(0, 10),
    })
  }
  drawerOpen.value = false
}

function deleteWorkspace(id: string) {
  workspaces.value = workspaces.value.filter(w => w.id !== id)
}

function slugify(v: string) {
  form.value.slug = v.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '')
}

const usagePct = (ws: Workspace) => Math.round((ws.nodeCount / ws.maxNodeCount) * 100)
</script>

<template>
  <div class="p-6 space-y-5">
    <!-- Toolbar -->
    <div class="flex items-center justify-between">
      <p class="text-sm text-muted-foreground">共 {{ workspaces.length }} 个工作空间</p>
      <Button size="sm" class="gap-1.5" @click="openCreate">
        <Plus class="size-3.5" /> 创建空间
      </Button>
    </div>

    <!-- Workspace grid -->
    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <div
        v-for="ws in workspaces"
        :key="ws.id"
        class="bg-card border border-border rounded-xl p-5 flex flex-col gap-4 hover:shadow-md transition-shadow"
      >
        <!-- Header -->
        <div class="flex items-start justify-between gap-2">
          <div class="flex items-center gap-3">
            <div class="size-10 rounded-lg bg-primary/10 flex items-center justify-center">
              <Server class="size-5 text-primary" />
            </div>
            <div>
              <p class="font-semibold text-sm">{{ ws.displayName }}</p>
              <p class="text-xs text-muted-foreground font-mono">{{ ws.slug }}</p>
            </div>
          </div>
          <div class="flex items-center gap-1">
            <span
              class="text-xs rounded-full px-2 py-0.5 font-medium"
              :class="ws.status === 'active'
                ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                : 'bg-muted text-muted-foreground'"
            >
              {{ ws.status === 'active' ? '运行中' : '已停用' }}
            </span>
            <DropdownMenu>
              <DropdownMenuTrigger as-child>
                <Button variant="ghost" size="sm" class="size-7 p-0">
                  <MoreHorizontal class="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem @click="openEdit(ws)">
                  <Pencil class="mr-2 size-3.5" /> 编辑
                </DropdownMenuItem>
                <DropdownMenuItem class="text-destructive focus:text-destructive" @click="deleteWorkspace(ws.id)">
                  <Trash2 class="mr-2 size-3.5" /> 删除
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>

        <!-- Node usage -->
        <div class="space-y-1.5">
          <div class="flex justify-between text-xs">
            <span class="text-muted-foreground">节点数量</span>
            <span class="font-medium">{{ ws.nodeCount }} / {{ ws.maxNodeCount }}</span>
          </div>
          <div class="h-1.5 bg-muted rounded-full overflow-hidden">
            <div
              class="h-full rounded-full transition-all"
              :class="usagePct(ws) > 80 ? 'bg-rose-500' : usagePct(ws) > 60 ? 'bg-amber-500' : 'bg-primary'"
              :style="{ width: usagePct(ws) + '%' }"
            />
          </div>
          <p class="text-xs text-muted-foreground">已使用 {{ usagePct(ws) }}%</p>
        </div>

        <!-- Footer -->
        <div class="flex items-center justify-between pt-1">
          <p class="text-xs text-muted-foreground">创建于 {{ ws.createdAt }}</p>
          <Button variant="outline" size="sm" class="gap-1 h-7 text-xs">
            进入空间 <ArrowRight class="size-3" />
          </Button>
        </div>
      </div>
    </div>

    <!-- Create / Edit Dialog -->
    <Dialog v-model:open="drawerOpen">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ editingWorkspace ? '编辑空间' : '创建空间' }}</DialogTitle>
          <DialogDescription>{{ editingWorkspace ? '修改工作空间配置' : '新建一个隔离的网络工作空间' }}</DialogDescription>
        </DialogHeader>
        <div class="space-y-4 py-2">
          <div class="space-y-1.5">
            <label class="text-sm font-medium">显示名称</label>
            <Input
              v-model="form.displayName"
              placeholder="例如：生产环境"
              @input="!editingWorkspace && slugify(form.displayName)"
            />
          </div>
          <div class="space-y-1.5">
            <label class="text-sm font-medium">Slug <span class="text-muted-foreground font-normal text-xs">(唯一标识符)</span></label>
            <Input v-model="form.slug" placeholder="例如：production" class="font-mono" />
          </div>
          <div class="space-y-1.5">
            <label class="text-sm font-medium">最大节点数</label>
            <Input v-model.number="form.maxNodeCount" type="number" min="1" max="1000" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" @click="drawerOpen = false">取消</Button>
          <Button @click="saveWorkspace">{{ editingWorkspace ? '保存' : '创建' }}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
