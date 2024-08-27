package httpserver

import (
	"embed"
	"errors"
	"fmt"
	"github.com/textthree/provider/config"
	"github.com/textthree/provider/core"
	"github.com/textthree/provider/core/types"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	routeTypeGoStatic = iota + 1
	routeTypeGoGroup
)

type MiddlewareHandler func(c *Context) error
type RequestHandler func(c *Context) // API / 控制器函数

// 框架核心结构体
type Engine struct {
	router              map[string]map[string]t3WebRoute
	globalMiddlewares   []MiddlewareHandler
	groupMiddlewares    map[string][]MiddlewareHandler
	requestHandler      RequestHandler
	container           core.Container
	cross               bool
	swaggerUiFileSystem fs.FS
	config              config.Service
}

type t3WebRoute struct {
	middlewares    []MiddlewareHandler
	requestHandler RequestHandler
	routeType      int8 // 路由类型：1.golang 静态路由 2.golang 分组路由
	prefix         string
}

// 使用 embed 包嵌入 swagger-ui 目录下的所有文件。
// go：embed 是 Golang 的一种特殊注释，"//"与"go:embed"之间不能有空格。用于指示编译器在编译时将指定的文件或目录嵌入到生成的二进制文件中。
// 虽然它看起来像普通注释，但实际上它是一个指令，告诉编译器执行特定的操作。
// go：embed 要求与之关联的变量是全局的，不要放在函数内部。
//
//go:embed swagger-ui/*
var swaggerUI embed.FS

// 初始化框架核心结构
func (self *Engine) NewHttpEngine(serviceCenter core.Container, cfgsvc config.Service) (engine *Engine) {
	// 路由 map 的第一维存请求方式，二维存控制器
	router := map[string]map[string]t3WebRoute{}
	router["GET"] = map[string]t3WebRoute{}
	router["POST"] = map[string]t3WebRoute{}
	router["PUT"] = map[string]t3WebRoute{}
	router["DELETE"] = map[string]t3WebRoute{}
	engine = &Engine{
		router:           router,
		groupMiddlewares: map[string][]MiddlewareHandler{}, // 分组路由(批量前缀)路由上挂的中间件
		container:        serviceCenter,
		config:           cfgsvc,
	}
	// swagger 支持
	if cfg := cfgsvc.GetSwagger(); cfg.FilePath != "" {
		// 创建子文件系统以指向 swagger-ui 目录
		swaggerFS, err := fs.Sub(swaggerUI, "swagger-ui")
		if err != nil {
			fmt.Println("无法创建文件系统: %v", err)
			return
		}
		engine.swaggerUiFileSystem = swaggerFS
	}
	return
}

// 注册全局中间件
func (self *Engine) UseMiddleware(handlers ...MiddlewareHandler) {
	self.globalMiddlewares = handlers
}

// 跨域
func (self *Engine) Cross() {
	self.cross = true
}

func (self t3WebRoute) IsEmpty() bool {
	return self.routeType == 0
}

// 添加路由到 map
// prefix 主要用于请求进来时匹配分组中间件
func (self *Engine) AddRoute(method, prefix, uri string, routeType int8, handler RequestHandler, middlewares ...MiddlewareHandler) {
	uri = strings.ToLower(uri)
	if self.router[method][uri].routeType != 0 {
		err := errors.New("route exist: " + uri)
		panic(err)
	}
	key := strings.Replace(prefix+uri, "/", "", -1)
	self.router[method][key] = t3WebRoute{
		middlewares:    middlewares,
		requestHandler: handler,
		routeType:      routeType,
		prefix:         prefix,
	}
}

func Cross(response http.ResponseWriter) {
	response.Header().Set("Access-Control-Allow-Origin", "*")
	response.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	response.Header().Set("Access-Control-Allow-Headers", "*")
}

// 框架核心结构实现 Handler 接口
func (self *Engine) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if self.cross {
		Cross(response)
	}
	if request.URL.Path == "/favicon.ico" {
		return
	}
	if request.Method == "OPTIONS" {
		response.WriteHeader(200)
		return
	}
	if strings.HasPrefix(request.URL.Path, "/swagger") {
		self.handelSwaggerUI(request, response)
		return
	}

	// 静态资源服务
	fileServerCfg := self.config.GetFileServer()
	if fileServerCfg != (types.FileSeverConfig{}) {
		http.StripPrefix(fileServerCfg.Route, http.FileServer(http.Dir(fileServerCfg.Path))).ServeHTTP(response, request)
		return
	}

	// 初始化自定义 context
	ctx := NewContext(request, response, self.container)

	// 寻找路由，handlers 包含中间件 + 控制器
	route := self.FindRouteHandler(request)
	if route.IsEmpty() {
		ctx.Resp.SetStatus(404).Text("404 not found")
		return
	}
	// 注入中间件、控制器给 context
	middlewareChain := self.globalMiddlewares
	if route.prefix != "" {
		middlewareChain = append(middlewareChain, self.groupMiddlewares[route.prefix]...)
	}
	for _, middleware := range route.middlewares {
		middlewareChain = append(middlewareChain, middleware)
	}
	ctx.SetMiddwares(middlewareChain)
	// 执行中间件、控制器
	if err := ctx.Next(); err != nil {
		ctx.Resp.SetStatus(500).Text(err.Error())
		return
	}

	// 执行控制器函数
	route.requestHandler(ctx)
}

// 匹配路由，如果没有匹配到，返回 nil
func (self *Engine) FindRouteHandler(request *http.Request) t3WebRoute {
	// 转换大小写，确保大小写不敏感
	method := strings.ToUpper(request.Method)
	key := strings.Replace(request.URL.Path, "/", "", -1)
	key = strings.ToLower(key)

	//fmt.Println(method, key)
	//for k, _ := range self.router[method] {
	//	fmt.Println(k)
	//}

	// 查找第一层 map
	if methodHandlers, ok := self.router[method]; ok {
		if handler, ok := methodHandlers[key]; ok {
			return handler
		}
	}
	return t3WebRoute{}
}

func (self *Engine) handelSwaggerUI(request *http.Request, response http.ResponseWriter) {
	// 判断配置是否开启
	if self.config.GetSwagger().FilePath == "" {
		response.WriteHeader(404)
		response.Write([]byte("404 not found"))
		return
	}

	// 处理 swagger-doc 路由
	docFilePath := self.config.GetSwagger().FilePath
	docFileName := filepath.Base(docFilePath)
	if strings.HasSuffix(request.URL.Path, docFileName) {
		data, _ := os.ReadFile(docFilePath)
		response.Write(data)
		return
	}

	// 处理 swagger-initializer.js 注入 swagger 文档 url
	if strings.HasSuffix(request.URL.Path, "swagger-initializer.js") {
		// 加载模板
		tmpl, err := template.ParseFS(self.swaggerUiFileSystem, "swagger-initializer.js")
		if err != nil {
			http.Error(response, "Unable to parse swagger-initializer.js", http.StatusInternalServerError)
			return
		}
		// 传递动态数据
		data := struct {
			SwaggerJsonURL string
		}{
			SwaggerJsonURL: "http://localhost:" + self.config.GetHttpPort() + "/" + docFileName,
		}
		// 渲染模板
		response.Header().Set("Content-Type", "application/javascript")
		tmpl.Execute(response, data)
		return
	}

	// 处理其他 Swagger UI 静态资源的请求
	http.StripPrefix("/swagger-ui/", http.FileServer(http.FS(self.swaggerUiFileSystem))).ServeHTTP(response, request)
}
