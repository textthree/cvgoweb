package middleware

//func Auth() httpserver.MiddlewareHandler {
//	cfg := provider.Services.NewSingle(config.Name).(config.Service)
//	secret := cfg.GetTokenSecret()
//
//	return func(context *httpserver.Context) error {
//		token, _ := context.Req.Header("Authorization")
//		var uid string
//		if token != "" {
//			uid = cryptokit.DynamicDecrypt(secret, token)
//		}
//		//clog.PinkPrintf("tokenKey=%s, token=%s, userId=%s \n", secret, token, uid)
//		if token == "" || cast.ToInt64(uid) == 0 {
//			ret := dto.BaseRes{
//				ApiCode:    1000,
//				ApiMessage: "Authorization failed.",
//			}
//			info := string(jsonkit.JsonEncode(ret))
//			return errors.New(info)
//		}
//		context.SetVal("uid", uid)
//		context.Next()
//		return nil
//	}
//}
