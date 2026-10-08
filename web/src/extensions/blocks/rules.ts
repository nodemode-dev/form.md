import { InputRule, inputRules, textblockTypeInputRule, wrappingInputRule } from "prosemirror-inputrules";
import type { MarkType, NodeType } from "prosemirror-model";
import { type Plugin, TextSelection } from "prosemirror-state";
import { newBlockId, schema } from "../../engine/doc/schema";
import { normaliseLanguage } from "./languages";
import { mathRules } from "./math";

const N = schema.nodes as Record<string, NodeType>;
const M = schema.marks as Record<string, MarkType>;

function markRule(re: RegExp, mark: MarkType, open: number): InputRule {
  return new InputRule(re, (state, match, start, end) => {
    const full = match[0];
    const inner = match[1] ?? "";
    const offset = full.length - inner.length - open * 2;
    const textStart = start + offset + open;
    const textEnd = textStart + inner.length;
    const $start = state.doc.resolve(start);
    if ($start.parent.type.spec.code) return null;
    if (state.doc.rangeHasMark(textStart, textEnd, M.code as MarkType) && mark !== M.code) return null;
    const tr = state.tr;
    tr.delete(textEnd, end);
    tr.delete(start + offset, textStart);
    tr.addMark(start + offset, start + offset + inner.length, mark.create());
    tr.removeStoredMark(mark);
    return tr;
  });
}

export function rulesPlugin(): Plugin {
  const rules: InputRule[] = [
    textblockTypeInputRule(/^\s{0,3}(#{1,6})\s$/, N.heading as NodeType, (m) => ({ level: (m[1] ?? "#").length })),
    wrappingInputRule(/^\s*([-+*])\s$/, N.bullet_list as NodeType),
    wrappingInputRule(
      /^(\d+)[.)]\s$/,
      N.ordered_list as NodeType,
      (m) => ({ start: Number(m[1]) || 1 }),
      (m, node) => node.childCount + Number(node.attrs.start) === Number(m[1]),
    ),
    new InputRule(/^\s*\[([ xX]?)\]\s$/, (state, match, start, end) => {
      const $start = state.doc.resolve(start);
      if ($start.parent.type !== N.paragraph) return null;
      const checked = (match[1] ?? "").toLowerCase() === "x";
      const item = (N.task_item as NodeType).create({ id: newBlockId(), checked }, [(N.paragraph as NodeType).create({ id: newBlockId() })]);
      const list = (N.task_list as NodeType).create({ id: newBlockId() }, [item]);
      const from = $start.before();
      const to = $start.after();
      if ($start.parent.content.size !== end - start) return null;
      const tr = state.tr.replaceWith(from, to, list);
      return tr.setSelection(TextSelection.create(tr.doc, from + 3)).scrollIntoView();
    }),
    wrappingInputRule(/^\s*>\s$/, N.blockquote as NodeType),
    textblockTypeInputRule(/^```([\w+#-]*)\s$/, N.code_block as NodeType, (m) => ({ language: normaliseLanguage(m[1] ?? "") })),
    new InputRule(/^(?:---|\*\*\*|___)$/, (state, _m, start) => {
      const $start = state.doc.resolve(start);
      if ($start.parent.type !== N.paragraph || $start.depth < 1) return null;
      const from = $start.before();
      const to = $start.after();
      const tr = state.tr.replaceWith(from, to, [(N.horizontal_rule as NodeType).create({ id: newBlockId() }), (N.paragraph as NodeType).create({ id: newBlockId() })]);
      return tr.scrollIntoView();
    }),
    markRule(/\*\*([^*\s](?:[^*]*[^*\s])?)\*\*$/, M.strong as MarkType, 2),
    markRule(/(?:^|[^*\w])\*([^*\s](?:[^*]*[^*\s])?)\*$/, M.em as MarkType, 1),
    markRule(/(?:^|[^_\w])_([^_\s](?:[^_]*[^_\s])?)_$/, M.em as MarkType, 1),
    markRule(/~~([^~\s](?:[^~]*[^~\s])?)~~$/, M.strike as MarkType, 2),
    markRule(/`([^`]+)`$/, M.code as MarkType, 1),
    ...mathRules(),
  ];
  return inputRules({ rules });
}
