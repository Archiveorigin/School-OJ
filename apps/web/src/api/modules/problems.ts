import { client, type PreparedProblem, type Problem } from "../client";

export async function listProblems() {
  const { data } = await client.get<Problem[]>("/problems");
  return Array.isArray(data) ? data : [];
}

export async function listPreparedProblems() {
  const { data } = await client.get<PreparedProblem[]>("/prepared-problems");
  return Array.isArray(data) ? data : [];
}

export async function createProblem(
  payload: FormData | Record<string, unknown>,
) {
  const { data } = await client.post<Problem>("/problems", payload, {
    timeout: payload instanceof FormData ? 120_000 : 30_000,
  });
  return data;
}

export async function parseProblemMarkdown(file: File) {
  const payload = new FormData();
  payload.append("file", file);
  const { data } = await client.post<{ problems?: any[]; warnings?: string[] }>(
    "/problems/parse-markdown",
    payload,
    { timeout: 60_000 },
  );
  return {
    problems: data.problems || [],
    warnings: data.warnings || [],
  };
}
