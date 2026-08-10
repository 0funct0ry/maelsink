import { fetchJson } from './fetchJson'
import type { ConfigEntry, UiInfo } from './apiTypes'

export function getInfo(): Promise<UiInfo> {
  return fetchJson<UiInfo>('/ui-api/v1/info')
}

export function getConfig(): Promise<ConfigEntry[]> {
  return fetchJson<ConfigEntry[]>('/ui-api/v1/config')
}
