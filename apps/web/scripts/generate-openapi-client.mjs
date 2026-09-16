import { readFile, writeFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const specPath = resolve(here, '../../api/openapi/openapi.json')
const outputPath = resolve(here, '../src/api/generated/client.ts')
const checkOnly = process.argv.includes('--check')
const spec = JSON.parse(await readFile(specPath, 'utf8'))

if (!String(spec.openapi || '').startsWith('3.')) {
  throw new Error('Only OpenAPI 3.x documents are supported')
}

function refName(ref) {
  return String(ref).split('/').at(-1)
}

function schemaType(schema = {}) {
  if (schema.nullable) return `(${schemaType({ ...schema, nullable: false })}) | null`
  if (schema.$ref) return `components['schemas'][${JSON.stringify(refName(schema.$ref))}]`
  if (schema.allOf) return schema.allOf.map(schemaType).join(' & ')
  if (schema.oneOf) return schema.oneOf.map(schemaType).join(' | ')
  if (schema.anyOf) return schema.anyOf.map(schemaType).join(' | ')
  if (schema.enum) return schema.enum.map((value) => JSON.stringify(value)).join(' | ')
  if (schema.type === 'array') return `Array<${schemaType(schema.items || {})}>`
  if (schema.type === 'integer' || schema.type === 'number') return 'number'
  if (schema.type === 'boolean') return 'boolean'
  if (schema.type === 'string') return 'string'
  if (schema.type === 'object' || schema.properties || schema.additionalProperties) {
    const required = new Set(schema.required || [])
    const fields = Object.entries(schema.properties || {}).map(
      ([name, value]) => `    ${JSON.stringify(name)}${required.has(name) ? '' : '?'}: ${schemaType(value)}`,
    )
    if (schema.additionalProperties) {
      fields.push(
        `    [key: string]: ${schema.additionalProperties === true ? 'unknown' : schemaType(schema.additionalProperties)}`,
      )
    }
    return fields.length ? `{\n${fields.join('\n')}\n  }` : 'Record<string, unknown>'
  }
  return 'unknown'
}

function responseSchema(operation) {
  const responses = operation.responses || {}
  for (const status of ['200', '201', '202', '204', 'default']) {
    const response = responses[status]
    const schema = response?.content?.['application/json']?.schema
    if (schema) return schemaType(schema)
  }
  return 'unknown'
}

function requestSchema(operation) {
  const content = operation.requestBody?.content || {}
  const schema = content['application/json']?.schema || content['multipart/form-data']?.schema
  return schema ? schemaType(schema) : 'never'
}

const operations = []
for (const [path, pathItem] of Object.entries(spec.paths || {})) {
  for (const method of ['get', 'post', 'put', 'patch', 'delete']) {
    const operation = pathItem[method]
    if (!operation) continue
    if (!operation.operationId) throw new Error(`${method.toUpperCase()} ${path} has no operationId`)
    operations.push({
      id: operation.operationId,
      method: method.toUpperCase(),
      path,
      segments: (operation.parameters || []).filter(parameter => parameter['x-path-segments']).map(parameter => parameter.name),
      body: requestSchema(operation),
      response: responseSchema(operation),
    })
  }
}

const duplicate = operations.find(
  (item, index) => operations.findIndex((candidate) => candidate.id === item.id) !== index,
)
if (duplicate) throw new Error(`Duplicate operationId: ${duplicate.id}`)

const schemaLines = Object.entries(spec.components?.schemas || {}).map(
  ([name, schema]) => `    ${JSON.stringify(name)}: ${schemaType(schema)}`,
)
const definitions = operations
  .map(
    ({ id, method, path, segments }) =>
      `  ${JSON.stringify(id)}: { method: ${JSON.stringify(method)}, path: ${JSON.stringify(path)}, segments: ${JSON.stringify(segments)} },`,
  )
  .join('\n')
const methods = operations
  .map(
    ({ id, body, response }) =>
      `  ${JSON.stringify(id)}: <TResponse = ${response}>(options: ApiRequestOptions<${body}> = {}) =>\n    requestOperation<TResponse, ${body}>(${JSON.stringify(id)}, options),`,
  )
  .join('\n')

const output = `/* eslint-disable */
// 此文件由 scripts/generate-openapi-client.mjs 根据 apps/api/openapi/openapi.json 生成。
// 请勿手工修改；修改契约后运行 npm run api:generate。

import type { AxiosRequestConfig, AxiosResponse } from 'axios'
import { http } from '../http'

export interface components {
  schemas: {
${schemaLines.join('\n')}
  }
}

export const apiOperations = {
${definitions}
} as const

export type ApiOperationId = keyof typeof apiOperations
export type ApiPathValue = string | number | boolean
export type ApiQueryValue = ApiPathValue | null | undefined | ApiPathValue[]

export interface ApiRequestOptions<TBody = unknown>
  extends Omit<AxiosRequestConfig, 'method' | 'url' | 'params' | 'data'> {
  path?: Record<string, ApiPathValue>
  query?: Record<string, ApiQueryValue>
  body?: TBody | FormData | Blob
}

function resolvePath(template: string, values: Record<string, ApiPathValue> = {}, segments: readonly string[] = []) {
  return template.replace(/\\{([^}]+)\\}/g, (_, key: string) => {
    const value = values[key]
    if (value === undefined || value === null || value === '') {
      throw new Error(\`Missing OpenAPI path parameter: \${key}\`)
    }
    if (segments.includes(key)) {
      const parts = String(value).split('/')
      if (parts.some(part => part === '..' || part === '.' || !part)) throw new Error('Invalid asset path')
      return parts.map(part => encodeURIComponent(part)).join('/')
    }
    return encodeURIComponent(String(value))
  })
}

export async function requestOperation<TResponse = unknown, TBody = unknown>(
  operationId: ApiOperationId,
  options: ApiRequestOptions<TBody> = {},
): Promise<AxiosResponse<TResponse>> {
  const definition = apiOperations[operationId]
  const { path, query, body, ...config } = options
  return http.request<TResponse>({
    ...config,
    method: definition.method,
    url: resolvePath(definition.path, path, definition.segments),
    params: query,
    data: body,
  })
}

export const generatedApi = {
${methods}
}

// 兼容现有页面的 Axios 调用；新代码优先使用 generatedApi 的 operationId 方法。
export const client = http
`

if (checkOnly) {
  let current = ''
  try {
    current = await readFile(outputPath, 'utf8')
  } catch {}
  if (current.replace(/\r\n/g, '\n') !== output.replace(/\r\n/g, '\n')) {
    console.error('OpenAPI Client 已过期，请运行 npm run api:generate')
    process.exit(1)
  }
  console.log(`OpenAPI Client 校验通过：${operations.length} 个操作`)
} else {
  await writeFile(outputPath, output, 'utf8')
  console.log(`已生成 ${outputPath}：${operations.length} 个操作`)
}
