package middleware

import (
	"bytes"
	"fmt"
	"image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/gin-gonic/gin"
	"github.com/li5414/ipapk-server/conf"
	"github.com/li5414/ipapk-server/models"
	"github.com/li5414/ipapk-server/serializers"
	"github.com/phinexdaz/ipapk"
	uuid "github.com/satori/go.uuid"
)

//上传阿里云oss 打开注释
// 定义进度条监听器。
type OssProgressListener struct {
}

// 定义进度变更事件处理函数。
func (listener *OssProgressListener) ProgressChanged(event *oss.ProgressEvent) {
	switch event.EventType {
	case oss.TransferStartedEvent:
		fmt.Printf("Transfer Started, ConsumedBytes: %d, TotalBytes %d.\n",
			event.ConsumedBytes, event.TotalBytes)
	case oss.TransferDataEvent:
		fmt.Printf("\rTransfer Data, ConsumedBytes: %d, TotalBytes %d, %d%%.",
			event.ConsumedBytes, event.TotalBytes, event.ConsumedBytes*100/event.TotalBytes)
	case oss.TransferCompletedEvent:
		fmt.Printf("\nTransfer Completed, ConsumedBytes: %d, TotalBytes %d.\n",
			event.ConsumedBytes, event.TotalBytes)
	case oss.TransferFailedEvent:
		fmt.Printf("\nTransfer Failed, ConsumedBytes: %d, TotalBytes %d.\n",
			event.ConsumedBytes, event.TotalBytes)
	default:
	}
}

func Upload(c *gin.Context) {
	changelog := c.PostForm("changelog")
	channel := c.PostForm("channel")
	file, err := c.FormFile("file")
	if err != nil {
		return
	}

	ext := models.BundleFileExtension(filepath.Ext(file.Filename))
	if !ext.IsValid() {
		return
	}

	uid := uuid.NewV4()
	_uuid := uid.String()
	filename := filepath.Join(".data", _uuid+string(ext.PlatformType().Extention()))
	if err := c.SaveUploadedFile(file, filename); err != nil {
		fmt.Println("Error:", err)
		return
	}
	//上传阿里云oss 打开注释
	//上传app文件到阿里云oss
	client, err := oss.New("<yourEndpoint>", "<yourAccessKeyId>", "<yourAccessKeySecret>")
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(-1)
	}

	// 获取存储空间。
	bucket, err := client.Bucket("blacksteed-pub-resouces")
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(-1)
	}
	fmt.Println("uuid:", filename, "channel:", channel)
	// 上传本地文件。
	err = bucket.PutObjectFromFile("app/"+_uuid+string(ext.PlatformType().Extention()), filename, oss.Progress(&OssProgressListener{}))
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(-1)
	}

	app, err := ipapk.NewAppParser(filename)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	iconBuffer := new(bytes.Buffer)
	if err := png.Encode(iconBuffer, app.Icon); err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("app.Name:", app.Name, "BundleId:", app.BundleId, "version:", app.Version)

	bundle := new(models.Bundle)
	bundle.UUID = _uuid
	bundle.PlatformType = ext.PlatformType()
	bundle.Name = app.Name
	bundle.FileName = file.Filename
	bundle.BundleId = app.BundleId
	bundle.Version = app.Version
	bundle.Build = app.Build
	bundle.Size = app.Size
	bundle.Icon = iconBuffer.Bytes()
	bundle.ChangeLog = changelog
	bundle.Channel = channel
	if err := models.AddBundle(bundle); err != nil {
		fmt.Println("Error:", err)
		return
	}
	//如果上传阿里云oss 打开注释
	//上传阿里云oss.本地删除本地文件
	filerr := os.Remove(filename)
	if filerr != nil {
		//如果删除失败则输出 file remove Error!
		fmt.Println("file remove Error!")
		//输出错误详细信息
		fmt.Printf("%s", filerr)
	} else {
		//如果删除成功则输出 file remove OK!
		fmt.Print("file remove OK!")
	}
	c.JSON(http.StatusOK, &serializers.BundleJSON{
		UUID:       _uuid,
		Name:       bundle.Name,
		Platform:   bundle.PlatformType.String(),
		Channel:    bundle.Channel,
		BundleId:   bundle.BundleId,
		Version:    bundle.Version,
		Build:      bundle.Build,
		InstallUrl: bundle.GetInstallUrl(conf.AppConfig.RemoteAddr()),
		QRCodeUrl:  "/qrcode/" + _uuid,
		IconUrl:    "/icon/" + _uuid,
		Changelog:  bundle.ChangeLog,
		Downloads:  bundle.Downloads,
	})
}

type MobileInfo struct {
	UDID        string `form:"UDID" xml:"UDID"  binding:"required"`
	DEVICE_NAME string `form:"DEVICE_NAME" xml:"DEVICE_NAME" binding:"required"`
	VERSION     string `form:"VERSION" xml:"VERSION" binding:"required"`
	PRODUCT     string `form:"PRODUCT" xml:"PRODUCT" binding:"required"`
	IMEI        string `form:"IMEI" xml:"IMEI" binding:"required"`
	ICCID       string `form:"ICCID" xml:"ICCID" binding:"required"`
}

func UDID(c *gin.Context) {
	var xml MobileInfo
	if err := c.ShouldBindXML(&xml); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.Redirect(http.StatusMovedPermanently, "/udid/"+xml.UDID+"/"+xml.DEVICE_NAME)
}

func ShowUDID(c *gin.Context) {
	uuid := c.Param("udid")
	name := c.Param("name")
	c.HTML(http.StatusOK, "udid.html", gin.H{
		"name": name,
		"uuid": uuid,
	})
}

func UploadPage(c *gin.Context) {
	c.HTML(http.StatusOK, "upload.html", gin.H{})
}

func GetQRCode(c *gin.Context) {
	_uuid := c.Param("uuid")

	bundle, err := models.GetBundleByUID(_uuid)
	if err != nil {
		return
	}

	data := fmt.Sprintf("%v/bundle/%v?_t=%v", conf.AppConfig.BaseURL(), bundle.UUID, time.Now().Unix())
	code, err := qr.Encode(data, qr.L, qr.Unicode)
	if err != nil {
		return
	}
	code, err = barcode.Scale(code, 160, 160)
	if err != nil {
		return
	}

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, code); err != nil {
		return
	}

	c.Data(http.StatusOK, "image/png", buf.Bytes())
}

func GetIcon(c *gin.Context) {
	_uuid := c.Param("uuid")

	bundle, err := models.GetBundleByUID(_uuid)
	if err != nil {
		return
	}

	c.Data(http.StatusOK, "image/png", bundle.Icon)
}

func GetChangelog(c *gin.Context) {
	_uuid := c.Param("uuid")

	bundle, err := models.GetBundleByUID(_uuid)
	if err != nil {
		return
	}

	c.HTML(http.StatusOK, "change.html", gin.H{
		"changelog": bundle.ChangeLog,
	})
}

func GetBundle(c *gin.Context) {
	_uuid := c.Param("uuid")

	bundle, err := models.GetBundleByUID(_uuid)
	if err != nil {
		return
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"bundle":     bundle,
		"installUrl": bundle.GetInstallUrl(conf.AppConfig.RemoteAddr()),
		"qrCodeUrl":  "/qrcode/" + bundle.UUID,
		"iconUrl":    "/icon/" + bundle.UUID,
	})
}

func GetBundles(c *gin.Context) {
	bundles, err := models.GetBundles()
	if err != nil {
		return
	}

	var bundleWithExtras []serializers.BundleWithExtraJSON
	for _, bundle := range bundles {
		bundleWithExtras = append(bundleWithExtras, serializers.BundleWithExtraJSON{
			Bundle:     *bundle,
			InstallUrl: bundle.GetInstallUrl(conf.AppConfig.RemoteAddr()),
			QrCodeUrl:  "/qrcode/" + bundle.UUID,
			IconUrl:    "/icon/" + bundle.UUID,
		})
	}

	c.HTML(http.StatusOK, "channel.html", gin.H{
		"bundleWithExtras": bundleWithExtras,
	})
}

func GetBundlesIOS(c *gin.Context) {
	bundles, err := models.GetBundlesIOS()
	if err != nil {
		return
	}

	var bundleWithExtras []serializers.BundleWithExtraJSON
	for _, bundle := range bundles {
		bundleWithExtras = append(bundleWithExtras, serializers.BundleWithExtraJSON{
			Bundle:     *bundle,
			InstallUrl: bundle.GetInstallUrl(conf.AppConfig.RemoteAddr()),
			QrCodeUrl:  "/qrcode/" + bundle.UUID,
			IconUrl:    "/icon/" + bundle.UUID,
		})
	}

	c.HTML(http.StatusOK, "list.html", gin.H{
		"bundleWithExtras": bundleWithExtras,
	})
}

func GetBundlesAndroid(c *gin.Context) {
	bundles, err := models.GetBundlesAndroid()
	if err != nil {
		return
	}

	var bundleWithExtras []serializers.BundleWithExtraJSON
	for _, bundle := range bundles {
		bundleWithExtras = append(bundleWithExtras, serializers.BundleWithExtraJSON{
			Bundle:     *bundle,
			InstallUrl: bundle.GetInstallUrl(conf.AppConfig.RemoteAddr()),
			QrCodeUrl:  "/qrcode/" + bundle.UUID,
			IconUrl:    "/icon/" + bundle.UUID,
		})
	}

	c.HTML(http.StatusOK, "list.html", gin.H{
		"bundleWithExtras": bundleWithExtras,
	})
}

func GetBundlesWithChannel(c *gin.Context) {
	_channel := c.Param("channel")
	_platform := c.Param("platform")
	bundles, err := models.GetBundlesByChannle(_channel, _platform)
	if err != nil {
		return
	}

	var bundleWithExtras []serializers.BundleWithExtraJSON
	for _, bundle := range bundles {
		bundleWithExtras = append(bundleWithExtras, serializers.BundleWithExtraJSON{
			Bundle:     *bundle,
			InstallUrl: bundle.GetInstallUrl(conf.AppConfig.RemoteAddr()),
			QrCodeUrl:  "/qrcode/" + bundle.UUID,
			IconUrl:    "/icon/" + bundle.UUID,
		})
	}

	c.HTML(http.StatusOK, "channel.html", gin.H{
		"bundleWithExtras": bundleWithExtras,
	})
}

func GetVersions(c *gin.Context) {
	_uuid := c.Param("uuid")

	bundle, err := models.GetBundleByUID(_uuid)
	if err != nil {
		return
	}

	versions, err := bundle.GetVersions()
	if err != nil {
		return
	}

	c.HTML(http.StatusOK, "version.html", gin.H{
		"versions": versions,
		"uuid":     bundle.UUID,
	})
}

func GetBuilds(c *gin.Context) {
	_uuid := c.Param("uuid")
	version := c.Param("ver")

	bundle, err := models.GetBundleByUID(_uuid)
	if err != nil {
		return
	}

	builds, err := bundle.GetBuilds(version)
	if err != nil {
		return
	}

	var bundles []serializers.BundleJSON
	for _, v := range builds {
		bundles = append(bundles, serializers.BundleJSON{
			UUID:       v.UUID,
			Name:       v.Name,
			Platform:   v.PlatformType.String(),
			Channel:    bundle.Channel,
			BundleId:   v.BundleId,
			Version:    v.Version,
			Build:      v.Build,
			InstallUrl: v.GetInstallUrl(conf.AppConfig.RemoteAddr()),
			QRCodeUrl:  "/qrcode/" + v.UUID,
			IconUrl:    "/icon/" + v.UUID,
			Changelog:  bundle.ChangeLog,
			Downloads:  v.Downloads,
		})
	}

	c.HTML(http.StatusOK, "build.html", gin.H{
		"builds": bundles,
	})
}

func GetPlist(c *gin.Context) {
	_uuid := c.Param("uuid")

	bundle, err := models.GetBundleByUID(_uuid)
	if err != nil {
		return
	}

	if bundle.PlatformType != models.BundlePlatformTypeIOS {
		return
	}

	ipaUrl := conf.AppConfig.RemoteAddr() + "/ipa/" + bundle.UUID

	data, err := models.NewPlist(bundle.Name, bundle.Version, bundle.BundleId, ipaUrl).Marshall()
	if err != nil {
		return
	}

	c.Data(http.StatusOK, "application/x-plist", data)
}

func DownloadAPP(c *gin.Context) {
	_uuid := c.Param("uuid")

	bundle, err := models.GetBundleByUID(_uuid)
	if err != nil {
		return
	}

	bundle.UpdateDownload()

	downloadUrl := conf.AppConfig.ProxyURL() + "/app/" + bundle.UUID + string(bundle.PlatformType.Extention())
	c.Redirect(http.StatusMovedPermanently, downloadUrl)
}

func RefreshDB(c *gin.Context) {
	client, err := oss.New("<yourEndpoint>", "<yourAccessKeyId>", "<yourAccessKeySecret>")
	if err != nil {
		log.Println("Error22:", err)
		os.Exit(-1)
	}

	f, err := os.Open("data.db")
	if err != nil {
		log.Println("open db error")
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		log.Println("stat fileinfo error")
	}

	modifyTime := fi.ModTime()
	log.Println("本地文件修改时间:", modifyTime)
	// 获取存储空间。
	bucket, err := client.Bucket("blacksteed-pub-resouces")
	if err != nil {
		log.Println("获取存储空间失败:", err)
	}

	err = bucket.GetObjectToFile("data.db", "data.db")
	if err != nil {
		log.Println("文件没有更新，不需要下载:", err)
	}
	if err := models.RstartDB(); err != nil {
		log.Fatal(err)
	}

	c.HTML(http.StatusOK, "refresh.html", gin.H{})
}
