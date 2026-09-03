# 题目包规范

## 1. 使用方式

题目新增、替换、归档及测试点修改统一通过“工单修改”完成。出题人创建工单时必须上传一份修改后的正确内容作为参考附件；管理员审核时重新上传最终完整 ZIP 题包并执行覆盖操作。

旧版直接新增、编辑、删除和测试点替换接口不会再直接改变题目。公共题库的现行内容来自管理员执行工单后生成的不可变题目版本。

## 2. 目录结构

ZIP 根目录必须包含 `problem.yaml`。推荐结构：

```text
problem.zip
├── problem.yaml
├── tests
│   ├── 01.in
│   ├── 01.out
│   ├── 02.in
│   └── 02.out
└── assets
    └── diagram.png
```

示例清单：

```yaml
slug: a-plus-b
title: A + B Problem
statement: 输入两个整数 a 和 b，输出它们的和。
time_limit_ms: 1000
memory_limit_mb: 128
output_limit_kb: 64
checker:
  type: float
  absolute_tolerance: 0.000001
  relative_tolerance: 0.000001
cases:
  - name: sample1
    input: tests/01.in
    output: tests/01.out
    weight: 50
  - name: sample2
    input: tests/02.in
    output: tests/02.out
    weight: 50
```

## 3. 校验规则

- 所有路径必须为相对路径，不得逃逸 ZIP 根目录。
- 只允许 `problem.yaml`、`tests/*.in`、`tests/*.out` 和 `assets/` 下受支持的图片。
- 每个测试点必须引用实际存在的 `.in` 和 `.out` 文件。
- 至少包含一个测试点，所有权重总和应与题目满分设计一致。
- 时间、内存和输出限制必须为有效正数，执行时由 Worker 沙箱强制实施。
- 压缩包、单文件、资源图片和测试数据都有大小限制；超限会返回明确错误，不会静默截断。
- 系统支持的提交语言为 `c`、`cpp`、`python` 和 `java`。

## 4. 答案比较器

`checker.type` 支持：

| 类型 | 行为 |
| --- | --- |
| `exact` | 默认值；统一换行符并忽略每行末尾空白后进行精确比较 |
| `tokens` | 按非空白令牌比较，忽略空白布局差异 |
| `float` | 数字令牌使用绝对和相对误差，非数字令牌仍须完全一致 |

浮点比较的误差必须为有限、非负且不大于 `1`。两个误差都省略时，各自默认为 `1e-6`。系统不支持可执行 checker 脚本。

## 5. 图片与题面资源

题面图片放在 `assets/` 下，并在题面中使用相对资源地址。API 会通过受控资源接口读取当前上下文绑定版本的文件，不应引用本机路径、外部临时地址或测试目录截图。

## 6. 生成示例题包

仓库脚本可以生成一个基础示例：

```bash
./scripts/create_problem_zip.sh /tmp/a-plus-b.zip
```

该脚本只用于开发和格式验证。正式工单附件和管理员最终题包仍需检查题面、限制、测试数据、权重和比较器配置。
