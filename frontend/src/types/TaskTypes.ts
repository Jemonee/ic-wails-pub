export interface ITaskCategory {
    // 分类名称
    name: string
    // 分类编码(全局唯一)
    key: string
}

export interface ITask {
    key: string
    lastKey?: string
    categoryKey: string
    // 任务内容
    content: string
    // 任务项
    item: ITaskItem[]
    // 任务进度
    progress: number
    // 获得时间
    gainTime: number
    // 开始时间
    startTime?: number
    // 结束时间
    endTime?: number
    planStartTime?: number
    planEndTime?: number
    // 是否完成
    done: boolean
}

export interface ITaskItem {
    // 任务项内容
    content: string
    // 任务项进度
    progress: number
    // 开始时间
    startTime?: number
    // 结束时间
    endTime?: number
    planStartTime?: number
    planEndTime?: number
    // 是否完成
    done: boolean
}

export {

}