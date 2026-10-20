import { importProblemMarkdown } from "../problems/importMarkdown";
import { emptyStatement, type StatementSections } from "./statement";

export type ImportAsset = {
  name: string;
  path: string;
  content_type: string;
  data: string;
};
export type ImportedProblem = {
  key: string;
  title: string;
  sections: StatementSections;
  assets: ImportAsset[];
  warnings: string[];
  time_limit_ms?: number;
  memory_limit_mb?: number;
  pages?: number[];
};
export function importEventMarkdown(source: string): ImportedProblem {
  const { title, draft, warnings } = importProblemMarkdown(source);
  const notes = [
    draft.hints,
    ...draft.samples.flatMap((s, i) =>
      s.explanation.trim()
        ? [`### 样例 ${i + 1} 说明\n\n${s.explanation}`]
        : [],
    ),
  ]
    .filter(Boolean)
    .join("\n\n");
  return {
    key: crypto.randomUUID(),
    title,
    sections: {
      description: draft.description,
      input_format: draft.input,
      output_format: draft.output,
      samples: draft.samples.map(({ input, output }) => ({ input, output })),
      notes,
    },
    assets: [],
    warnings,
  };
}
export function blankImportedProblem(
  title: string,
  pages?: number[],
): ImportedProblem {
  return {
    key: crypto.randomUUID(),
    title,
    sections: emptyStatement(),
    assets: [],
    warnings: [],
    pages,
  };
}
