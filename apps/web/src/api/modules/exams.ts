import { client } from "../client";
import type { CreateExamInput } from "../../features/exams/types";

export async function createExam(payload: CreateExamInput) {
  const { data } = await client.post<{ id: number }>("/exams", payload);
  return data;
}
