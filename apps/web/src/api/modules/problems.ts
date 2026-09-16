import { generatedApi, type PreparedProblem, type Problem } from "../client";

export async function listProblems() {
  const { data } = await generatedApi.listProblems();
  return Array.isArray(data) ? data : [];
}

export async function listPreparedProblems() {
  const { data } = await generatedApi.listPreparedProblems();
  return Array.isArray(data) ? data : [];
}

export async function createProblem(
  payload: FormData | Record<string, unknown>,
) {
  const { data } = await generatedApi.problemChangeTicketOnly<Problem>({ timeout: payload instanceof FormData ? 120_000 : 30_000, body: payload });
  return data;
}

export async function parseProblemMarkdown(file: File) {
  const payload = new FormData();
  payload.append("file", file);
  const { data } = await generatedApi.parseMarkdownBatch<{ problems?: any[]; warnings?: string[] }>({ timeout: 60_000, body: payload });
  return {
    problems: data.problems || [],
    warnings: data.warnings || [],
  };
}
