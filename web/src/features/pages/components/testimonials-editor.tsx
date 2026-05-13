import { Plus, Trash2 } from 'lucide-react'
import type { TestimonialsSection, TestimonialItem } from '@/types'
import type { SectionEditorProps } from './section-editor'
import { Field, inputClass, textareaClass } from './editor-primitives'

const EMPTY_ITEM: TestimonialItem = { quote: '', author: '', role: '', avatar: '' }

export function TestimonialsEditor({ value, onChange }: SectionEditorProps) {
  const section = value as Partial<TestimonialsSection>
  const items: TestimonialItem[] = section.items ?? []

  function updateItem(idx: number, field: keyof TestimonialItem, val: string) {
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
              Testimonial {idx + 1}
            </span>
            <button
              type="button"
              onClick={() => removeItem(idx)}
              className="text-brand-red opacity-50 hover:opacity-100 transition"
              aria-label="Remove testimonial"
            >
              <Trash2 size={14} />
            </button>
          </div>

          <Field label="Quote">
            <textarea
              rows={3}
              value={item.quote}
              onChange={e => updateItem(idx, 'quote', e.target.value)}
              placeholder="What they said…"
              className={textareaClass}
            />
          </Field>

          <div className="grid grid-cols-2 gap-4">
            <Field label="Author">
              <input
                type="text"
                value={item.author}
                onChange={e => updateItem(idx, 'author', e.target.value)}
                placeholder="Jane Smith"
                className={inputClass}
              />
            </Field>
            <Field label="Role">
              <input
                type="text"
                value={item.role}
                onChange={e => updateItem(idx, 'role', e.target.value)}
                placeholder="CEO, Acme Corp"
                className={inputClass}
              />
            </Field>
          </div>

          <Field label="Avatar URL">
            <input
              type="text"
              value={item.avatar ?? ''}
              onChange={e => updateItem(idx, 'avatar', e.target.value)}
              placeholder="https://… (optional)"
              className={inputClass}
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
        Add testimonial
      </button>
    </div>
  )
}
