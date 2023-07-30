package register

import (
	"embed"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
)

// APIRouter api路由者
type APIRouter interface {
	Route(r *wkhttp.WKHttp)
}

var apiRoutes = make([]APIRouter, 0)

// Add 添加api
func Add(r APIRouter) {
	apiRoutes = append(apiRoutes, r)
}

var taskRoutes = make([]TaskRouter, 0)

// GetRoutes 获取所有路由者
func GetRoutes() []APIRouter {
	return apiRoutes
}

// TaskRouter task路由者
type TaskRouter interface {
	RegisterTasks()
}

// AddTask 添加任务
func AddTask(task TaskRouter) {
	taskRoutes = append(taskRoutes, task)
}

// GetTasks 获取所有任务
func GetTasks() []TaskRouter {
	return taskRoutes
}

var modules = make([]Module, 0)

// 模块
type Module struct {
	SetupAPI func(ctx interface{}) APIRouter // api 路由
	SQLDir   SQLFS                           // sql目录
}

func AddModule(m Module) {
	modules = append(modules, m)

	m.SQLDir.ReadDir("sql")
}

type SQLFS struct {
	embed.FS
}

func GetModules() []Module {
	return modules
}
