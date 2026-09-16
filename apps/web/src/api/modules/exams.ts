import { generatedApi } from "../client";
import type { CreateExamInput } from "../../features/exams/types";

export async function createExam(payload: CreateExamInput) {
  const { data } = await generatedApi.createExam<{ id: number }>({ body: payload });
  return data;
}
