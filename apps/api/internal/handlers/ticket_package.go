package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"school-oj/apps/api/internal/authz"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/models"
	"school-oj/apps/api/internal/services"
)

type ticketPackageReview struct {
	TestMode          string                   `json:"test_mode"`
	Revision          int                      `json:"revision"`
	ExpectedVersionID *uint                    `json:"expected_version_id"`
	Title             string                   `json:"title"`
	Statement         string                   `json:"statement"`
	TimeLimitMS       int                      `json:"time_limit_ms"`
	MemoryLimitMB     int                      `json:"memory_limit_mb"`
	OutputLimitKB     int                      `json:"output_limit_kb"`
	Checker           services.CheckerManifest `json:"checker"`
	ReplaceTests      bool                     `json:"replace_tests"`
	AttachmentIDs     []uint                   `json:"attachment_ids"`
	Tags              []string                 `json:"tags"`
}

// Bound concurrent package assembly, which reuses the existing ZIP builder.
var ticketPackageSlots = make(chan struct{}, 2)

func acquireTicketPackage(ctx context.Context) (func(), error) {
	select {
	case ticketPackageSlots <- struct{}{}:
		return func() { <-ticketPackageSlots }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s Server) ticketReviewDefaults(ticket models.ProblemChangeTicket) (ticketPackageReview, error) {
	review := ticketPackageReview{Revision: ticket.Revision, Title: ticket.DraftTitle, Statement: ticket.DraftStatement, TimeLimitMS: 1000, MemoryLimitMB: 256, OutputLimitKB: 1024, Checker: services.CheckerManifest{Type: "exact"}, ReplaceTests: ticket.ProblemID == nil, AttachmentIDs: []uint{}}
	if ticket.ProblemID != nil {
		var p models.Problem
		if s.DB.Where("deleted_at IS NULL AND archived_at IS NULL").First(&p, *ticket.ProblemID).Error != nil {
			return review, errors.New("题目不存在或已归档，不能执行题库变更")
		}
		review.ExpectedVersionID = p.CurrentVersionID
		review.Tags = collectProblemTags([]models.Problem{p})
		review.Title, review.Statement = p.Title, p.Statement
		review.TimeLimitMS, review.MemoryLimitMB, review.OutputLimitKB = p.TimeLimitMS, p.MemoryLimitMB, p.OutputLimitKB
		b, _ := json.Marshal(p.Manifest)
		var manifest services.ProblemManifest
		_ = json.Unmarshal(b, &manifest)
		if manifest.Checker.Type != "" {
			review.Checker = manifest.Checker
		}
	} else {
		for _, a := range ticket.Attachments {
			ext := strings.ToLower(path.Ext(a.Name))
			if ext == ".zip" || ext == ".in" || ext == ".out" || strings.HasPrefix(a.ContentType, "image/") {
				review.AttachmentIDs = append(review.AttachmentIDs, a.ID)
			}
		}
	}
	if review.ReplaceTests {
		review.TestMode = "replace"
	} else {
		review.TestMode = "preserve"
	}
	return review, nil
}

func (s Server) previewProblemChangeTicket(c *gin.Context) {
	s.DB = s.DB.WithContext(c.Request.Context())
	ticket, _, ok := s.authorizedTicket(c)
	if !ok {
		return
	}
	user, _ := middleware.CurrentUser(c)
	if !authz.Has(user, authz.TicketAny) {
		c.JSON(403, gin.H{"error": "仅系统管理员可预检题库变更"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	var review ticketPackageReview
	if !bind(c, &review) {
		return
	}
	_, pkg, err := s.assembleTicketPackage(c, ticket, review)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	assets := map[string]string{}
	for _, a := range pkg.Assets {
		assets[a.Path] = "data:" + a.ContentType + ";base64," + base64.StdEncoding.EncodeToString(a.Body)
	}
	names := make([]string, 0, len(pkg.Manifest.Cases))
	for _, tc := range pkg.Manifest.Cases {
		names = append(names, tc.Name)
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(200, gin.H{"title": pkg.Manifest.Title, "statement": pkg.Manifest.Statement, "case_count": len(names), "case_names": names, "asset_count": len(pkg.Assets), "asset_urls": assets, "revision": ticket.Revision, "expected_version_id": review.ExpectedVersionID})
}

func (s Server) readTicketObject(ctx context.Context, object string, limit int64) ([]byte, error) {
	if s.MinIO == nil {
		return nil, errors.New("附件存储暂不可用")
	}
	src, err := s.MinIO.GetObject(ctx, s.Cfg.MinIOBucket, object, minio.GetObjectOptions{})
	if err != nil {
		return nil, errors.New("无法读取已保存的附件")
	}
	defer src.Close()
	return services.ReadLimited(ticketContextReader{ctx, src}, limit, "附件或题包")
}

type ticketMaterial struct {
	name string
	body []byte
}

func (s Server) assembleTicketPackage(c *gin.Context, ticket models.ProblemChangeTicket, review ticketPackageReview) ([]byte, services.ParsedProblemPackage, error) {
	empty := services.ParsedProblemPackage{}
	if ticket.Status != models.ProblemChangePending || ticket.Revision != review.Revision {
		return nil, empty, errors.New("工单已变更，请刷新后重新核对")
	}
	if ticket.Action != models.ProblemChangeCreate && ticket.Action != models.ProblemChangeReplace {
		return nil, empty, errors.New("此类工单不需要组装题包")
	}
	if ticket.Action == models.ProblemChangeReplace && ticket.ProblemID == nil {
		return nil, empty, errors.New("场次整体反馈应填写处置结果后完成")
	}
	if strings.TrimSpace(review.Title) == "" || len([]rune(review.Title)) > 200 || strings.TrimSpace(review.Statement) == "" || len(review.Statement) > 200000 {
		return nil, empty, errors.New("请填写有效的题目标题和题面")
	}
	if review.TimeLimitMS < 1 || review.TimeLimitMS > 60000 || review.MemoryLimitMB < 1 || review.MemoryLimitMB > 4096 || review.OutputLimitKB < 1 || review.OutputLimitKB > 65536 {
		return nil, empty, errors.New("判题限制须为：时间 1–60000 ms，内存 1–4096 MiB，输出 1–65536 KiB")
	}
	mode := review.TestMode
	if mode == "" {
		mode = "preserve"
		if review.ReplaceTests {
			mode = "replace"
		}
	}
	if mode != "preserve" && mode != "merge" && mode != "replace" {
		return nil, empty, errors.New("测试点处理方式无效")
	}
	if ticket.ProblemID == nil && mode != "replace" {
		return nil, empty, errors.New("新增题目必须提供至少一对测试点")
	}
	release, err := acquireTicketPackage(c.Request.Context())
	if err != nil {
		return nil, empty, err
	}
	defer release()
	var base []byte
	var manifest services.ProblemManifest
	if ticket.ProblemID != nil {
		var p models.Problem
		if s.DB.WithContext(c.Request.Context()).Where("deleted_at IS NULL AND archived_at IS NULL").First(&p, *ticket.ProblemID).Error != nil || review.ExpectedVersionID == nil || !sameTicketID(p.CurrentVersionID, review.ExpectedVersionID) {
			return nil, empty, errors.New("题目版本已变化或已归档，请刷新核对")
		}
		base, err = s.readTicketObject(c.Request.Context(), p.PackageObject, services.MaxProblemPackageSize)
		if err != nil {
			return nil, empty, err
		}
		pkg, err := services.ParseProblemPackage(base)
		if err != nil {
			return nil, empty, err
		}
		manifest = pkg.Manifest
	}
	materials, err := s.ticketMaterials(c.Request.Context(), ticket, review.AttachmentIDs)
	if err != nil {
		return nil, empty, err
	}
	cases, assets, aliases, err := ticketMaterialParts(materials, mode != "preserve")
	if err != nil {
		return nil, empty, err
	}
	if mode == "merge" {
		cases, err = services.MergeProblemCases(base, cases)
		if err != nil {
			return nil, empty, err
		}
	}
	for _, a := range manifest.Assets {
		aliases[a.Path] = a.Path
	}
	for _, a := range assets {
		aliases[a.Path] = a.Path
	}
	statement, err := rewriteTicketImages(review.Statement, aliases)
	if err != nil {
		return nil, empty, err
	}
	if base == nil {
		draft := services.ProblemPackageDraft{Title: review.Title, Statement: statement, TimeLimitMS: review.TimeLimitMS, MemoryLimitMB: review.MemoryLimitMB, OutputLimitKB: review.OutputLimitKB, Checker: review.Checker, Cases: cases}
		for _, a := range assets {
			draft.Assets = append(draft.Assets, services.ProblemAssetDraft{Name: a.Name, Path: a.Path, ContentType: a.ContentType, Data: base64.StdEncoding.EncodeToString(a.Body)})
		}
		return services.BuildProblemPackage(draft)
	}
	manifest.Title, manifest.Statement = review.Title, statement
	manifest.TimeLimitMS, manifest.MemoryLimitMB, manifest.OutputLimitKB, manifest.Checker = review.TimeLimitMS, review.MemoryLimitMB, review.OutputLimitKB, review.Checker
	return services.RebuildProblemPackage(base, manifest, cases, assets...)
}

func (s Server) ticketMaterials(ctx context.Context, ticket models.ProblemChangeTicket, ids []uint) ([]ticketMaterial, error) {
	if len(ids) > ticketFileLimit(ticket.Action) {
		return nil, errors.New("选择的附件过多")
	}
	byID := map[uint]models.ProblemTicketAttachment{}
	for _, a := range ticket.Attachments {
		byID[a.ID] = a
	}
	seen := map[uint]bool{}
	names := map[string]bool{}
	materials := []ticketMaterial{}
	var total int64
	appendFile := func(name string, body []byte) error {
		clean := path.Clean(strings.ReplaceAll(name, "\\", "/"))
		if strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") || strings.ContainsAny(clean, ":\x00") {
			return errors.New("附件包存在不安全的路径")
		}
		key := strings.ToLower(clean)
		if names[key] {
			return fmt.Errorf("所选材料包含重复文件 %s，请移除重复来源", clean)
		}
		names[key] = true
		total += int64(len(body))
		if total > ticketTotalAttachmentLimit {
			return errors.New("所选材料展开后总计不能超过 96 MiB")
		}
		materials = append(materials, ticketMaterial{clean, body})
		return nil
	}
	for _, id := range ids {
		a, ok := byID[id]
		if !ok || seen[id] {
			return nil, errors.New("选择了无效或重复的附件")
		}
		seen[id] = true
		body, err := s.readTicketObject(ctx, a.Object, problemTicketAttachmentLimit)
		if err != nil {
			return nil, fmt.Errorf("%s：%w", a.Name, err)
		}
		if !strings.EqualFold(path.Ext(a.Name), ".zip") {
			if err = appendFile(a.Name, body); err != nil {
				return nil, err
			}
			continue
		}
		z, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
		if err != nil {
			return nil, fmt.Errorf("%s 不是有效 ZIP", a.Name)
		}
		if len(z.File) > 1024 {
			return nil, errors.New("ZIP 条目超过 1024 个")
		}
		for _, file := range z.File {
			if err = ctx.Err(); err != nil {
				return nil, err
			}
			if file.FileInfo().IsDir() {
				continue
			}
			if !file.Mode().IsRegular() {
				return nil, errors.New("ZIP 不支持链接或特殊文件")
			}
			if file.UncompressedSize64 > uint64(problemTicketAttachmentLimit) || int64(file.UncompressedSize64) > ticketTotalAttachmentLimit-total {
				return nil, errors.New("ZIP 展开后超过大小限制")
			}
			reader, err := file.Open()
			if err != nil {
				return nil, err
			}
			data, err := services.ReadLimited(ticketContextReader{ctx, reader}, min(problemTicketAttachmentLimit, ticketTotalAttachmentLimit-total), "测试文件")
			reader.Close()
			if err != nil {
				return nil, err
			}
			if err = appendFile(file.Name, data); err != nil {
				return nil, err
			}
		}
	}
	sort.Slice(materials, func(i, j int) bool { return materials[i].name < materials[j].name })
	return materials, nil
}

func ticketMaterialParts(files []ticketMaterial, replaceTests bool) ([]services.ProblemCaseDraft, []services.ParsedProblemAsset, map[string]string, error) {
	type pair struct{ input, output *ticketMaterial }
	pairs := map[string]*pair{}
	assets := []services.ParsedProblemAsset{}
	aliases := map[string]string{}
	assetNames := map[string]bool{}
	for i := range files {
		f := &files[i]
		ext := strings.ToLower(path.Ext(f.name))
		if ext == ".in" || ext == ".out" {
			if !replaceTests {
				return nil, nil, nil, errors.New("已选择测试文件；请选择按名称合并或完整替换，或取消这些附件")
			}
			key := strings.ToLower(strings.TrimSuffix(f.name, path.Ext(f.name)))
			p := pairs[key]
			if p == nil {
				p = &pair{}
				pairs[key] = p
			}
			if ext == ".in" {
				p.input = f
			} else {
				p.output = f
			}
			continue
		}
		if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".webp" {
			assetPath := "assets/" + path.Base(f.name)
			if assetNames[strings.ToLower(assetPath)] {
				return nil, nil, nil, fmt.Errorf("图片文件名重复：%s", f.name)
			}
			assetNames[strings.ToLower(assetPath)] = true
			if len(f.body) > services.MaxProblemAssetSize {
				return nil, nil, nil, fmt.Errorf("题面图片 %s 超过 5 MiB，请压缩后补充", f.name)
			}
			contentType, err := services.ValidateProblemAsset(assetPath, f.body)
			if err != nil {
				return nil, nil, nil, err
			}
			assets = append(assets, services.ParsedProblemAsset{Name: path.Base(f.name), Path: assetPath, ContentType: contentType, Body: f.body, Size: int64(len(f.body))})
			for _, alias := range []string{f.name, path.Base(f.name), "attachments/" + path.Base(f.name), assetPath} {
				aliases[alias] = assetPath
			}
			continue
		}
		// A supplied package's manifest never silently overrides the reviewed draft.
		if path.Base(f.name) == "problem.yaml" {
			continue
		}
		return nil, nil, nil, fmt.Errorf("%s 不属于测试点或题面图片，请取消该参考附件", f.name)
	}
	var cases []services.ProblemCaseDraft
	if replaceTests {
		if len(pairs) == 0 {
			return nil, nil, nil, errors.New("缺少测试点：请选择包含同名 .in / .out 的 ZIP 或零散文件")
		}
		keys := make([]string, 0, len(pairs))
		for key := range pairs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			p := pairs[key]
			if p.input == nil || p.output == nil {
				return nil, nil, nil, fmt.Errorf("测试点 %s 缺少配对的 .in 或 .out", key)
			}
			cases = append(cases, services.ProblemCaseDraft{Name: key, Input: string(p.input.body), Output: string(p.output.body), Weight: 1})
		}
	}
	return cases, assets, aliases, nil
}

var ticketInlineImage = regexp.MustCompile(`!\[[^\]\n]*\]\(\s*(<[^>\n]+>|[^\s)]+)(?:\s+["'][^\n]*["'])?\s*\)`)
var ticketReferenceImage = regexp.MustCompile(`(?m)^(\s{0,3}\[[^\]\n]+\]:\s*)(<[^>\n]+>|\S+)`)

func rewriteTicketImages(statement string, aliases map[string]string) (string, error) {
	var failure error
	replace := func(match string, re *regexp.Regexp, index int) string {
		parts := re.FindStringSubmatchIndex(match)
		if len(parts) <= index*2+1 {
			return match
		}
		start, end := parts[index*2], parts[index*2+1]
		raw := strings.Trim(match[start:end], "<>")
		decoded, err := url.PathUnescape(raw)
		if err != nil {
			failure = errors.New("图片路径编码无效")
			return match
		}
		u, err := url.Parse(decoded)
		if err != nil {
			failure = errors.New("图片路径无效")
			return match
		}
		if u.Scheme == "http" || u.Scheme == "https" {
			return match
		}
		if u.IsAbs() || strings.HasPrefix(decoded, "//") {
			failure = fmt.Errorf("题面图片须使用已选择的附件资源：%s", raw)
			return match
		}
		key := strings.TrimPrefix(path.Clean(decoded), "./")
		target, ok := aliases[key]
		if !ok {
			target, ok = aliases[path.Base(key)]
		}
		if !ok {
			failure = fmt.Errorf("图片 %s 没有匹配的附件，请选择图片或修正题面路径", raw)
			return match
		}
		encoded := "assets/" + url.PathEscape(strings.TrimPrefix(target, "assets/"))
		return match[:start] + encoded + match[end:]
	}
	statement = mapTicketMarkdownText(statement, func(text string) string {
		return ticketInlineImage.ReplaceAllStringFunc(text, func(v string) string { return replace(v, ticketInlineImage, 1) })
	})
	// Only rewrite definitions used by images; ordinary Markdown links stay unchanged.
	statement = mapTicketMarkdownText(statement, func(text string) string {
		return ticketReferenceImage.ReplaceAllStringFunc(text, func(v string) string {
			end := strings.Index(v, "]")
			start := strings.Index(v, "[")
			if start < 0 || end < start {
				return v
			}
			label := v[start+1 : end]
			used := regexp.MustCompile(`(?i)!\[[^\]\n]*\]\[` + regexp.QuoteMeta(label) + `\]|!\[` + regexp.QuoteMeta(label) + `\](?:\[\])?`)
			if !used.MatchString(statement) {
				return v
			}
			return replace(v, ticketReferenceImage, 2)
		})
	})
	return statement, failure
}

// Code examples are literal content; attachment normalization must not edit them.
func mapTicketMarkdownText(source string, transform func(string) string) string {
	var out strings.Builder
	var fence byte
	var width int
	for _, line := range strings.SplitAfter(source, "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if len(trimmed) > 0 && (trimmed[0] == '`' || trimmed[0] == '~') {
			n := 0
			for n < len(trimmed) && trimmed[n] == trimmed[0] {
				n++
			}
			if n >= 3 && (fence == 0 || fence == trimmed[0] && n >= width && strings.TrimSpace(trimmed[n:]) == "") {
				if fence == 0 {
					fence, width = trimmed[0], n
				} else {
					fence = 0
				}
				out.WriteString(line)
				continue
			}
		}
		if fence != 0 || strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
			out.WriteString(line)
			continue
		}
		for len(line) > 0 {
			start := strings.IndexByte(line, '`')
			if start < 0 {
				out.WriteString(transform(line))
				break
			}
			n := 1
			for start+n < len(line) && line[start+n] == '`' {
				n++
			}
			end := strings.Index(line[start+n:], strings.Repeat("`", n))
			if end < 0 {
				out.WriteString(transform(line))
				break
			}
			end += start + n + n
			out.WriteString(transform(line[:start]))
			out.WriteString(line[start:end])
			line = line[end:]
		}
	}
	return out.String()
}
