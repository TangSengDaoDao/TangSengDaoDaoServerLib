package module

import (
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/register"
	"github.com/gocraft/dbr/v2"
	migrate "github.com/rubenv/sql-migrate"
)

func Setup(ctx *config.Context) error {

	// 获取所有模块
	ms := register.GetModules(ctx)

	// Legacy compatibility path. New applications must execute schema changes
	// through an explicit migration process and use SetupRuntime for ordinary
	// service startup. Keep the former behavior here for downstream consumers
	// that have not yet moved to the explicit lifecycle.
	// 初始化SQL
	var sqlfss []*register.SQLFS
	for _, m := range ms {
		if m.SQLDir != nil {
			sqlfss = append(sqlfss, m.SQLDir)
		}

	}
	err := executeSQL(sqlfss, ctx.DB())
	if err != nil {
		return err
	}
	return setupRuntimeModules(ctx, ms)

}

// SetupRuntime registers APIs and optional tasks only. It must be used by
// ordinary Server startup so a restart cannot execute schema migration.
func SetupRuntime(ctx *config.Context) error {
	return setupRuntimeModules(ctx, register.GetModules(ctx))
}

func setupRuntimeModules(ctx *config.Context, ms []register.Module) error {
	if ctx == nil || ctx.GetHttpRoute() == nil {
		return fmt.Errorf("runtime setup requires an HTTP route")
	}
	// 注册api
	for _, m := range ms {
		if m.SetupAPI != nil {
			a := m.SetupAPI()
			if a != nil {
				a.Route(ctx.GetHttpRoute())
			}
		}
		if ctx.SetupTask {
			if m.SetupTask != nil {
				t := m.SetupTask()
				if t != nil {
					t.RegisterTasks()
				}
			}
		}
	}
	return nil
}

// RegisteredMigrationSource exposes only the SQLFS registry. Calling it does
// not instantiate modules or create a business runtime context.
func RegisteredMigrationSource() FileDirMigrationSource {
	return FileDirMigrationSource{sqlfss: register.GetMigrationSources()}
}

func Start(ctx *config.Context) error {
	// 获取所有模块
	ms := register.GetModules(ctx)
	for _, m := range ms {
		if m.Start != nil {
			err := m.Start()
			if err != nil {
				return err
			}
		}

	}
	return nil
}
func Stop(ctx *config.Context) error {
	// 获取所有模块
	ms := register.GetModules(ctx)
	for _, m := range ms {
		if m.Stop != nil {
			err := m.Stop()
			if err != nil {
				return err
			}
		}

	}
	return nil
}

// 执行sql
func executeSQL(sqlfss []*register.SQLFS, session *dbr.Session) error {
	migrations := &FileDirMigrationSource{
		sqlfss: sqlfss,
	}
	_, err := migrate.Exec(session.DB, "mysql", migrations, migrate.Up)
	if err != nil {
		return err
	}
	return nil
}

type byID []*migrate.Migration

func (b byID) Len() int           { return len(b) }
func (b byID) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byID) Less(i, j int) bool { return b[i].Less(b[j]) }

// FileDirMigrationSource 文件目录源 遇到目录进行递归获取
type FileDirMigrationSource struct {
	sqlfss []*register.SQLFS
}

// FindMigrations FindMigrations
func (f FileDirMigrationSource) FindMigrations() ([]*migrate.Migration, error) {

	if len(f.sqlfss) == 0 {
		return nil, nil
	}
	migrations := make([]*migrate.Migration, 0, 100)

	for _, sqlfs := range f.sqlfss {
		err := f.findMigrations(sqlfs, &migrations)
		if err != nil {
			return nil, err
		}
	}

	// Make sure migrations are sorted
	sort.Sort(byID(migrations))

	return migrations, nil
}

func (f FileDirMigrationSource) findMigrations(fs *register.SQLFS, migrations *[]*migrate.Migration) error {

	files, err := fs.ReadDir("sql")
	if err != nil {
		return err
	}
	for _, info := range files {

		if strings.HasSuffix(info.Name(), ".sql") {
			file, err := fs.Open(path.Join("sql", info.Name()))
			if err != nil {
				return fmt.Errorf("error while opening %s: %s", info.Name(), err)
			}

			migration, parseErr := migrate.ParseMigration(info.Name(), file.(io.ReadSeeker))
			closeErr := file.Close()
			if parseErr != nil {
				return fmt.Errorf("error while parsing %s: %s", info.Name(), parseErr)
			}
			if closeErr != nil {
				return fmt.Errorf("error while closing %s: %s", info.Name(), closeErr)
			}
			*migrations = append(*migrations, migration)

		}
	}

	return nil
}
