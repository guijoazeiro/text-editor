'use client';

import { Editor } from '@tiptap/react';

interface Props {
  editor: Editor | null;
  disabled?: boolean;
}

interface ToolButton {
  label: string;
  title: string;
  action: () => void;
  isActive: () => boolean;
}

export default function EditorToolbar({ editor, disabled }: Props) {
  if (!editor) return null;

  const tools: ToolButton[] = [
    {
      label: 'B',
      title: 'Bold (Ctrl+B)',
      action: () => editor.chain().focus().toggleBold().run(),
      isActive: () => editor.isActive('bold'),
    },
    {
      label: 'I',
      title: 'Italic (Ctrl+I)',
      action: () => editor.chain().focus().toggleItalic().run(),
      isActive: () => editor.isActive('italic'),
    },
    {
      label: 'U',
      title: 'Underline (Ctrl+U)',
      action: () => editor.chain().focus().toggleUnderline().run(),
      isActive: () => editor.isActive('underline'),
    },
    {
      label: 'S',
      title: 'Strikethrough',
      action: () => editor.chain().focus().toggleStrike().run(),
      isActive: () => editor.isActive('strike'),
    },
  ];

  const headings: { level: 1 | 2 | 3; label: string }[] = [
    { level: 1, label: 'H1' },
    { level: 2, label: 'H2' },
    { level: 3, label: 'H3' },
  ];

  const lists = [
    {
      label: '≡',
      title: 'Bullet list',
      action: () => editor.chain().focus().toggleBulletList().run(),
      isActive: () => editor.isActive('bulletList'),
    },
    {
      label: '1.',
      title: 'Ordered list',
      action: () => editor.chain().focus().toggleOrderedList().run(),
      isActive: () => editor.isActive('orderedList'),
    },
  ];

  const extras = [
    {
      label: '"',
      title: 'Blockquote',
      action: () => editor.chain().focus().toggleBlockquote().run(),
      isActive: () => editor.isActive('blockquote'),
    },
    {
      label: '</>',
      title: 'Code block',
      action: () => editor.chain().focus().toggleCodeBlock().run(),
      isActive: () => editor.isActive('codeBlock'),
    },
    {
      label: '—',
      title: 'Horizontal rule',
      action: () => editor.chain().focus().setHorizontalRule().run(),
      isActive: () => false,
    },
  ];

  const history = [
    {
      label: '↩',
      title: 'Undo (Ctrl+Z)',
      action: () => editor.chain().focus().undo().run(),
      isActive: () => false,
    },
    {
      label: '↪',
      title: 'Redo (Ctrl+Shift+Z)',
      action: () => editor.chain().focus().redo().run(),
      isActive: () => false,
    },
  ];

  const btn = (
    key: string,
    { label, title, action, isActive }: ToolButton,
  ) => (
    <button
      key={key}
      title={title}
      onClick={action}
      disabled={disabled}
      className={`
        px-2.5 py-1.5 rounded text-sm font-medium transition select-none
        disabled:opacity-40 disabled:cursor-not-allowed
        ${isActive()
          ? 'bg-[#1479b0] text-white'
          : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'}
      `}
    >
      {label}
    </button>
  );

  const divider = (key: string) => (
    <span key={key} className="w-px h-5 bg-gray-200 mx-1" />
  );

  return (
    <div className="flex flex-wrap items-center gap-0.5 px-4 py-2 border-b border-gray-200 bg-gray-50">
      {tools.map((t, i) => btn(`fmt-${i}`, t))}
      {divider('d1')}
      {headings.map(({ level, label }) =>
        btn(`h${level}`, {
          label,
          title: `Heading ${level}`,
          action: () => editor.chain().focus().toggleHeading({ level }).run(),
          isActive: () => editor.isActive('heading', { level }),
        }),
      )}
      {divider('d2')}
      {lists.map((t, i) => btn(`list-${i}`, t))}
      {divider('d3')}
      {extras.map((t, i) => btn(`extra-${i}`, t))}
      {divider('d4')}
      {history.map((t, i) => btn(`hist-${i}`, t))}
    </div>
  );
}
