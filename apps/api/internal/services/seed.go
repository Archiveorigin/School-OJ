package services

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"school-oj/apps/api/internal/config"
	"school-oj/apps/api/internal/models"

	"github.com/minio/minio-go/v7"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func EnsureBucket(ctx context.Context, client *minio.Client, cfg config.Config) error {
	var lastErr error
	for i := 0; i < 30; i++ {
		exists, err := client.BucketExists(ctx, cfg.MinIOBucket)
		if err == nil {
			if exists {
				return nil
			}
			return client.MakeBucket(ctx, cfg.MinIOBucket, minio.MakeBucketOptions{})
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return lastErr
}

func Seed(ctx context.Context, db *gorm.DB, client *minio.Client, cfg config.Config) error {
	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	admin := user("admin@school.local", "系统管理员", models.RoleAdmin, "")
	teacher := user("teacher@school.local", "任课教师", models.RoleTeacher, "")
	student := user("student@school.local", "学生账号", models.RoleStudent, "S20260001")
	if err := db.Create(&admin).Error; err != nil {
		return err
	}
	if err := db.Create(&teacher).Error; err != nil {
		return err
	}
	if err := db.Create(&student).Error; err != nil {
		return err
	}

	course := models.Course{Code: "CS101-2026", Name: "程序设计基础", Term: "2026 春", TeacherID: teacher.ID, Description: "课程编程练习与自动评测"}
	if err := db.Create(&course).Error; err != nil {
		return err
	}
	class := models.Class{CourseID: course.ID, Name: "计科一班"}
	if err := db.Create(&class).Error; err != nil {
		return err
	}
	_ = db.Create(&models.CourseMembership{CourseID: course.ID, UserID: teacher.ID, Role: models.RoleTeacher}).Error
	_ = db.Create(&models.CourseMembership{CourseID: course.ID, UserID: student.ID, Role: models.RoleStudent}).Error
	_ = db.Create(&models.ClassMembership{ClassID: class.ID, UserID: student.ID}).Error

	pkgBytes, err := samplePackage()
	if err != nil {
		return err
	}
	sum := sha256.Sum256(pkgBytes)
	object := "problems/a-plus-b/sample.zip"
	if _, err := client.PutObject(ctx, cfg.MinIOBucket, object, bytes.NewReader(pkgBytes), int64(len(pkgBytes)), minio.PutObjectOptions{ContentType: "application/zip"}); err != nil {
		return err
	}
	problem := models.Problem{
		OwnerID:         teacher.ID,
		Slug:            "a-plus-b",
		Title:           "A + B Problem",
		Statement:       "输入两个整数 a 和 b，输出它们的和。",
		Tags:            datatypes.JSONMap{"tags": []string{"入门", "数学"}},
		TimeLimitMS:     1000,
		MemoryLimitMB:   128,
		OutputLimitKB:   64,
		PackageObject:   object,
		PackageChecksum: hex.EncodeToString(sum[:]),
		Manifest: datatypes.JSONMap{
			"slug": "a-plus-b",
			"cases": []map[string]any{
				{"name": "sample1", "input": "tests/01.in", "output": "tests/01.out", "weight": 50},
				{"name": "sample2", "input": "tests/02.in", "output": "tests/02.out", "weight": 50},
			},
		},
	}
	if err := db.Create(&problem).Error; err != nil {
		return err
	}
	now := time.Now()
	due := now.Add(7 * 24 * time.Hour)
	assignment := models.Assignment{CourseID: course.ID, Title: "第一次作业", Description: "基础输入输出练习", StartsAt: &now, DueAt: &due}
	if err := db.Create(&assignment).Error; err != nil {
		return err
	}
	_ = db.Create(&models.AssignmentProblem{AssignmentID: assignment.ID, ProblemID: problem.ID, Score: 100}).Error
	examEnd := now.Add(2 * time.Hour)
	exam := models.Exam{CourseID: course.ID, Title: "期中考试", Description: "自动判题考试流程", StartsAt: &now, EndsAt: &examEnd}
	if err := db.Create(&exam).Error; err != nil {
		return err
	}
	_ = db.Create(&models.ExamProblem{ExamID: exam.ID, ProblemID: problem.ID, Score: 100}).Error
	return nil
}

func user(email, name string, role models.Role, studentNo string) models.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	return models.User{Email: email, Name: name, Role: role, StudentNo: studentNo, PasswordHash: string(hash), EmailVerified: true}
}

func samplePackage() ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"problem.yaml": strings.TrimSpace(`
slug: a-plus-b
title: A + B Problem
statement: 输入两个整数 a 和 b，输出它们的和。
time_limit_ms: 1000
memory_limit_mb: 128
output_limit_kb: 64
cases:
  - name: sample1
    input: tests/01.in
    output: tests/01.out
    weight: 50
  - name: sample2
    input: tests/02.in
    output: tests/02.out
    weight: 50
`) + "\n",
		"tests/01.in":  "1 2\n",
		"tests/01.out": "3\n",
		"tests/02.in":  "100 250\n",
		"tests/02.out": "350\n",
	}
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := fmt.Fprint(w, body); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
