import {ChatLineSquare, CloseBold, FullScreen, HomeFilled, SemiSelect, Setting} from "@element-plus/icons-vue";
import {Window} from "@wailsio/runtime";
import {Component, markRaw, ref} from "vue";
import {DoubleClickTitleBar} from "../../bindings/ic-wails/internal/api/windowmanagerapi";

export const isMacOS = (): boolean => {
    return navigator.userAgent.toUpperCase().includes("MAC")
}

export const canShowCustomTitleBtn = (): boolean => {
    return !isMacOS()
}

export const headButtons = [
    {
        name: "设置",
        icon: Setting,
        click: () => {
            showSetting.value = !showSetting.value
        }
    },
    {
        name: "最小化",
        icon: SemiSelect,
        click: async () => {
            await minimise()
        }
    },
    {
        name: "最大化",
        icon: FullScreen,
        click: async () => {
           await maximize()
        }
    },
    {
        name: "关闭",
        icon: CloseBold,
        click: async () => {
            await Window.Close()
        }
    },
]

const minimise = async () => {
    try {
        await Window.Minimise()
        console.log('最小化调用成功')
    } catch (error) {
        console.error('最小化失败:', error)
    }
}

const maximize = async () => {
    await Window.IsMaximised() ? await Window.UnMaximise() : await Window.Maximise()
}

const fullScreen = async () => {
    await Window.IsFullscreen() ? await Window.UnFullscreen() : await Window.Fullscreen()
}

export const titleBarDoubleClick = () => {
    DoubleClickTitleBar().then(res => {
        if (res.success) {
            let action = res.data;
            if (action === "Maximize") {
                return maximize()
            }
            if (action === "Fill") {
                return fullScreen()
            }
            if (action === "Minimize") {
                return minimise()
            }
        } else {

        }
    })
}

export const  showSetting = ref<boolean>(false)

export interface IMenu  {
    id: number
    name: string
    title: string
    path: string
    icon?: Component
}

export const quickMenu:IMenu[] = [
    {
        id: 0,
        name: '主页',
        title: '主页',
        path: '/',
        icon: markRaw(HomeFilled)
    },
    {
        id: 5,
        name: 'AI 对话',
        title: 'AI 对话',
        path: '/chat',
        icon: markRaw(ChatLineSquare)
    },
]


const getQuickMenu = (): IMenu[] => {
    return quickMenu.map((item) => ({...item}))
}

export {
    getQuickMenu,
}