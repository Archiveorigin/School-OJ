import { generatedApi } from '../client'

export function getStatementAsset(problemID: number, assetPath: string, assetBase?: string, signal?: AbortSignal) {
  if (assetBase) {
    const event = assetBase.match(/^\/event-papers\/(exam|contest)\/(\d+)\/problems\/(\d+)\/assets$/)
    if (event && Number(event[3]) === problemID) return generatedApi.getEventProblemAsset<Blob>({ path: { kind: event[1], event_id: event[2], problem_id: event[3], asset_path: assetPath }, responseType: 'blob', signal })
    const exam = assetBase.match(/^\/exams\/(\d+)\/problems\/(\d+)\/assets$/)
    if (!exam || Number(exam[2]) !== problemID) throw new Error('Invalid exam asset scope')
    return generatedApi.getExamProblemAsset<Blob>({
      path: { id: exam[1], problem_id: exam[2], asset_path: assetPath }, responseType: 'blob', signal
    })
  }
  return generatedApi.getProblemAsset<Blob>({ path: { id: problemID, asset_path: assetPath }, responseType: 'blob', signal })
}
