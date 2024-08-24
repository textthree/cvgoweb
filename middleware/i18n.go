package middleware

import (
	"cvgo/provider"
	"cvgo/provider/httpserver"
	"cvgo/provider/i18n"
	"github.com/spf13/viper"
	"github.com/textthree/cvgokit/filekit"
	"os"
	"path/filepath"
)

// dir 语言包文件所在目录（绝对路径）
func I18n() httpserver.MiddlewareHandler {
	return func(ctx *httpserver.Context) error {
		pwd, _ := os.Getwd()
		lngCode := ctx.GetVal("Language").ToString()
		if lngCode == "" {
			lngCode = "en"
		}
		i18nDir := filepath.Join(pwd, "i18n")
		languagePkg := filepath.Join(i18nDir, lngCode+".json")
		provider.Clog().Trace("Use i18n middleware", languagePkg)
		i18nService := ctx.Holder().NewSingle(i18n.Name).(i18n.Service)
		i18nService.SetLngCode(lngCode)
		if !i18nService.LoadedPackage(lngCode) {
			if exists, _ := filekit.PathExists(languagePkg); exists {
				file := viper.New()
				file.AddConfigPath(i18nDir)
				file.SetConfigName(lngCode + ".json")
				file.SetConfigType("json")
				if err := file.ReadInConfig(); err != nil {
					provider.Clog().Error(err)
				} else {
					i18nService.SetLanguagePackage(lngCode, file)
				}
			} else {
				provider.Clog().Error("语言包不存在:", languagePkg)
			}
		}
		ctx.Next()
		return nil
	}
}
