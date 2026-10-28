<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArrowDown,
  ArrowRight,
  ArrowUp,
  ArrowUpDown,
  CheckCircle2,
  Circle,
  CircleHelp,
  CirclePlus,
  CircleSlash2,
  Clock3,
  MoreHorizontal,
  SlidersHorizontal,
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
} from '@lucide/vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { demoTasks, type TaskPriority, type TaskStatus } from '@/fixtures/tasks'

/**
 * Demo page shipped with the starter: a data table wired to static fixtures.
 *
 * Sorting and the Status/Priority facet buttons are deliberately inert placeholders —
 * they exist to show the intended layout. See `AdminUsersView.vue` for the pattern this
 * page should follow once it is backed by a real API (composable + server-side paging).
 */

type TaskColumnId = 'task' | 'title' | 'status' | 'priority'

const { t } = useI18n()

const filterText = ref('')
const rowsPerPage = ref('10')
const currentPage = ref(1)
const selectedTaskIds = ref<string[]>([])
const visibleColumns = ref<TaskColumnId[]>([
  'task',
  'title',
  'status',
  'priority',
])

const columnOptions = [
  'task',
  'title',
  'status',
  'priority',
] as const satisfies readonly TaskColumnId[]

// Icons only; the visible text comes from i18n so the meta tables stay language-neutral.
const statusIcons = {
  backlog: CircleHelp,
  todo: Circle,
  inProgress: Clock3,
  done: CheckCircle2,
  canceled: CircleSlash2,
} as const satisfies Record<TaskStatus, object>

const priorityIcons = {
  low: ArrowDown,
  medium: ArrowRight,
  high: ArrowUp,
} as const satisfies Record<TaskPriority, object>

// Search matches what the user actually sees, so the translated label/status/priority
// are folded into the haystack alongside the raw id and title.
const searchableTasks = computed(() =>
  demoTasks.map((task) => ({
    task,
    haystack: [
      task.id,
      task.title,
      t(`tasks.label.${task.label}`),
      t(`tasks.status.${task.status}`),
      t(`tasks.priority.${task.priority}`),
    ]
      .join(' ')
      .toLowerCase(),
  })),
)

const filteredTasks = computed(() => {
  const query = filterText.value.trim().toLowerCase()

  if (query === '') {
    return demoTasks
  }

  return searchableTasks.value
    .filter((entry) => entry.haystack.includes(query))
    .map((entry) => entry.task)
})

const pageSize = computed(() => Number(rowsPerPage.value))
const pageCount = computed(() =>
  Math.max(1, Math.ceil(filteredTasks.value.length / pageSize.value)),
)
const pagedTasks = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredTasks.value.slice(start, start + pageSize.value)
})
const isColumnVisible = computed(() => {
  const visible = new Set(visibleColumns.value)
  return (columnId: TaskColumnId) => visible.has(columnId)
})
const selectedCount = computed(
  () =>
    filteredTasks.value.filter((task) =>
      selectedTaskIds.value.includes(task.id),
    ).length,
)
const allPageSelected = computed(
  () =>
    pagedTasks.value.length > 0 &&
    pagedTasks.value.every((task) => selectedTaskIds.value.includes(task.id)),
)
const somePageSelected = computed(
  () =>
    !allPageSelected.value &&
    pagedTasks.value.some((task) => selectedTaskIds.value.includes(task.id)),
)

watch([filterText, rowsPerPage], () => {
  currentPage.value = 1
})

watch(pageCount, (nextPageCount) => {
  if (currentPage.value > nextPageCount) {
    currentPage.value = nextPageCount
  }
})

function toggleAllOnPage(nextValue: boolean) {
  const pageIds = pagedTasks.value.map((task) => task.id)

  if (nextValue) {
    selectedTaskIds.value = Array.from(
      new Set([...selectedTaskIds.value, ...pageIds]),
    )
    return
  }

  selectedTaskIds.value = selectedTaskIds.value.filter(
    (id) => !pageIds.includes(id),
  )
}

function toggleTaskSelection(taskId: string, nextValue: boolean) {
  if (nextValue) {
    selectedTaskIds.value = Array.from(
      new Set([...selectedTaskIds.value, taskId]),
    )
    return
  }

  selectedTaskIds.value = selectedTaskIds.value.filter((id) => id !== taskId)
}

function toggleColumnVisibility(columnId: TaskColumnId, nextValue: boolean) {
  if (nextValue) {
    // Rebuild from `columnOptions` rather than appending, so columns keep their declared
    // display order regardless of the order the user toggled them back on.
    visibleColumns.value = columnOptions.filter(
      (column) => column === columnId || visibleColumns.value.includes(column),
    )
    return
  }

  if (visibleColumns.value.length === 1) {
    return
  }

  visibleColumns.value = visibleColumns.value.filter((id) => id !== columnId)
}
</script>

<template>
  <div class="flex flex-1 flex-col gap-6 p-4 lg:p-6">
    <div class="flex items-start justify-between gap-4">
      <div class="space-y-1">
        <h2 class="text-2xl font-semibold">{{ t('tasks.title') }}</h2>
        <p class="text-sm text-muted-foreground">
          {{ t('tasks.description') }}
        </p>
      </div>
    </div>

    <div
      class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between"
    >
      <div class="flex flex-col gap-3 sm:flex-row">
        <Input
          v-model="filterText"
          :placeholder="t('tasks.filterPlaceholder')"
          class="block-9 inline-full sm:inline-80 lg:inline-94"
        />
        <div class="flex flex-wrap gap-3">
          <!-- Facet filters are layout placeholders; wire them up alongside a real API. -->
          <Button
            variant="outline"
            size="sm"
            disabled
            class="justify-start gap-2.5 block-9"
          >
            <CirclePlus class="block-4 inline-4" />
            {{ t('tasks.filter.status') }}
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled
            class="justify-start gap-2.5 block-9"
          >
            <CirclePlus class="block-4 inline-4" />
            {{ t('tasks.filter.priority') }}
          </Button>
        </div>
      </div>

      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button
            variant="outline"
            size="sm"
            class="gap-2.5 block-9 inline-full sm:inline-auto"
          >
            <SlidersHorizontal class="block-4 inline-4" />
            {{ t('table.action.view') }}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="end"
          class="inline-48"
        >
          <DropdownMenuCheckboxItem
            v-for="column in columnOptions"
            :key="column"
            :model-value="isColumnVisible(column)"
            :disabled="visibleColumns.length === 1 && isColumnVisible(column)"
            @update:model-value="
              toggleColumnVisibility(column, $event === true)
            "
          >
            {{ t(`tasks.column.${column}`) }}
          </DropdownMenuCheckboxItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>

    <div class="overflow-hidden rounded-lg border">
      <Table>
        <TableHeader class="bg-muted">
          <TableRow class="hover:bg-transparent">
            <TableHead class="inline-12">
              <Checkbox
                :aria-label="t('tasks.selectAllAriaLabel')"
                :model-value="
                  allPageSelected || (somePageSelected && 'indeterminate')
                "
                @update:model-value="toggleAllOnPage($event === true)"
              />
            </TableHead>
            <TableHead
              v-if="isColumnVisible('task')"
              class="text-sm font-semibold inline-17.5 min-inline-17.5"
            >
              {{ t('tasks.column.task') }}
            </TableHead>
            <TableHead
              v-if="isColumnVisible('title')"
              class="text-sm font-semibold min-inline-105"
            >
              <!-- Sorting is not implemented for the demo fixtures. -->
              <span class="inline-flex items-center gap-2 font-semibold">
                {{ t('tasks.column.title') }}
                <ArrowUpDown class="text-muted-foreground block-4 inline-4" />
              </span>
            </TableHead>
            <TableHead
              v-if="isColumnVisible('status')"
              class="text-sm font-semibold inline-22 min-inline-22"
            >
              <span class="inline-flex items-center gap-2 font-semibold">
                {{ t('tasks.column.status') }}
                <ArrowUpDown class="text-muted-foreground block-4 inline-4" />
              </span>
            </TableHead>
            <TableHead
              v-if="isColumnVisible('priority')"
              class="text-sm font-semibold inline-20 min-inline-20"
            >
              <span class="inline-flex items-center gap-2 font-semibold">
                {{ t('tasks.column.priority') }}
                <ArrowUpDown class="text-muted-foreground block-4 inline-4" />
              </span>
            </TableHead>
            <TableHead class="inline-14" />
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow
            v-for="task in pagedTasks"
            :key="task.id"
          >
            <TableCell class="align-middle">
              <Checkbox
                :aria-label="t('tasks.selectRowAriaLabel')"
                :model-value="selectedTaskIds.includes(task.id)"
                @update:model-value="
                  toggleTaskSelection(task.id, $event === true)
                "
              />
            </TableCell>
            <TableCell
              v-if="isColumnVisible('task')"
              class="font-medium"
            >
              {{ task.id }}
            </TableCell>
            <TableCell v-if="isColumnVisible('title')">
              <div class="flex items-center gap-3 min-inline-0">
                <Badge
                  variant="outline"
                  class="rounded-full px-3 py-1 text-xs font-medium"
                >
                  {{ t(`tasks.label.${task.label}`) }}
                </Badge>
                <span class="block truncate text-sm font-medium min-inline-0">
                  {{ task.title }}
                </span>
              </div>
            </TableCell>
            <TableCell v-if="isColumnVisible('status')">
              <div class="flex items-center gap-2 text-sm">
                <component
                  :is="statusIcons[task.status]"
                  class="shrink-0 text-muted-foreground block-4 inline-4"
                />
                <span>{{ t(`tasks.status.${task.status}`) }}</span>
              </div>
            </TableCell>
            <TableCell v-if="isColumnVisible('priority')">
              <div class="flex items-center gap-2 text-sm">
                <component
                  :is="priorityIcons[task.priority]"
                  class="shrink-0 text-muted-foreground block-4 inline-4"
                />
                <span>{{ t(`tasks.priority.${task.priority}`) }}</span>
              </div>
            </TableCell>
            <TableCell>
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    class="rounded-full"
                  >
                    <span class="sr-only">{{
                      t('tasks.openMenuAriaLabel')
                    }}</span>
                    <MoreHorizontal class="block-4 inline-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem>{{
                    t('tasks.action.viewTask')
                  }}</DropdownMenuItem>
                  <DropdownMenuItem>{{
                    t('tasks.action.copyId')
                  }}</DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem>{{
                    t('tasks.action.archive')
                  }}</DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>

    <div
      class="flex flex-col gap-4 px-1 lg:flex-row lg:items-center lg:justify-between"
    >
      <div class="text-sm text-muted-foreground">
        {{
          t('table.pagination.rowSelected', {
            selected: selectedCount,
            total: filteredTasks.length,
          })
        }}
      </div>

      <div
        class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-end sm:gap-6"
      >
        <div class="flex items-center gap-3">
          <span class="text-sm font-medium">{{
            t('table.pagination.rowsPerPage')
          }}</span>
          <Select v-model="rowsPerPage">
            <SelectTrigger class="inline-20">
              <SelectValue placeholder="10" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="10">10</SelectItem>
              <SelectItem value="20">20</SelectItem>
              <SelectItem value="30">30</SelectItem>
              <SelectItem value="40">40</SelectItem>
              <SelectItem value="50">50</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div class="text-sm font-medium">
          {{
            t('table.pagination.pageOf', {
              page: currentPage,
              total: pageCount,
            })
          }}
        </div>

        <div class="flex items-center gap-2">
          <Button
            variant="outline"
            size="icon"
            :disabled="currentPage === 1"
            @click="currentPage = 1"
          >
            <span class="sr-only">{{
              t('table.pagination.goToFirstPage')
            }}</span>
            <ChevronsLeft class="block-4 inline-4" />
          </Button>
          <Button
            variant="outline"
            size="icon"
            :disabled="currentPage === 1"
            @click="currentPage = Math.max(1, currentPage - 1)"
          >
            <span class="sr-only">{{
              t('table.pagination.goToPreviousPage')
            }}</span>
            <ChevronLeft class="block-4 inline-4" />
          </Button>
          <Button
            variant="outline"
            size="icon"
            :disabled="currentPage === pageCount"
            @click="currentPage = Math.min(pageCount, currentPage + 1)"
          >
            <span class="sr-only">{{
              t('table.pagination.goToNextPage')
            }}</span>
            <ChevronRight class="block-4 inline-4" />
          </Button>
          <Button
            variant="outline"
            size="icon"
            :disabled="currentPage === pageCount"
            @click="currentPage = pageCount"
          >
            <span class="sr-only">{{
              t('table.pagination.goToLastPage')
            }}</span>
            <ChevronsRight class="block-4 inline-4" />
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>
