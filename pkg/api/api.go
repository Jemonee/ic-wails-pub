package api

import (
	"fmt"
	"net/http"

	"ic-wails/pkg/common"

	"github.com/gin-gonic/gin"
)

// IApi 业务 API 控制器接口，实现此接口在 Register 中注册自身路由
type IApi interface {
	Register(router *gin.RouterGroup)
}

// BaseApi API 层基类，业务 API 内嵌此结构体以复用通用能力。
type BaseApi struct{}

// DeferPanicHandler 统一 panic 恢复，将 ServicePanic/error 转为 R[T] 失败响应并写回客户端。
// 使用方式：在 handler 首行 defer a.DeferPanicHandler(c)
func (a *BaseApi) DeferPanicHandler(c *gin.Context) {
	if r := recover(); r != nil {
		var result common.R[any]
		switch err := r.(type) {
		case common.ServicePanic:
			result = common.F[any](err.Code, err.Msg)
		case error:
			result = common.F[any](500, "操作产生异常错误："+err.Error())
		default:
			result = common.F[any](500, "发生未知异常："+fmt.Sprintf("%v", r))
		}
		c.JSON(http.StatusOK, result)
	}
}

// DeferWailsPanicHandler 统一 panic 恢复（Wails 绑定专用），将 ServicePanic/error 转为 R[T] 失败响应。
// 使用方式：在 Wails 绑定方法首行 defer pkgapi.DeferWailsPanicHandler(&result)
func DeferWailsPanicHandler[T any](result *common.R[T]) {
	if r := recover(); r != nil {
		switch err := r.(type) {
		case common.ServicePanic:
			*result = common.F[T](err.Code, err.Msg)
		case error:
			*result = common.F[T](500, "操作产生异常错误："+err.Error())
		default:
			*result = common.F[T](500, "发生未知异常："+fmt.Sprintf("%v", r))
		}
	}
}
