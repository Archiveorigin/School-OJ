import { generatedApi } from "../client";
import type { CourseSummary } from "../../features/exams/types";

export async function listCourses() {
  const { data } = await generatedApi.listCourses<CourseSummary[]>();
  return Array.isArray(data) ? data : [];
}
