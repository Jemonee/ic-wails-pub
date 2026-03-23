
export interface IPromiseWithResolvers<T> {
    promise: Promise<T>
    resolve: T | PromiseLike<T>
    reject: (reason?: any) => void
}


const PromiseWithResolvers = <T>(): IPromiseWithResolvers<T> => {
    let reject, resolve
    const promise = new Promise<T>((res, rej) => {
        resolve = res
        reject = rej
    })

    return {
        promise,
        resolve,
        reject,
    } as IPromiseWithResolvers<T>
}


export {
    PromiseWithResolvers,
}