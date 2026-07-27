import { describe, expect, it } from 'vitest'
import { decodeXimoAIHomeHTMLCover, resolveXimoAIHomeCoverType } from '@/utils/ximoaiHomeCover'

describe('resolveXimoAIHomeCoverType', () => {
  it('keeps image and GIF covers in image mode', () => {
    expect(resolveXimoAIHomeCoverType('data:image/png;base64,abc')).toBe('image')
    expect(resolveXimoAIHomeCoverType('data:image/gif;base64,abc')).toBe('image')
  })

  it('recognizes uploaded video and HTML data URLs', () => {
    expect(resolveXimoAIHomeCoverType('data:video/mp4;base64,abc')).toBe('video')
    expect(resolveXimoAIHomeCoverType('data:text/html;base64,abc')).toBe('html')
  })

  it('recognizes media URLs by their file extension', () => {
    expect(resolveXimoAIHomeCoverType('https://cdn.example.com/intro.webm')).toBe('video')
    expect(resolveXimoAIHomeCoverType('https://cdn.example.com/intro.html?theme=dark')).toBe('html')
    expect(resolveXimoAIHomeCoverType('https://cdn.example.com/cover.png')).toBe('image')
  })

  it('decodes uploaded HTML for sandboxed srcdoc rendering', () => {
    expect(decodeXimoAIHomeHTMLCover('data:text/html;base64,PGgxPkhlbGxvPC9oMT4=')).toBe('<h1>Hello</h1>')
    expect(decodeXimoAIHomeHTMLCover('data:image/png;base64,AAAA')).toBe('')
  })
})
