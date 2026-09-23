import type { GachaRecord } from './uigf'
export interface Interval { pulls: number; lower_bound: boolean; uncertain: boolean }
export interface History {
  records: (GachaRecord & { interval?: Interval })[]; total: number; next_offset: number | null; revision: string
  analytics: { ranks: Record<string, number>; top_rate: number | null; complete_intervals: number; average_interval: number | null; from: string; to: string; months: { month: string; total: number; top: number; unknown: number }[]; items: { id: string; name: string; rank: string; count: number }[]; item_kinds: number }
}
export const poolNames: Record<string, string> = { '1': '常驻跃迁', '2': '始发跃迁', '11': '角色活动跃迁', '12': '光锥活动跃迁', '21': '联动角色跃迁', '22': '联动光锥跃迁' }
export function intervalText(value?: Interval): string { return !value ? '—' : value.uncertain ? '稀有度缺失，间隔不确定' : `${value.lower_bound ? '至少 ' : ''}${value.pulls} 抽` }
export function rarityName(rank: string): string { return rank === 'unknown' || !rank ? '未知稀有度' : `${rank} 星` }
