import { generatedApi } from '../client'

export function getStatementAsset(problemID: number, assetPath: string, assetBase?: string, signal?: AbortSignal) {
  if (assetBase) {
    const exam = assetBase.match(/^\/exams\/(\d+)\/problems\/(\d+)\/assets$/)
    if (!exam || Number(exam[2]) !== problemID) throw new Error('Invalid exam asset scope')
    return generatedApi.getExamProblemAsset<Blob>({
      path: { id: exam[1], problem_id: exam[2], asset_path: assetPath }, responseType: 'blob', signal
    })
  }
  return generatedApi.getProblemAsset<Blob>({ path: { id: problemID, asset_path: assetPath }, responseType: 'blob', signal })
}
