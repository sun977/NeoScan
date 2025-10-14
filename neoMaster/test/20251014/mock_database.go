// Package test 提供用于测试的Mock数据库实现
// 这是一个纯内存的GORM兼容数据库，避免CGO依赖
package test

import (
	"database/sql/driver"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// MockDB 内存数据库结构
type MockDB struct {
	tables map[string]map[string]interface{} // 表名 -> 记录ID -> 记录数据
}

// MockDialector GORM方言实现
type MockDialector struct {
	db *MockDB
}

// Name 返回方言名称
func (d MockDialector) Name() string {
	return "mock"
}

// Initialize 初始化方言
func (d MockDialector) Initialize(db *gorm.DB) error {
	// 注册回调
	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{})
	return nil
}

// Migrator 返回迁移器
func (d MockDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return &MockMigrator{db: db}
}

// DataTypeOf 返回数据类型
func (d MockDialector) DataTypeOf(field *schema.Field) string {
	return "TEXT"
}

// DefaultValueOf 返回默认值
func (d MockDialector) DefaultValueOf(field *schema.Field) clause.Expression {
	return clause.Expr{SQL: "NULL"}
}

// BindVarTo 绑定变量
func (d MockDialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	writer.WriteByte('?')
}

// QuoteTo 引用标识符
func (d MockDialector) QuoteTo(writer clause.Writer, str string) {
	writer.WriteByte('`')
	writer.WriteString(str)
	writer.WriteByte('`')
}

// Explain 解释查询
func (d MockDialector) Explain(sql string, vars ...interface{}) string {
	return fmt.Sprintf("EXPLAIN: %s", sql)
}

// MockMigrator GORM迁移器实现
type MockMigrator struct {
	db *gorm.DB
}

// AutoMigrate 自动迁移
func (m *MockMigrator) AutoMigrate(dst ...interface{}) error {
	return nil
}

// CurrentDatabase 返回当前数据库名称
func (m *MockMigrator) CurrentDatabase() string {
	return "mock_database"
}

// FullDataTypeOf 返回完整数据类型
func (m *MockMigrator) FullDataTypeOf(field *schema.Field) clause.Expr {
	return clause.Expr{SQL: "TEXT"}
}

// GetTypeAliases 获取类型别名
func (m *MockMigrator) GetTypeAliases(databaseTypeName string) []string {
	return []string{}
}

// CreateTable 创建表
func (m *MockMigrator) CreateTable(dst ...interface{}) error {
	return nil
}

// DropTable 删除表
func (m *MockMigrator) DropTable(dst ...interface{}) error {
	return nil
}

// HasTable 检查表是否存在
func (m *MockMigrator) HasTable(dst interface{}) bool {
	return true
}

// RenameTable 重命名表
func (m *MockMigrator) RenameTable(oldName, newName interface{}) error {
	return nil
}

// GetTables 获取所有表名
func (m *MockMigrator) GetTables() ([]string, error) {
	return []string{}, nil
}

// MockTableType 实现gorm.TableType接口
type MockTableType struct{}

func (t MockTableType) Schema() string                   { return "mock" }
func (t MockTableType) Name() string                     { return "mock_table" }
func (t MockTableType) Type() string                     { return "TABLE" }
func (t MockTableType) Comment() (comment string, ok bool) { return "", false }

// TableType 获取表类型
func (m *MockMigrator) TableType(dst interface{}) (gorm.TableType, error) {
	return MockTableType{}, nil
}

// AddColumn 添加列
func (m *MockMigrator) AddColumn(dst interface{}, field string) error {
	return nil
}

// DropColumn 删除列
func (m *MockMigrator) DropColumn(dst interface{}, field string) error {
	return nil
}

// AlterColumn 修改列
func (m *MockMigrator) AlterColumn(dst interface{}, field string) error {
	return nil
}

// MigrateColumn 迁移列
func (m *MockMigrator) MigrateColumn(dst interface{}, field *schema.Field, columnType gorm.ColumnType) error {
	return nil
}

// MigrateColumnUnique 迁移列唯一约束
func (m *MockMigrator) MigrateColumnUnique(dst interface{}, field *schema.Field, columnType gorm.ColumnType) error {
	return nil
}

// HasColumn 检查列是否存在
func (m *MockMigrator) HasColumn(dst interface{}, field string) bool {
	return true
}

// RenameColumn 重命名列
func (m *MockMigrator) RenameColumn(dst interface{}, oldName, field string) error {
	return nil
}

// ColumnTypes 获取列类型
func (m *MockMigrator) ColumnTypes(dst interface{}) ([]gorm.ColumnType, error) {
	return []gorm.ColumnType{}, nil
}

// CreateView 创建视图
func (m *MockMigrator) CreateView(name string, option gorm.ViewOption) error {
	return nil
}

// DropView 删除视图
func (m *MockMigrator) DropView(name string) error {
	return nil
}

// CreateConstraint 创建约束
func (m *MockMigrator) CreateConstraint(dst interface{}, name string) error {
	return nil
}

// DropConstraint 删除约束
func (m *MockMigrator) DropConstraint(dst interface{}, name string) error {
	return nil
}

// HasConstraint 检查约束是否存在
func (m *MockMigrator) HasConstraint(dst interface{}, name string) bool {
	return false
}

// CreateIndex 创建索引
func (m *MockMigrator) CreateIndex(dst interface{}, name string) error {
	return nil
}

// DropIndex 删除索引
func (m *MockMigrator) DropIndex(dst interface{}, name string) error {
	return nil
}

// HasIndex 检查索引是否存在
func (m *MockMigrator) HasIndex(dst interface{}, name string) bool {
	return false
}

// RenameIndex 重命名索引
func (m *MockMigrator) RenameIndex(dst interface{}, oldName, newName string) error {
	return nil
}

// GetIndexes 获取索引
func (m *MockMigrator) GetIndexes(dst interface{}) ([]gorm.Index, error) {
	return []gorm.Index{}, nil
}

// MockConnector 连接器实现
type MockConnector struct{}

// Connect 连接数据库
func (c MockConnector) Connect(ctx interface{}) (driver.Conn, error) {
	return &MockConn{}, nil
}

// Driver 返回驱动
func (c MockConnector) Driver() driver.Driver {
	return &MockDriver{}
}

// MockDriver 驱动实现
type MockDriver struct{}

// Open 打开连接
func (d *MockDriver) Open(name string) (driver.Conn, error) {
	return &MockConn{}, nil
}

// MockConn 连接实现
type MockConn struct{}

// Prepare 准备语句
func (c *MockConn) Prepare(query string) (driver.Stmt, error) {
	return &MockStmt{}, nil
}

// Close 关闭连接
func (c *MockConn) Close() error {
	return nil
}

// Begin 开始事务
func (c *MockConn) Begin() (driver.Tx, error) {
	return &MockTx{}, nil
}

// MockStmt 语句实现
type MockStmt struct{}

// Close 关闭语句
func (s *MockStmt) Close() error {
	return nil
}

// NumInput 输入参数数量
func (s *MockStmt) NumInput() int {
	return 0
}

// Exec 执行语句
func (s *MockStmt) Exec(args []driver.Value) (driver.Result, error) {
	return &MockResult{}, nil
}

// Query 查询
func (s *MockStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &MockRows{}, nil
}

// MockTx 事务实现
type MockTx struct{}

// Commit 提交事务
func (t *MockTx) Commit() error {
	return nil
}

// Rollback 回滚事务
func (t *MockTx) Rollback() error {
	return nil
}

// MockResult 结果实现
type MockResult struct{}

// LastInsertId 最后插入ID
func (r *MockResult) LastInsertId() (int64, error) {
	return 1, nil
}

// RowsAffected 影响行数
func (r *MockResult) RowsAffected() (int64, error) {
	return 1, nil
}

// MockRows 行集实现
type MockRows struct {
	closed bool
}

// Columns 列名
func (r *MockRows) Columns() []string {
	return []string{}
}

// Close 关闭行集
func (r *MockRows) Close() error {
	r.closed = true
	return nil
}

// Next 下一行
func (r *MockRows) Next(dest []driver.Value) error {
	return fmt.Errorf("no more rows")
}

// NewMockGormDB 创建Mock GORM数据库实例
func NewMockGormDB() (*gorm.DB, error) {
	mockDB := &MockDB{
		tables: make(map[string]map[string]interface{}),
	}
	
	dialector := MockDialector{db: mockDB}
	
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	
	return db, nil
}