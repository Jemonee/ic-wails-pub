import {computed, ref} from "vue"

export type ThemeMode = 'auto' | 'light' | 'dark'
type ResolvedTheme = 'light' | 'dark'

const STORAGE_KEY = 'ic-wails-theme-mode'
const mediaQuery = typeof window !== 'undefined'
    ? window.matchMedia('(prefers-color-scheme: dark)')
    : null

export const themeMode = ref<ThemeMode>('auto')
const resolvedTheme = ref<ResolvedTheme>('light')

export const currentTheme = computed(() => resolvedTheme.value)

function getSystemTheme(): ResolvedTheme {
    if (mediaQuery?.matches) {
        return 'dark'
    }
    return 'light'
}

function applyResolvedTheme(theme: ResolvedTheme) {
    resolvedTheme.value = theme
    const root = document.documentElement
    root.dataset.theme = theme
    root.classList.toggle('dark', theme === 'dark')
    root.style.colorScheme = theme
}

function syncTheme(mode: ThemeMode) {
    if (mode === 'auto') {
        applyResolvedTheme(getSystemTheme())
        return
    }
    applyResolvedTheme(mode)
}

function handleSystemThemeChange() {
    if (themeMode.value === 'auto') {
        applyResolvedTheme(getSystemTheme())
    }
}

export function setThemeMode(mode: ThemeMode) {
    themeMode.value = mode
    localStorage.setItem(STORAGE_KEY, mode)
    syncTheme(mode)
}

export function initializeTheme() {
    const savedMode = localStorage.getItem(STORAGE_KEY) as ThemeMode | null
    themeMode.value = savedMode || 'auto'
    syncTheme(themeMode.value)

    if (mediaQuery) {
        if (typeof mediaQuery.addEventListener === 'function') {
            mediaQuery.addEventListener('change', handleSystemThemeChange)
        } else if (typeof mediaQuery.addListener === 'function') {
            mediaQuery.addListener(handleSystemThemeChange)
        }
    }
}