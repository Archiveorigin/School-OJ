# Problem Package

A problem package is a ZIP file with a required `problem.yaml` at the archive root.

Example:

```yaml
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
```

Rules:

- Paths must be relative and cannot escape the ZIP root.
- Each case must reference an input and output file that exists in the ZIP.
- Supported submission languages are `c`, `cpp`, `python`, and `java`.
- Time, memory, and output limits are enforced by the worker sandbox.

Create a sample package:

```bash
./scripts/create_problem_zip.sh /tmp/a-plus-b.zip
```

Teachers do not need to hand-write this YAML for ordinary problems. In the web UI, open `题库`, click `上传题目包`, then choose:

- `上传现有 ZIP`: upload a prepared package.
- `表单创建题目`: fill in slug, title, statement, limits, and test cases. The API generates `problem.yaml`, test files, and the ZIP package automatically.
