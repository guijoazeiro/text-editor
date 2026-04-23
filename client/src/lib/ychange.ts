import { Mark, mergeAttributes } from "@tiptap/core";

export const YChange = Mark.create({
  name: "ychange",

  addAttributes() {
    return {
      user: {
        default: null,
      },
      type: {
        default: null,
      },
      color: {
        default: null,
      },
    };
  },

  parseHTML() {
    return [
      {
        tag: "ychange",
      },
    ];
  },

  renderHTML({ HTMLAttributes }) {
    // HTMLAttributes.color is the ColorDef object: { light: string, dark: string }
    const color = HTMLAttributes.color?.dark || "#4b5563"; // Dark gray fallback
    return [
      "span",
      mergeAttributes(HTMLAttributes, {
        class: "remote-change",
        style: `color: ${color};`, // Set text color to the "dark" color
      }),
      0,
    ];
  },
});
