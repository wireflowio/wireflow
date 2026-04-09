<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { ZoomIn, ZoomOut, Maximize2, X } from 'lucide-vue-next'

definePage({
  meta: { title: '网络拓扑', description: '可视化网络节点连接拓扑图。' },
})

interface TopoNode {
  id: string
  name: string
  x: number
  y: number
  status: 'online' | 'offline' | 'relay'
}

interface TopoLink {
  source: string
  target: string
  quality: number // 0-100
  type: 'p2p' | 'relay'
}

const nodes = ref<TopoNode[]>([
  { id: 'alpha', name: 'node-alpha', x: 400, y: 200, status: 'online' },
  { id: 'beta', name: 'node-beta', x: 600, y: 350, status: 'online' },
  { id: 'gamma', name: 'node-gamma', x: 200, y: 350, status: 'offline' },
  { id: 'delta', name: 'node-delta', x: 500, y: 500, status: 'online' },
  { id: 'epsilon', name: 'node-epsilon', x: 300, y: 500, status: 'relay' },
  { id: 'relay1', name: 'relay-01', x: 400, y: 380, status: 'relay' },
])

const links = ref<TopoLink[]>([
  { source: 'alpha', target: 'beta', quality: 95, type: 'p2p' },
  { source: 'alpha', target: 'gamma', quality: 0, type: 'relay' },
  { source: 'alpha', target: 'relay1', quality: 88, type: 'p2p' },
  { source: 'beta', target: 'delta', quality: 72, type: 'p2p' },
  { source: 'relay1', target: 'epsilon', quality: 60, type: 'relay' },
  { source: 'relay1', target: 'delta', quality: 45, type: 'relay' },
  { source: 'epsilon', target: 'gamma', quality: 0, type: 'relay' },
])

const scale = ref(1)
const translateX = ref(0)
const translateY = ref(0)
const selectedNode = ref<TopoNode | null>(null)
const dragging = ref<{ nodeId: string; ox: number; oy: number } | null>(null)
const svgEl = ref<SVGSVGElement | null>(null)

function getNode(id: string) { return nodes.value.find(n => n.id === id) }

function linkPath(link: TopoLink) {
  const s = getNode(link.source)
  const t = getNode(link.target)
  if (!s || !t) return ''
  const mx = (s.x + t.x) / 2
  const my = (s.y + t.y) / 2 - 30
  return `M ${s.x} ${s.y} Q ${mx} ${my} ${t.x} ${t.y}`
}

function linkColor(q: number) {
  if (q === 0) return '#71717a' // gray = offline
  if (q >= 80) return 'oklch(0.6 0.18 145)' // green
  if (q >= 50) return 'oklch(0.7 0.18 80)'  // amber
  return 'oklch(0.6 0.22 25)'              // red
}

function nodeColor(status: TopoNode['status']) {
  if (status === 'online') return 'oklch(0.6 0.18 145)'
  if (status === 'relay') return 'var(--primary)'
  return '#71717a'
}

function zoom(delta: number) {
  scale.value = Math.max(0.3, Math.min(3, scale.value + delta))
}

function center() {
  scale.value = 1
  translateX.value = 0
  translateY.value = 0
}

// Drag nodes
function onNodeMouseDown(e: MouseEvent, node: TopoNode) {
  e.stopPropagation()
  selectedNode.value = node
  dragging.value = { nodeId: node.id, ox: e.clientX - node.x, oy: e.clientY - node.y }
}

function onMouseMove(e: MouseEvent) {
  if (!dragging.value) return
  const node = nodes.value.find(n => n.id === dragging.value!.nodeId)
  if (node) {
    node.x = e.clientX - dragging.value.ox
    node.y = e.clientY - dragging.value.oy
  }
}

function onMouseUp() { dragging.value = null }

onMounted(() => {
  window.addEventListener('mousemove', onMouseMove)
  window.addEventListener('mouseup', onMouseUp)
})
onUnmounted(() => {
  window.removeEventListener('mousemove', onMouseMove)
  window.removeEventListener('mouseup', onMouseUp)
})
</script>

<template>
  <div class="p-6 h-full flex flex-col gap-4">
    <!-- Controls -->
    <div class="flex items-center gap-2">
      <!-- Legend -->
      <div class="flex items-center gap-4 text-xs text-muted-foreground flex-1">
        <span class="flex items-center gap-1.5">
          <span class="size-2.5 rounded-full bg-emerald-500 inline-block" />在线 (p2p)
        </span>
        <span class="flex items-center gap-1.5">
          <span class="size-2.5 rounded-full bg-primary inline-block" />中继节点
        </span>
        <span class="flex items-center gap-1.5">
          <span class="size-2.5 rounded-full bg-zinc-400 inline-block" />离线
        </span>
        <span class="flex items-center gap-4 ml-2">
          <span class="flex items-center gap-1.5">
            <span class="h-0.5 w-5 inline-block bg-emerald-500 rounded" />优质链路
          </span>
          <span class="flex items-center gap-1.5">
            <span class="h-0.5 w-5 inline-block bg-amber-500 rounded" />中等链路
          </span>
          <span class="flex items-center gap-1.5">
            <span class="h-0.5 w-5 inline-block bg-zinc-400 rounded" />离线链路
          </span>
        </span>
      </div>

      <div class="flex items-center gap-1">
        <button @click="zoom(-0.2)" class="size-8 flex items-center justify-center rounded-lg border border-border hover:bg-muted transition-colors">
          <ZoomOut class="size-4" />
        </button>
        <span class="text-xs text-muted-foreground w-12 text-center">{{ Math.round(scale * 100) }}%</span>
        <button @click="zoom(0.2)" class="size-8 flex items-center justify-center rounded-lg border border-border hover:bg-muted transition-colors">
          <ZoomIn class="size-4" />
        </button>
        <button @click="center" class="size-8 flex items-center justify-center rounded-lg border border-border hover:bg-muted transition-colors ml-1">
          <Maximize2 class="size-4" />
        </button>
      </div>
    </div>

    <!-- Canvas -->
    <div class="relative flex-1 bg-card border border-border rounded-xl overflow-hidden" style="min-height: 480px">
      <svg
        ref="svgEl"
        class="w-full h-full select-none"
        :style="{ cursor: dragging ? 'grabbing' : 'grab' }"
      >
        <g :transform="`translate(${translateX},${translateY}) scale(${scale})`">
          <!-- Links -->
          <path
            v-for="link in links"
            :key="`${link.source}-${link.target}`"
            :d="linkPath(link)"
            fill="none"
            :stroke="linkColor(link.quality)"
            :stroke-width="link.quality > 0 ? 2 : 1.5"
            :stroke-dasharray="link.type === 'relay' ? '6 3' : 'none'"
            stroke-opacity="0.7"
          />

          <!-- Quality labels on links -->
          <text
            v-for="link in links.filter(l => l.quality > 0)"
            :key="`label-${link.source}-${link.target}`"
            :x="getNode(link.source) && getNode(link.target) ? (getNode(link.source)!.x + getNode(link.target)!.x) / 2 : 0"
            :y="getNode(link.source) && getNode(link.target) ? (getNode(link.source)!.y + getNode(link.target)!.y) / 2 - 35 : 0"
            text-anchor="middle"
            class="fill-muted-foreground"
            font-size="10"
            style="fill: var(--muted-foreground)"
          >{{ link.quality }}%</text>

          <!-- Nodes -->
          <g
            v-for="node in nodes"
            :key="node.id"
            :transform="`translate(${node.x},${node.y})`"
            @mousedown="onNodeMouseDown($event, node)"
            class="cursor-grab"
          >
            <!-- Outer glow ring for online -->
            <circle
              v-if="node.status !== 'offline'"
              r="24"
              :fill="nodeColor(node.status)"
              fill-opacity="0.12"
            />
            <circle
              r="18"
              :fill="nodeColor(node.status)"
              fill-opacity="0.15"
              :stroke="nodeColor(node.status)"
              stroke-width="2"
            />
            <!-- Status dot -->
            <circle
              cx="13" cy="-13"
              r="5"
              :fill="node.status === 'online' ? 'oklch(0.6 0.18 145)' : node.status === 'relay' ? 'var(--primary)' : '#71717a'"
              stroke="white"
              stroke-width="1.5"
            />
            <!-- Label -->
            <text
              y="36"
              text-anchor="middle"
              font-size="11"
              font-weight="500"
              style="fill: var(--foreground)"
            >{{ node.name }}</text>
          </g>
        </g>
      </svg>

      <!-- Node detail panel -->
      <div
        v-if="selectedNode"
        class="absolute top-4 right-4 w-64 bg-card border border-border rounded-xl shadow-lg p-4"
      >
        <div class="flex items-center justify-between mb-3">
          <h4 class="font-semibold text-sm">节点详情</h4>
          <button @click="selectedNode = null" class="text-muted-foreground hover:text-foreground">
            <X class="size-4" />
          </button>
        </div>
        <div class="space-y-2 text-sm">
          <div class="flex justify-between">
            <span class="text-muted-foreground">名称</span>
            <span class="font-medium">{{ selectedNode.name }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">状态</span>
            <span
              class="text-xs rounded-full px-2 py-0.5 font-medium"
              :class="selectedNode.status === 'online'
                ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                : selectedNode.status === 'relay'
                  ? 'bg-primary/10 text-primary'
                  : 'bg-muted text-muted-foreground'"
            >
              {{ selectedNode.status === 'online' ? '在线' : selectedNode.status === 'relay' ? '中继' : '离线' }}
            </span>
          </div>
          <div class="flex justify-between">
            <span class="text-muted-foreground">连接数</span>
            <span class="font-medium">
              {{ links.filter(l => l.source === selectedNode!.id || l.target === selectedNode!.id).length }}
            </span>
          </div>
          <div class="mt-3 pt-3 border-t border-border">
            <p class="text-xs text-muted-foreground mb-2">相连节点</p>
            <div class="space-y-1">
              <div
                v-for="link in links.filter(l => l.source === selectedNode!.id || l.target === selectedNode!.id)"
                :key="`${link.source}-${link.target}`"
                class="flex items-center justify-between text-xs"
              >
                <span class="text-muted-foreground">
                  {{ link.source === selectedNode!.id ? link.target : link.source }}
                </span>
                <span :class="link.quality >= 80 ? 'text-emerald-500' : link.quality >= 50 ? 'text-amber-500' : 'text-zinc-400'">
                  {{ link.quality > 0 ? link.quality + '%' : '离线' }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
