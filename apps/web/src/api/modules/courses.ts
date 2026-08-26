import { client } from "../client";
import type { CourseSummary } from "../../features/exams/types";

export async function listCourses() {
  const { data } = await client.get<CourseSummary[]>("/courses");
  return Array.isArray(data) ? data : [];
}
