import { useState, useEffect } from 'react'
import {
  DndContext,
  closestCenter,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from '@dnd-kit/core'
import {
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
  arrayMove,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { GripVertical, Plus, Trash2 } from 'lucide-react'
import { useNavItems, useUpdateNavItems } from '../hooks/use-nav-items'
import { SaveIndicator, type SaveState } from '@/components/shared/save-indicator'
import type { NavItem } from '@/types'

type DraftItem = Omit<NavItem, 'id' | 'order'> & { _key: string }

const inputClass = [
  'border-2 border-ink/15 rounded-lg px-3 py-2 font-mono text-xs text-ink',
  'placeholder:text-ink/25 focus:outline-none focus:border-ink transition bg-white',
].join(' ')

function SortableRow({
  item,
  onChange,
  onRemove,
}: {
  item: DraftItem
  onChange: (key: string, field: keyof Omit<DraftItem, '_key'>, value: string | boolean) => void
  onRemove: (key: string) => void
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: item._key,
  })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  return (
    <div ref={setNodeRef} style={style} className="flex items-center gap-2">
      <button
        type="button"
        className="text-ink/25 hover:text-ink/50 transition cursor-grab active:cursor-grabbing shrink-0"
        {...attributes}
        {...listeners}
      >
        <GripVertical size={14} />
      </button>

      <input
        type="text"
        value={item.label}
        onChange={(e) => onChange(item._key, 'label', e.target.value)}
        placeholder="Label"
        className={`${inputClass} w-28`}
      />

      <input
        type="text"
        value={item.url}
        onChange={(e) => onChange(item._key, 'url', e.target.value)}
        placeholder="/about or https://…"
        className={`${inputClass} flex-1`}
      />

      <label className="flex items-center gap-1.5 shrink-0 cursor-pointer">
        <input
          type="checkbox"
          checked={item.is_external}
          onChange={(e) => onChange(item._key, 'is_external', e.target.checked)}
          className="accent-ink w-3 h-3"
        />
        <span className="font-mono text-[0.6rem] text-ink/50 uppercase tracking-widest">Ext</span>
      </label>

      <button
        type="button"
        onClick={() => onRemove(item._key)}
        className="text-ink/25 hover:text-red-500 transition shrink-0"
      >
        <Trash2 size={13} />
      </button>
    </div>
  )
}

let _keyCounter = 0
function genKey() {
  return `nav-${++_keyCounter}`
}

function toDraft(item: NavItem): DraftItem {
  return { _key: genKey(), label: item.label, url: item.url, is_external: item.is_external }
}

export function NavEditor() {
  const { data: navItems = [] } = useNavItems()
  const { mutateAsync: saveNav } = useUpdateNavItems()
  const [items, setItems] = useState<DraftItem[]>([])
  const [saveState, setSaveState] = useState<SaveState>('idle')

  useEffect(() => {
    setItems(navItems.map(toDraft))
  }, [navItems])

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 5 } }))

  function handleDragEnd(event: DragEndEvent) {
    const { active, over } = event
    if (over && active.id !== over.id) {
      setItems((prev) => {
        const oldIndex = prev.findIndex((i) => i._key === active.id)
        const newIndex = prev.findIndex((i) => i._key === over.id)
        return arrayMove(prev, oldIndex, newIndex)
      })
    }
  }

  function handleChange(key: string, field: keyof Omit<DraftItem, '_key'>, value: string | boolean) {
    setItems((prev) => prev.map((i) => (i._key === key ? { ...i, [field]: value } : i)))
  }

  function addItem() {
    setItems((prev) => [...prev, { _key: genKey(), label: '', url: '', is_external: false }])
  }

  function removeItem(key: string) {
    setItems((prev) => prev.filter((i) => i._key !== key))
  }

  async function handleSave() {
    setSaveState('saving')
    try {
      await saveNav(items.map(({ label, url, is_external }) => ({ label, url, is_external })))
      setSaveState('saved')
      setTimeout(() => setSaveState('idle'), 2000)
    } catch {
      setSaveState('error')
      setTimeout(() => setSaveState('idle'), 3000)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-2 pb-2 border-b-2 border-ink/8">
        <span className="font-mono text-[0.6rem] uppercase tracking-widest text-ink/35 w-[22px]" />
        <span className="font-mono text-[0.6rem] uppercase tracking-widest text-ink/35 w-28">Label</span>
        <span className="font-mono text-[0.6rem] uppercase tracking-widest text-ink/35 flex-1">URL</span>
        <span className="font-mono text-[0.6rem] uppercase tracking-widest text-ink/35 w-12 text-center">Ext</span>
        <span className="w-[21px]" />
      </div>

      <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
        <SortableContext items={items.map((i) => i._key)} strategy={verticalListSortingStrategy}>
          <div className="flex flex-col gap-2">
            {items.map((item) => (
              <SortableRow
                key={item._key}
                item={item}
                onChange={handleChange}
                onRemove={removeItem}
              />
            ))}
          </div>
        </SortableContext>
      </DndContext>

      {items.length === 0 && (
        <p className="font-mono text-[0.65rem] text-ink/30 text-center py-4">No nav items yet.</p>
      )}

      <button
        type="button"
        onClick={addItem}
        className="flex items-center gap-1.5 font-mono text-xs text-ink/50 hover:text-ink transition self-start"
      >
        <Plus size={13} />
        Add item
      </button>

      <div className="flex items-center justify-between pt-2 border-t-2 border-ink/8">
        <SaveIndicator state={saveState} />
        <button
          type="button"
          onClick={handleSave}
          disabled={saveState === 'saving'}
          className="border-2 border-ink rounded-xl px-5 py-2.5 font-mono text-xs font-bold uppercase tracking-widest text-ink hover:bg-ink hover:text-cream transition disabled:opacity-40"
        >
          Save
        </button>
      </div>
    </div>
  )
}
