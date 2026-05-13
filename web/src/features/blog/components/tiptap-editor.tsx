import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import Image from '@tiptap/extension-image'
import Placeholder from '@tiptap/extension-placeholder'
import { cn } from '@/lib/utils'

interface TiptapEditorProps {
  content: string
  onChange: (html: string) => void
  placeholder?: string
}

function ToolbarBtn({
  onClick,
  isActive,
  label,
}: {
  onClick: () => void
  isActive?: boolean
  label: string
}) {
  return (
    <button
      type="button"
      onMouseDown={(e) => {
        e.preventDefault()
        onClick()
      }}
      className={cn(
        'px-2.5 py-1.5 rounded font-mono text-xs border-2 transition select-none',
        isActive
          ? 'border-ink bg-ink text-cream font-bold'
          : 'border-transparent text-ink/50 hover:border-ink/20 hover:bg-white hover:text-ink',
      )}
    >
      {label}
    </button>
  )
}

export function TiptapEditor({ content, onChange, placeholder = 'Start writing...' }: TiptapEditorProps) {
  const editor = useEditor({
    extensions: [
      StarterKit.configure({ heading: { levels: [2, 3] } }),
      Link.configure({ openOnClick: false }),
      Image,
      Placeholder.configure({ placeholder }),
    ],
    content,
    onUpdate: ({ editor }) => onChange(editor.getHTML()),
    editorProps: {
      attributes: {
        class: 'tiptap-content outline-none min-h-96 p-6 font-sans text-ink text-base leading-relaxed',
      },
    },
  })

  if (!editor) return null

  function handleSetLink() {
    const previous = editor!.getAttributes('link').href as string | undefined
    const url = window.prompt('Link URL:', previous ?? '')
    if (url === null) return
    if (url === '') {
      editor!.chain().focus().unsetLink().run()
    } else {
      editor!.chain().focus().setLink({ href: url }).run()
    }
  }

  return (
    <div className="flex flex-col">
      <div className="px-4 py-2 border-b-2 border-ink/10 flex items-center gap-0.5 flex-wrap bg-ink/[0.03]">
        <ToolbarBtn
          onClick={() => editor.chain().focus().toggleBold().run()}
          isActive={editor.isActive('bold')}
          label="B"
        />
        <ToolbarBtn
          onClick={() => editor.chain().focus().toggleItalic().run()}
          isActive={editor.isActive('italic')}
          label="I"
        />
        <span className="w-px h-4 bg-ink/15 mx-1" />
        <ToolbarBtn
          onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
          isActive={editor.isActive('heading', { level: 2 })}
          label="H2"
        />
        <ToolbarBtn
          onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
          isActive={editor.isActive('heading', { level: 3 })}
          label="H3"
        />
        <span className="w-px h-4 bg-ink/15 mx-1" />
        <ToolbarBtn
          onClick={() => editor.chain().focus().toggleBulletList().run()}
          isActive={editor.isActive('bulletList')}
          label="• List"
        />
        <ToolbarBtn
          onClick={() => editor.chain().focus().toggleBlockquote().run()}
          isActive={editor.isActive('blockquote')}
          label={'" Quote'}
        />
        <ToolbarBtn
          onClick={handleSetLink}
          isActive={editor.isActive('link')}
          label="Link"
        />
      </div>

      <EditorContent editor={editor} />
    </div>
  )
}
