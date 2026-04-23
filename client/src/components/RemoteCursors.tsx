"use client";

import { useEffect, useRef, useCallback, RefObject } from "react";
import { RemoteUser } from "@/hooks/useYjsEditor";

interface Props {
  textareaRef: RefObject<HTMLTextAreaElement | null>;
  content: string;
  remoteUsers: RemoteUser[];
}

interface CaretCoords {
  top: number;
  left: number;
  lineHeight: number;
}

export default function RemoteCursors({
  textareaRef,
  content,
  remoteUsers,
}: Props) {
  const mirrorRef = useRef<HTMLDivElement | null>(null);
  const canvasRef = useRef<HTMLDivElement | null>(null);

  const syncMirror = useCallback(() => {
    const ta = textareaRef.current;
    const mirror = mirrorRef.current;
    if (!ta || !mirror) return;

    const style = window.getComputedStyle(ta);
    const props = [
      "fontFamily",
      "fontSize",
      "fontWeight",
      "lineHeight",
      "letterSpacing",
      "wordSpacing",
      "paddingTop",
      "paddingRight",
      "paddingBottom",
      "paddingLeft",
      "borderTopWidth",
      "borderRightWidth",
      "borderBottomWidth",
      "borderLeftWidth",
      "boxSizing",
      "whiteSpace",
      "wordWrap",
      "overflowWrap",
      "tabSize",
    ] as const;

    props.forEach((p) => {
      (mirror.style as unknown as Record<string, string>)[p] = style[p];
    });

    mirror.style.width = ta.offsetWidth + "px";
    mirror.style.height = ta.offsetHeight + "px";
    mirror.style.position = "absolute";
    mirror.style.top = "0";
    mirror.style.left = "0";
    mirror.style.visibility = "hidden";
    mirror.style.pointerEvents = "none";
    mirror.style.overflow = "hidden";
    mirror.style.whiteSpace = "pre-wrap";
    mirror.style.wordWrap = "break-word";
  }, [textareaRef]);

  const getCaretCoords = useCallback(
    (offset: number, text: string): CaretCoords | null => {
      const ta = textareaRef.current;
      const mirror = mirrorRef.current;
      if (!ta || !mirror) return null;

      syncMirror();

      const before = document.createTextNode(text.slice(0, offset));
      const marker = document.createElement("span");
      marker.textContent = "\u200b";
      const after = document.createTextNode(text.slice(offset));

      mirror.innerHTML = "";
      mirror.appendChild(before);
      mirror.appendChild(marker);
      mirror.appendChild(after);

      const taRect = ta.getBoundingClientRect();
      const markerRect = marker.getBoundingClientRect();

      const top = markerRect.top - taRect.top + ta.scrollTop;
      const left = markerRect.left - taRect.left;
      const lineHeight =
        parseFloat(window.getComputedStyle(ta).lineHeight) || 20;

      mirror.innerHTML = "";

      return { top, left, lineHeight };
    },
    [textareaRef, syncMirror],
  );

  const draw = useCallback(() => {
    const canvas = canvasRef.current;
    const ta = textareaRef.current;
    if (!canvas || !ta) return;

    canvas.innerHTML = "";
    canvas.style.position = "absolute";
    canvas.style.top = "0";
    canvas.style.left = "0";
    canvas.style.width = ta.offsetWidth + "px";
    canvas.style.height = ta.offsetHeight + "px";
    canvas.style.pointerEvents = "none";
    canvas.style.overflow = "hidden";

    remoteUsers.forEach(({ user, cursor }) => {
      if (!cursor) return;
      const { anchor, head } = cursor;
      const color = user.color;

      const caretOffset = head;
      const coords = getCaretCoords(caretOffset, content);
      if (!coords) return;

      if (anchor !== head) {
        const start = Math.min(anchor, head);
        const end = Math.max(anchor, head);

        const lines = content.split("\n");
        let charIndex = 0;
        lines.forEach((line) => {
          const lineStart = charIndex;
          const lineEnd = charIndex + line.length;
          charIndex += line.length + 1;

          const overlapStart = Math.max(start, lineStart);
          const overlapEnd = Math.min(end, lineEnd);
          if (overlapStart >= overlapEnd) return;

          const c1 = getCaretCoords(overlapStart, content);
          const c2 = getCaretCoords(overlapEnd, content);
          if (!c1 || !c2) return;

          const highlight = document.createElement("div");
          highlight.style.cssText = `
            position: absolute;
            top: ${c1.top}px;
            left: ${c1.left}px;
            width: ${Math.max(c2.left - c1.left, 4)}px;
            height: ${c1.lineHeight}px;
            background: ${color};
            opacity: 0.2;
            border-radius: 2px;
          `;
          canvas.appendChild(highlight);
        });
      }

      const caret = document.createElement("div");
      caret.style.cssText = `
        position: absolute;
        top: ${coords.top}px;
        left: ${coords.left}px;
        width: 2px;
        height: ${coords.lineHeight}px;
        background: ${color};
        border-radius: 1px;
        animation: yjsCursorBlink 1.2s step-end infinite;
      `;
      canvas.appendChild(caret);

      const badge = document.createElement("div");
      badge.textContent = user.name;
      badge.style.cssText = `
        position: absolute;
        top: ${coords.top - 22}px;
        left: ${coords.left}px;
        background: ${color};
        color: #fff;
        font-size: 11px;
        font-family: inherit;
        font-weight: 500;
        padding: 2px 6px;
        border-radius: 4px;
        white-space: nowrap;
        pointer-events: none;
        opacity: 0;
        animation: yjsBadgeFade 2.5s ease forwards;
        transform-origin: bottom left;
      `;
      canvas.appendChild(badge);
    });
  }, [remoteUsers, content, getCaretCoords, textareaRef]);

  useEffect(() => {
    const ta = textareaRef.current;
    if (!ta || !ta.parentElement) return;

    const mirror = document.createElement("div");
    mirrorRef.current = mirror;
    ta.parentElement.appendChild(mirror);

    return () => {
      mirror.remove();
      mirrorRef.current = null;
    };
  }, [textareaRef]);

  useEffect(() => {
    draw();
  }, [draw]);

  useEffect(() => {
    const id = "yjs-cursor-styles";
    if (document.getElementById(id)) return;
    const style = document.createElement("style");
    style.id = id;
    style.textContent = `
      @keyframes yjsCursorBlink {
        0%, 100% { opacity: 1; }
        50% { opacity: 0; }
      }
      @keyframes yjsBadgeFade {
        0% { opacity: 1; transform: scale(1); }
        60% { opacity: 1; }
        100% { opacity: 0; transform: scale(0.95); }
      }
    `;
    document.head.appendChild(style);
    return () => style.remove();
  }, []);

  return <div ref={canvasRef} aria-hidden="true" />;
}
