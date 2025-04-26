package domain

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

type IpConfContext struct {
	Ctx 		*context.Context
	AppCtx 		*app.RequestContext
	ClientCtx 	*ClientContext
}

type ClientContext struct {
	IP string `json:"ip"`
}

func BuildIpConfContext(c *context.Context, ctx *app.RequestContext) *IpConfContext {
	ipConfCtx := &IpConfContext{
		Ctx: 		c,
		AppCtx: 	ctx,
		ClientCtx: 	&ClientContext{},
	}
	return ipConfCtx
}