import { Plus, Trash2 } from 'lucide-react'
import type { FeaturesSection, FeaturesItem } from '@/types'
import type { SectionEditorProps } from './section-editor'
import { Field, inputClass, textareaClass } from './editor-primitives'

const EMPTY_ITEM: FeaturesItem = { icon: '', title: '', description: '' }

export function FeaturesEditor({ value, onChange }: SectionEditorProps) {
  const section = value as Partial<FeaturesSection>
  const items: FeaturesItem[] = section.items ?? []

  function updateItem(idx: number, field: keyof FeaturesItem, val: string) {
    onChange({
      ...value,
      items: items.map((item, i) => i === idx ? { ...item, [field]: val } : item),
    })
  }

  function addItem() {
    onChange({ ...value, items: [...items, { ...EMPTY_ITEM }] })
  }

  function removeItem(idx: number) {
    onChange({ ...value, items: items.filter((_, i) => i !== idx) })
  }

  return (
    <div className="flex flex-col gap-6">
      {items.map((item, idx) => (
        <div key={idx} className="border-2 border-ink/10 rounded-lg p-5 flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <span className="font-mono text-xs text-ink/40 uppercase tracking-widest">
              Item {idx + 1}
            </span>
            <button
              type="button"
              onClick={() => removeItem(idx)}
              className="text-brand-red opacity-50 hover:opacity-100 transition"
              aria-label="Remove item"
            >
              <Trash2 size={14} />
            </button>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <Field label="Icon">
              <input
                type="text"
                value={item.icon}
                onChange={e => updateItem(idx, 'icon', e.target.value)}
                placeholder="e.g. star, check, zap"
                className={inputClass}
              />
            </Field>
            <Field label="Title">
              <input
                type="text"
                value={item.title}
                onChange={e => updateItem(idx, 'title', e.target.value)}
                placeholder="Feature name"
                className={inputClass}
              />
            </Field>
          </div>

          <Field label="Description">
            <textarea
              rows={2}
              value={item.description}
              onChange={e => updateItem(idx, 'description', e.target.value)}
              placeholder="Brief description of this feature"
              className={textareaClass}
            />
          </Field>
        </div>
      ))}

      <button
        type="button"
        onClick={addItem}
        className="flex items-center gap-2 font-mono text-xs text-ink/50 hover:text-ink transition border-2 border-dashed border-ink/20 hover:border-ink/40 rounded-lg px-4 py-3 w-full justify-center"
      >
        <Plus size={14} />
        Add feature
      </button>
    </div>
  )
}
