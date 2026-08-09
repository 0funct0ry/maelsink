import { fetchJson } from './fetchJson'
import type { UiInfo } from './apiTypes'

export function getInfo(): Promise<UiInfo> {
  return fetchJson<UiInfo>('/ui-api/v1/info')
}
