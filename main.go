package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/li5414/ipapk-server/conf"
	"github.com/li5414/ipapk-server/middleware"
	"github.com/li5414/ipapk-server/models"
	"github.com/li5414/ipapk-server/templates"
	"github.com/li5414/ipapk-server/utils"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func Init() {
	_, err := os.Stat(".data")
	if os.IsNotExist(err) {
		os.MkdirAll(".data", 0755)
	}
	if err := utils.InitCA(); err != nil {
		log.Fatal(err)
	}
	if err := conf.InitConfig("config.json"); err != nil {
		log.Fatal(err)
	}

	if err := models.InitDB(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	Init()

	router := gin.Default()
	router.SetFuncMap(templates.TplFuncMap)
	router.LoadHTMLGlob("public/views/*")

	router.Static("app", ".data")
	router.Static("static", "public/static")
	router.StaticFile("myCA.cer", ".ca/myCA.cer")

	router.GET("/udid/:udid/:name", middleware.ShowUDID)
	router.POST("/udid", middleware.UDID)
	router.POST("/upload", middleware.Upload)
	router.GET("/", middleware.GetBundles)
	router.GET("/ios", middleware.GetBundlesIOS)
	router.GET("/android", middleware.GetBundlesAndroid)
	router.GET("/channel/:channel/:platform",middleware.GetBundlesWithChannel)
	router.GET("/refresh",middleware.RefreshDB)
	router.GET("/bundle/:uuid", middleware.GetBundle)
	router.GET("/log/:uuid", middleware.GetChangelog)
	router.GET("/qrcode/:uuid", middleware.GetQRCode)
	router.GET("/icon/:uuid", middleware.GetIcon)
	router.GET("/plist/:uuid", middleware.GetPlist)
	router.GET("/ipa/:uuid", middleware.DownloadAPP)
	router.GET("/apk/:uuid", middleware.DownloadAPP)
	router.GET("/version/:uuid", middleware.GetVersions)
	router.GET("/version/:uuid/:ver", middleware.GetBuilds)

	srv := &http.Server{
		Addr:    conf.AppConfig.Addr(),
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServeTLS(".ca/mycert1.cer", ".ca/mycert1.key"); err != nil {
			log.Printf("listen: %v\n", err)
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown:", err)
	}
}
