'use client';

import { useEffect, useRef } from 'react';
import { useEditor, EditorContent } from '@tiptap/react';
import StarterKit from '@tiptap/starter-kit';
import Collaboration from '@tiptap/extension-collaboration';
import CollaborationCursor from '@tiptap/extension-collaboration-cursor';
import Underline from '@tiptap/extension-underline';
import Placeholder from '@tiptap/extension-placeholder';
import * as Y from 'yjs';
import { YjsWebSocketProvider } from '@/lib/yjs-provider';

interface Props {
  ydoc: Y.Doc;
  provider: YjsWebSocketProvider;
  editable: boolean;
  userName: string;
  userColor: string;
  onReady?: () => void;
}

export default function TiptapEditor({
  ydoc,
  provider,
  editable,
  userName,
  userColor,
  onReady,
}: Props) {
  const onReadyCalled = useRef(false);

  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        history: false,
      }),
      Underline,
      Placeholder.configure({
        placeholder: 'Start typing… (CRDT-powered real-time collaboration)',
        emptyEditorClass: 'is-editor-empty',
      }),
      Collaboration.configure({
        document: ydoc,
        field: 'content',
      }),
      CollaborationCursor.configure({
        provider: provider,
        user: { name: userName, color: userColor },
      }),
    ],
    editable,
    editorProps: {
      attributes: {
        class: 'tiptap-editor',
        spellcheck: 'false',
      },
    },
    onCreate: () => {
      if (!onReadyCalled.current) {
        onReadyCalled.current = true;
        onReady?.();
      }
    },
    immediatelyRender: false,
  });

  useEffect(() => {
    if (editor && editor.isEditable !== editable) {
      editor.setEditable(editable);
    }
  }, [editor, editable]);

  useEffect(() => {
    return () => {
      editor?.destroy();
    };
  }, [editor]);

  return (
    <div className="tiptap-wrapper">
      <EditorContent editor={editor} />
    </div>
  );
}
