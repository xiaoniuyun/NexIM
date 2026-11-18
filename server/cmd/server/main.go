package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"nexim/server/internal/api"
	"nexim/server/internal/config"
	"nexim/server/internal/gateway"
	"nexim/server/internal/mq"
	"nexim/server/internal/service"
	"nexim/server/internal/store"

	"go.uber.org/zap"
)

func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}
	if err := config.Load(cfgPath); err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	log, _ := zap.NewProduction()
	defer log.Sync()

	// 输出部署形态摘要（lite / standard）
	log.Info("deployment profile", zap.String("summary", config.Summary()),
		zap.Bool("lite", config.IsLite()), zap.Bool("standard", config.IsStandard()))

	st, err := store.New(&config.C, log)
	if err != nil {
		log.Fatal("init store", zap.Error(err))
	}

	authSvc := service.NewAuth(st, &config.C)
	msgSvc := service.NewMessage(st)
	contactSvc := service.NewContact(st)
	groupSvc := service.NewGroup(st)
	fileDir := config.C.Storage.LocalDir
	if fileDir == "" {
		fileDir = "data/files"
	}
	fileSvc := service.NewFile(st, fileDir)
	adminSvc := service.NewAdmin(st)
	callSvc := service.NewCall(st)
	hub := gateway.NewHub(st, &config.C, msgSvc, contactSvc, authSvc, callSvc, log)
	router := api.NewRouter(st, &config.C, authSvc, msgSvc, contactSvc, groupSvc, fileSvc, adminSvc, callSvc, hub)

	// NATS JetStream 事件总线（standard 形态）：消息落库后发布事件，各节点消费后向本节点在线用户投递
	if config.C.MQ.URL != "" && (strings.EqualFold(config.C.MQ.Driver, "jetstream") || strings.EqualFold(config.C.MQ.Driver, "nats")) {
		nodeID := nodeName()
		mqClient, mqErr := mq.NewNATS(config.C.MQ.URL, config.C.MQ.Stream, config.C.MQ.Retention, log)
		if mqErr != nil {
			if config.IsStandard() {
				log.Fatal("init nats jetstream", zap.Error(mqErr))
			}
			log.Warn("nats unavailable, mq disabled", zap.Error(mqErr))
		} else {
			hub.SetMQ(mqClient, nodeID)
			defer mqClient.Close()
			log.Info("nats jetstream ready",
				zap.String("url", config.C.MQ.URL),
				zap.String("stream", config.C.MQ.Stream),
				zap.String("node", nodeID))
		}
	}

	// WebSocket 服务（独立端口）
	wsMux := http.NewServeMux()
	wsMux.HandleFunc("/ws", router.WSHandler)
	wsSrv := &http.Server{Addr: config.C.Server.WSAddr, Handler: wsMux}
	go func() {
		log.Info("gateway listening", zap.String("addr", config.C.Server.WSAddr))
		if err := wsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("ws server", zap.Error(err))
		}
	}()

	// HTTP API 服务
	log.Info("api listening", zap.String("addr", config.C.Server.HTTPAddr), zap.String("profile", config.C.Deployment.Profile))
	httpSrv := &http.Server{Addr: config.C.Server.HTTPAddr, Handler: router.Engine()}
	if err := httpSrv.ListenAndServe(); err != nil {
		log.Fatal("http server", zap.Error(err))
	}
}

// nodeName 生成网关节点标识：容器主机名 + 进程号，用于 MQ 事件去重与独立消费位点
func nodeName() string {
	host, _ := os.Hostname()
	if host == "" {
		host = "nexim"
	}
	return fmt.Sprintf("%s-%d", host, os.Getpid())
}
