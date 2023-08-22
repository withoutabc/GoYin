package main

import (
	"GoYin/server/common/middleware"
	user "GoYin/server/kitex_gen/user/userservice"
	"GoYin/server/service/user/config"
	"GoYin/server/service/user/dao"
	"GoYin/server/service/user/initialize"
	"GoYin/server/service/user/pkg"
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
	socialClient := initialize.InitSocial()
	interactionClient := initialize.InitInteraction()
	chatClient := initialize.InitChat()
	p := provider.NewOpenTelemetryProvider(
		provider.WithServiceName(config.GlobalServerConfig.Name),
		provider.WithExportEndpoint(config.GlobalServerConfig.OtelInfo.EndPoint),
		provider.WithInsecure(),
	)
	defer p.Shutdown(context.Background())
	impl := &UserServiceImpl{
		Jwt:                middleware.NewJWT(config.GlobalServerConfig.JWTInfo.SigningKey),
		InteractionManager: pkg.NewInteractionManager(interactionClient),
		SocialManager:      pkg.NewSocialManager(socialClient),
		ChatManager:        pkg.NewChatManager(chatClient),
		RedisManager:       dao.NewRedisManager(rdb),
		MysqlManager:       dao.NewUser(db),
	}
	// Create new server.
	srv := user.NewServer(impl,
		server.WithServiceAddr(utils.NewNetAddr("tcp", net.JoinHostPort(config.GlobalServerConfig.Host, config.GlobalServerConfig.Port))),
		server.WithRegistry(r),
		server.WithRegistryInfo(info),
		server.WithLimit(&limit.Option{MaxConnections: 2000, MaxQPS: 500}),
		server.WithSuite(tracing.NewServerSuite()),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: config.GlobalServerConfig.Name}),
	)

	err := srv.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
