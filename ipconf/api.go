package ipconf

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/feichai0017/GoChat/ipconf/domain"
)

type Response struct {
	Message string 		`json:"message"`
	Code    int    		`json:"code"`
	Data    interface{} `json:"data"`
}

// GetIpInfoList API adapte application layer
func GetIpInfoList(c context.Context, ctx *app.RequestContext) {
	//TODO: process if return eds is 0
	defer func() {
		if err := recover(); err != nil {
			ctx.JSON(consts.StatusBadRequest, utils.H{"error": err})
		}
	}()
	// build client request info
	ipConfCtx := domain.BuildIpConfContext(&c, ctx)
	// dispatch request to different endport
	eds := domain.Dispatch(ipConfCtx)
	// pack response with top 5 endports
	ipConfCtx.AppCtx.JSON(consts.StatusOK, packRes(top5Endports(eds)))
}