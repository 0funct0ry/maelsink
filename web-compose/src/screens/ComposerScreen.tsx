import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import CodeMirror, { type ReactCodeMirrorRef } from '@uiw/react-codemirror'
import { json as jsonLang } from '@codemirror/lang-json'
import { Paperclip, Trash2 } from 'lucide-react'
import Button from '../components/Button'
import Modal from '../components/Modal'
import { useVarsStore } from '../stores/useVarsStore'
import { useDraftsStore } from '../stores/useDraftsStore'
import { emlLanguage } from '../lib/eml-lang'
import { exampleContent, exampleEmlAttachments, exampleTemplates } from '../lib/exampleTemplates'
import { extractHtmlPreview } from '../lib/previewHtml'
import { useIsDarkMode } from '../lib/useIsDarkMode'
import {
  ComposeApiError,
  getFunctions,
  renderTemplate,
  sendTemplate,
  type AttachmentInput,
  type FuncDoc,
  type ResolvedAttachment,
  type TemplateFormat,
} from '../lib/composeApi'

interface RecentSend {
  at: number
  to: string[]
  ok: boolean
  message: string
}

interface RenderError {
  message: string
  line?: number
  column?: number
}

const DEBOUNCE_MS = 400

function placeholderArgs(argsDoc: string): string {
  // FuncDoc.Args is a human-readable signature like "name string, n int" —
  // just count comma-separated pieces and emit generic placeholders rather
  // than trying to fully parse Go-ish type syntax.
  const trimmed = argsDoc.trim()
  if (!trimmed) return ''
  const count = trimmed.split(',').filter((p) => p.trim().length > 0).length
  return Array.from({ length: count }, (_, i) => `arg${i + 1}`).join(' ')
}

export default function ComposerScreen() {
  const vars = useVarsStore((s) => s.vars)
  const drafts = useDraftsStore((s) => s.drafts)
  const saveDraft = useDraftsStore((s) => s.saveDraft)
  const deleteDraft = useDraftsStore((s) => s.deleteDraft)

  const [mode, setMode] = useState<TemplateFormat>('eml')
  const [emlContent, setEmlContent] = useState(exampleTemplates[0].eml)
  const [jsonContent, setJsonContent] = useState(exampleTemplates[0].json)
  // Attachments only apply to EML mode (JSON's own MessageSpec carries its
  // attachments inline in the template text instead — see composeApi.ts's
  // AttachmentInput doc).
  const [emlAttachments, setEmlAttachments] = useState<AttachmentInput[]>(exampleEmlAttachments(exampleTemplates[0].id))
  const [exampleMenuOpen, setExampleMenuOpen] = useState(false)
  const exampleMenuRef = useRef<HTMLDivElement>(null)
  const isDarkMode = useIsDarkMode()

  const [preview, setPreview] = useState('')
  const [previewAttachments, setPreviewAttachments] = useState<ResolvedAttachment[]>([])
  const [renderError, setRenderError] = useState<RenderError | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [previewTab, setPreviewTab] = useState<'rendered' | 'raw'>('rendered')

  const [functions, setFunctions] = useState<FuncDoc[]>([])
  const [functionQuery, setFunctionQuery] = useState('')
  const [sending, setSending] = useState(false)
  const [recentSends, setRecentSends] = useState<RecentSend[]>([])

  const [draftsModalOpen, setDraftsModalOpen] = useState(false)
  const [draftName, setDraftName] = useState('')

  const editorRef = useRef<ReactCodeMirrorRef>(null)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const content = mode === 'eml' ? emlContent : jsonContent
  const setContent = mode === 'eml' ? setEmlContent : setJsonContent

  useEffect(() => {
    getFunctions()
      .then(setFunctions)
      .catch(() => setFunctions([]))
  }, [])

  useEffect(() => {
    if (!exampleMenuOpen) return
    function handleClickOutside(event: MouseEvent) {
      if (exampleMenuRef.current && !exampleMenuRef.current.contains(event.target as Node)) {
        setExampleMenuOpen(false)
      }
    }
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') setExampleMenuOpen(false)
    }
    document.addEventListener('mousedown', handleClickOutside)
    window.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      window.removeEventListener('keydown', handleKeyDown)
    }
  }, [exampleMenuOpen])

  const runPreview = useCallback(
    (
      currentMode: TemplateFormat,
      currentContent: string,
      currentVars: Record<string, string>,
      currentAttachments: AttachmentInput[],
    ) => {
      setPreviewLoading(true)
      renderTemplate({
        template: currentContent,
        format: currentMode,
        vars: currentVars,
        attachments: currentMode === 'eml' ? currentAttachments : undefined,
      })
        .then((res) => {
          setPreview(res.rendered)
          setPreviewAttachments(res.attachments ?? [])
          setRenderError(null)
        })
        .catch((err) => {
          if (err instanceof ComposeApiError) {
            setRenderError({ message: err.message, line: err.line, column: err.column })
          } else {
            setRenderError({ message: String(err) })
          }
        })
        .finally(() => setPreviewLoading(false))
    },
    [],
  )

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => {
      runPreview(mode, content, vars, emlAttachments)
    }, DEBOUNCE_MS)
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
    }
  }, [mode, content, vars, emlAttachments, runPreview])

  function insertAtCursor(text: string) {
    const view = editorRef.current?.view
    if (!view) {
      setContent(content + text)
      return
    }
    const pos = view.state.selection.main.head
    view.dispatch({
      changes: { from: pos, to: pos, insert: text },
      selection: { anchor: pos + text.length },
    })
    setContent(view.state.doc.toString())
  }

  async function handleSend() {
    setSending(true)
    try {
      const res = await sendTemplate({
        template: content,
        format: mode,
        vars,
        attachments: mode === 'eml' ? emlAttachments : undefined,
      })
      setRecentSends((prev) => [{ at: Date.now(), to: res.to, ok: true, message: 'Sent' }, ...prev].slice(0, 20))
    } catch (err) {
      const message = err instanceof ComposeApiError ? err.message : String(err)
      setRecentSends((prev) => [{ at: Date.now(), to: [], ok: false, message }, ...prev].slice(0, 20))
    } finally {
      setSending(false)
    }
  }

  function handleLoadExample(id: string) {
    const content = exampleContent(id, mode)
    if (content !== undefined) setContent(content)
    setEmlAttachments(exampleEmlAttachments(id))
    setExampleMenuOpen(false)
  }

  function addAttachment() {
    setEmlAttachments((prev) => [...prev, { path: '', filename: '' }])
  }

  function updateAttachment(index: number, patch: Partial<AttachmentInput>) {
    setEmlAttachments((prev) => prev.map((a, i) => (i === index ? { ...a, ...patch } : a)))
  }

  function removeAttachment(index: number) {
    setEmlAttachments((prev) => prev.filter((_, i) => i !== index))
  }

  function handleSaveDraft() {
    const name = draftName.trim()
    if (!name) return
    saveDraft(name, { format: mode, content, attachments: mode === 'eml' ? emlAttachments : undefined })
    setDraftName('')
  }

  function handleLoadDraft(name: string) {
    const draft = drafts[name]
    if (!draft) return
    setMode(draft.format)
    if (draft.format === 'eml') {
      setEmlContent(draft.content)
      setEmlAttachments(draft.attachments ?? [])
    } else {
      setJsonContent(draft.content)
    }
    setDraftsModalOpen(false)
  }

  const filteredFunctions = useMemo(() => {
    const q = functionQuery.trim().toLowerCase()
    if (!q) return functions
    return functions.filter(
      (f) => f.name.toLowerCase().includes(q) || f.description.toLowerCase().includes(q),
    )
  }, [functions, functionQuery])

  const extensions = useMemo(() => [mode === 'eml' ? emlLanguage : jsonLang()], [mode])

  const htmlPreview = useMemo(() => extractHtmlPreview(mode, preview), [mode, preview])

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="flex items-center justify-between gap-2 border-b border-border-soft px-4 py-2">
        <div className="flex items-center gap-1 rounded-md bg-surface-2 p-1">
          <button
            type="button"
            onClick={() => setMode('eml')}
            className={`rounded px-3 py-1 text-sm ${mode === 'eml' ? 'bg-accent text-white' : 'text-text-secondary'}`}
          >
            EML
          </button>
          <button
            type="button"
            onClick={() => setMode('json')}
            className={`rounded px-3 py-1 text-sm ${mode === 'json' ? 'bg-accent text-white' : 'text-text-secondary'}`}
          >
            JSON
          </button>
        </div>
        <div className="flex items-center gap-2">
          <div className="relative" ref={exampleMenuRef}>
            <Button variant="secondary" onClick={() => setExampleMenuOpen((v) => !v)}>
              Load example
            </Button>
            {exampleMenuOpen && (
              <div className="scrollbar-thin absolute right-0 z-10 mt-1 max-h-80 w-64 overflow-y-auto rounded-md border border-border bg-surface p-1 shadow-lg">
                {exampleTemplates.map((example) => (
                  <button
                    key={example.id}
                    type="button"
                    className="block w-full rounded px-2 py-1.5 text-left text-sm text-text-primary hover:bg-surface-2"
                    onClick={() => handleLoadExample(example.id)}
                  >
                    {example.label}
                  </button>
                ))}
              </div>
            )}
          </div>
          <Button variant="secondary" onClick={() => setDraftsModalOpen(true)}>
            Drafts
          </Button>
          <Button onClick={handleSend} loading={sending}>
            Send
          </Button>
        </div>
      </div>

      <div className="flex flex-1 overflow-hidden">
        <div className="scrollbar-thin flex w-56 flex-col overflow-y-auto border-r border-border-soft p-3">
          <h2 className="mb-2 text-xs font-semibold uppercase text-text-tertiary">Vars</h2>
          {Object.keys(vars).length === 0 ? (
            <p className="text-xs text-text-secondary">No vars set. Add some on the Vars screen.</p>
          ) : (
            <ul className="mb-4 space-y-1">
              {Object.entries(vars).map(([key]) => (
                <li key={key}>
                  <button
                    type="button"
                    className="w-full truncate rounded px-2 py-1 text-left font-mono text-xs text-text-primary hover:bg-surface-2"
                    onClick={() => insertAtCursor(`{{.${key}}}`)}
                    title={`Insert {{.${key}}}`}
                  >
                    {key}
                  </button>
                </li>
              ))}
            </ul>
          )}

          {mode === 'eml' && (
            <>
              <h2 className="mb-2 mt-2 text-xs font-semibold uppercase text-text-tertiary">Attachments</h2>
              {emlAttachments.length === 0 ? (
                <p className="mb-2 text-xs text-text-secondary">No attachments.</p>
              ) : (
                <ul className="mb-2 space-y-1.5">
                  {emlAttachments.map((att, i) => (
                    <li key={i} className="space-y-1">
                      <input
                        className="w-full rounded-md border border-border bg-bg px-2 py-1 font-mono text-xs text-text-primary"
                        placeholder="{{ fPDF }}"
                        value={att.path}
                        onChange={(e) => updateAttachment(i, { path: e.target.value })}
                      />
                      <div className="flex gap-1">
                        <input
                          className="w-full rounded-md border border-border bg-bg px-2 py-1 text-xs text-text-primary"
                          placeholder="filename"
                          value={att.filename}
                          onChange={(e) => updateAttachment(i, { filename: e.target.value })}
                        />
                        <Button variant="ghost" onClick={() => removeAttachment(i)}>
                          ×
                        </Button>
                      </div>
                    </li>
                  ))}
                </ul>
              )}
              <Button variant="secondary" onClick={addAttachment}>
                + Add attachment
              </Button>
            </>
          )}

          <h2 className="mb-2 mt-2 text-xs font-semibold uppercase text-text-tertiary">Functions</h2>
          <input
            className="mb-2 w-full rounded-md border border-border bg-bg px-2 py-1 text-xs text-text-primary"
            placeholder="Search functions…"
            value={functionQuery}
            onChange={(e) => setFunctionQuery(e.target.value)}
          />
          <ul className="scrollbar-thin space-y-1 overflow-y-auto">
            {filteredFunctions.map((fn) => (
              <li key={fn.name}>
                <button
                  type="button"
                  className="w-full truncate rounded px-2 py-1 text-left font-mono text-xs text-text-primary hover:bg-surface-2"
                  onClick={() => insertAtCursor(`{{ ${fn.name} ${placeholderArgs(fn.args)} }}`.replace(/\s+/g, ' ').trim())}
                  title={fn.description}
                >
                  {fn.name}
                </button>
              </li>
            ))}
          </ul>
        </div>

        <div className="flex flex-1 flex-col overflow-hidden">
          <CodeMirror
            key={mode}
            ref={editorRef}
            value={content}
            height="100%"
            theme={isDarkMode ? 'dark' : 'light'}
            extensions={extensions}
            onChange={(value) => setContent(value)}
            className="flex-1 overflow-auto text-sm"
          />
        </div>

        <div className="scrollbar-thin flex w-96 flex-col overflow-y-auto border-l border-border-soft p-3">
          <div className="mb-2 flex items-center justify-between">
            <h2 className="text-xs font-semibold uppercase text-text-tertiary">
              Preview {previewLoading && <span className="text-text-tertiary">(rendering…)</span>}
            </h2>
            {!renderError && htmlPreview && (
              <div className="flex items-center gap-1 rounded-md bg-surface-2 p-0.5">
                <button
                  type="button"
                  onClick={() => setPreviewTab('rendered')}
                  className={`rounded px-2 py-0.5 text-xs ${previewTab === 'rendered' ? 'bg-accent text-white' : 'text-text-secondary'}`}
                >
                  Rendered
                </button>
                <button
                  type="button"
                  onClick={() => setPreviewTab('raw')}
                  className={`rounded px-2 py-0.5 text-xs ${previewTab === 'raw' ? 'bg-accent text-white' : 'text-text-secondary'}`}
                >
                  Raw
                </button>
              </div>
            )}
          </div>
          {renderError ? (
            <div className="mb-2 rounded-md border border-danger bg-danger/10 p-2 text-xs text-danger">
              {renderError.line != null && (
                <div className="mb-1 font-semibold">
                  Line {renderError.line}
                  {renderError.column != null ? `, column ${renderError.column}` : ''}
                </div>
              )}
              <pre className="whitespace-pre-wrap">{renderError.message}</pre>
            </div>
          ) : htmlPreview && previewTab === 'rendered' ? (
            <iframe
              title="Rendered HTML preview"
              srcDoc={htmlPreview}
              sandbox=""
              className="mb-4 h-64 w-full rounded-md border border-border-soft bg-white"
            />
          ) : (
            <pre className="mb-4 whitespace-pre-wrap rounded-md border border-border-soft bg-surface p-2 text-xs text-text-primary">
              {preview}
            </pre>
          )}

          {!renderError && previewAttachments.length > 0 && (
            <div className="mb-4">
              <h2 className="mb-1 text-xs font-semibold uppercase text-text-tertiary">
                Attachments ({previewAttachments.length})
              </h2>
              <ul className="space-y-0.5">
                {previewAttachments.map((att, i) => (
                  <li
                    key={i}
                    className="flex items-center gap-1 truncate text-xs text-text-secondary"
                    title={att.path}
                  >
                    <Paperclip className="h-3 w-3 shrink-0" />
                    {att.filename || att.path}
                  </li>
                ))}
              </ul>
            </div>
          )}

          <div className="mb-2 mt-2 flex items-center justify-between">
            <h2 className="text-xs font-semibold uppercase text-text-tertiary">Recent sends</h2>
            {recentSends.length > 0 && (
              <button
                type="button"
                aria-label="Clear recent sends"
                title="Clear recent sends"
                onClick={() => setRecentSends([])}
                className="rounded-md p-1.5 text-text-tertiary transition-colors hover:bg-surface-2 hover:text-danger"
              >
                <Trash2 className="h-4 w-4" aria-hidden="true" />
              </button>
            )}
          </div>
          {recentSends.length === 0 ? (
            <p className="text-xs text-text-secondary">Nothing sent yet this session.</p>
          ) : (
            <ul className="space-y-1">
              {recentSends.map((s) => (
                <li
                  key={s.at}
                  className={`rounded px-2 py-1 text-xs ${s.ok ? 'text-success' : 'text-danger'}`}
                >
                  {s.ok ? `Sent to ${s.to.join(', ')}` : `Failed: ${s.message}`}
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      <Modal open={draftsModalOpen} onClose={() => setDraftsModalOpen(false)}>
        <h2 className="mb-4 text-lg font-semibold text-text-primary">Drafts</h2>
        <div className="mb-4 flex items-end gap-2">
          <div className="flex-1">
            <label htmlFor="draft-name" className="mb-1 block text-xs text-text-secondary">
              Draft name
            </label>
            <input
              id="draft-name"
              className="w-full rounded-md border border-border bg-bg px-2 py-1 text-text-primary"
              value={draftName}
              onChange={(e) => setDraftName(e.target.value)}
            />
          </div>
          <Button onClick={handleSaveDraft}>Save current</Button>
        </div>
        {Object.keys(drafts).length === 0 ? (
          <p className="text-sm text-text-secondary">No saved drafts.</p>
        ) : (
          <ul className="space-y-2">
            {Object.entries(drafts).map(([name, draft]) => (
              <li key={name} className="flex items-center justify-between gap-2">
                <span className="truncate text-sm text-text-primary">
                  {name} <span className="text-xs text-text-tertiary">({draft.format})</span>
                </span>
                <div className="flex gap-2">
                  <Button variant="secondary" onClick={() => handleLoadDraft(name)}>
                    Load
                  </Button>
                  <Button variant="ghost" onClick={() => deleteDraft(name)}>
                    Delete
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </Modal>
    </div>
  )
}
