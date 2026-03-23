import { afterEach, vi } from 'vitest'

const matchMediaMock = vi.fn().mockImplementation((query: string) => ({
  matches: false,
  media: query,
  onchange: null,
  addEventListener: vi.fn(),
  removeEventListener: vi.fn(),
  addListener: vi.fn(),
  removeListener: vi.fn(),
  dispatchEvent: vi.fn(),
}))

Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: matchMediaMock,
})

afterEach(() => {
  localStorage.clear()
  document.documentElement.dataset.theme = ''
  document.documentElement.className = ''
  document.documentElement.style.colorScheme = ''
})