import { Plus, Trash2 } from 'lucide-react'
import type { ProcessSection, ProcessStep } from '@/types'
import type { SectionEditorProps } from './section-editor'
import { Field, inputClass, textareaClass } from './editor-primitives'

const EMPTY_STEP: ProcessStep = { title: '', description: '' }

export function ProcessEditor({ value, onChange }: SectionEditorProps) {
  const section = value as Partial<ProcessSection>
  const steps: ProcessStep[] = section.steps ?? []

  function updateStep(idx: number, field: keyof ProcessStep, val: string) {
    onChange({ ...value, steps: steps.map((s, i) => i === idx ? { ...s, [field]: val } : s) })
  }

  return (
    <div className="flex flex-col gap-6">
      <Field label="Heading">
        <input type="text" value={section.title ?? ''} onChange={e => onChange({ ...value, title: e.target.value })} className={inputClass} />
      </Field>
      <Field label="Subheading">
        <textarea rows={2} value={section.subtitle ?? ''} onChange={e => onChange({ ...value, subtitle: e.target.value })} className={textareaClass} />
      </Field>

      {steps.map((step, idx) => (
        <div key={idx} className="border-2 border-ink/10 rounded-lg p-4 flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <span className="font-mono text-xs text-ink/40 uppercase tracking-widest">Step {idx + 1}</span>
            <button type="button" onClick={() => onChange({ ...value, steps: steps.filter((_, i) => i !== idx) })} className="text-brand-red opacity-50 hover:opacity-100 transition">
              <Trash2 size={14} />
            </button>
          </div>
          <Field label="Title"><input type="text" value={step.title} onChange={e => updateStep(idx, 'title', e.target.value)} className={inputClass} /></Field>
          <Field label="Description"><textarea rows={2} value={step.description} onChange={e => updateStep(idx, 'description', e.target.value)} className={textareaClass} /></Field>
        </div>
      ))}

      <button type="button" onClick={() => onChange({ ...value, steps: [...steps, { ...EMPTY_STEP }] })} className="flex items-center gap-2 font-mono text-xs text-ink/50 hover:text-ink transition border-2 border-dashed border-ink/20 hover:border-ink/40 rounded-lg px-4 py-3 w-full justify-center">
        <Plus size={14} />Add step
      </button>
    </div>
  )
}
