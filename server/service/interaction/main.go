package main

import (
	interaction "GoYin/server/kitex_gen/interaction/interactionserver"
	"GoYin/server/service/interaction/config"
	"GoYin/server/service/interaction/dao"
	"GoYin/server/service/interaction/initialize"
	"GoYin/server/service/interaction/pkg"
	"context"
	"github.com/cloudwego/kitex/pkg/limit"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/pkg/utils"
	"github.com/cloudwego/kitex/server"
	"github.com/kitex-contrib/obs-opentelemetry/provider"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	"log"
	"net"
)

func main() {
	initialize.InitLogger()
	r, info := initialize.InitNacos()
	db := initialize.InitDB()
	rdb := initialize.InitRedis()
	videoClient := initialize.InitVideo()
	p := provider.NewOpenTelemetryProvider(
		provider.WithServiceName(config.GlobalServerConfig.Name),
		provider.WithExportEndpoint(config.GlobalServerConfig.OtelInfo.EndPoint),
		provider.WithInsecure(),
	)
	defer p.Shutdown(context.Background())
	impl := &InteractionServerImpl{
		VideoManager:    pkg.NewVideoManager(videoClient),
		RedisManager:    dao.NewRedisManager(rdb),
		CommentManager:  dao.NewMysqlManager(db),
		FavoriteManager: dao.NewMysqlManager(db),
	}
	svr := interaction.NewServer(impl,
		server.WithServiceAddr(utils.NewNetAddr("tcp", net.JoinHostPort(config.GlobalServerConfig.Host, config.GlobalServerConfig.Port))),
		server.WithRegistry(r),
		server.WithRegistryInfo(info),
		server.WithLimit(&limit.Option{MaxConnections: 2000, MaxQPS: 500}),
		server.WithSuite(tracing.NewServerSuite()),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: config.GlobalServerConfig.Name}))

	err := svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
