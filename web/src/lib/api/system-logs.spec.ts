import { describe, expect, it } from 'vitest'
import { parseSystemLogEntry } from '@/lib/api/system-logs'

const validEntry = {
  id: 1,
  timestamp: 1_700_000_000,
  level: 'INFO',
  message: 'started',
  text: 'INFO started',
  source: 'main',
}

describe('parseSystemLogEntry', () => {
  it('返回合法帧', () => {
    expect(parseSystemLogEntry(JSON.stringify(validEntry))).toEqual(validEntry)
  })

  // 回归：解析失败曾经从 parser.feed() 一路抛进流的失败通道，让一条畸形日志
  // 被当成连接故障，整个日志流从此不再重连。
  it('非法 JSON 返回 null 而不是抛出', () => {
    expect(parseSystemLogEntry('{"id": 1')).toBeNull()
  })

  it('字段不合约时返回 null', () => {
    expect(
      parseSystemLogEntry(JSON.stringify({ ...validEntry, level: '' })),
    ).toBeNull()
  })
})
