-- Remove participation data for previously deleted or missing exams.
DELETE FROM submissions WHERE exam_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM exams WHERE exams.id = submissions.exam_id AND exams.deleted_at IS NULL);
DELETE FROM exam_activities WHERE exam_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM exams WHERE exams.id = exam_activities.exam_id AND exams.deleted_at IS NULL);
DELETE FROM exam_attempts WHERE exam_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM exams WHERE exams.id = exam_attempts.exam_id AND exams.deleted_at IS NULL);
DELETE FROM exam_participants WHERE exam_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM exams WHERE exams.id = exam_participants.exam_id AND exams.deleted_at IS NULL);
DELETE FROM plagiarism_jobs WHERE exam_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM exams WHERE exams.id = plagiarism_jobs.exam_id AND exams.deleted_at IS NULL);
DELETE FROM exam_problems WHERE exam_id IS NOT NULL AND NOT EXISTS (SELECT 1 FROM exams WHERE exams.id = exam_problems.exam_id AND exams.deleted_at IS NULL);
DELETE FROM audit_logs WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '3 months';
-- Serialize submission writes against exam deletion to prevent late orphan rows.
CREATE FUNCTION guard_active_submission_exam() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.exam_id IS NOT NULL THEN
  PERFORM id FROM exams WHERE id=NEW.exam_id AND deleted_at IS NULL FOR SHARE;
  IF NOT FOUND THEN RAISE EXCEPTION 'exam no longer exists' USING ERRCODE='23503'; END IF;
 END IF;
 RETURN NEW;
END;
$$;
CREATE TRIGGER guard_active_submission_exam BEFORE INSERT OR UPDATE OF exam_id ON submissions
FOR EACH ROW EXECUTE FUNCTION guard_active_submission_exam();
