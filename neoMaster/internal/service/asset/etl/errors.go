// 错误分类
// 定义 ETL 过程中可能遇到的错误类型，用于分类和处理
package etl

import (
	"errors"
	"net"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// ETLErrorType 定义错误类型
type ETLErrorType int

const (
	ErrorTypeUnknown    ETLErrorType = iota
	ErrorTypeTransient               // 瞬时错误 (可重试)
	ErrorTypePersistent              // 持久错误 (不可重试)
)

// ClassifyError 根据错误类型进行分类
func ClassifyError(err error) ETLErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}

	// 1. 检查是否为 MySQL 错误
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1213, // ER_LOCK_DEADLOCK: Deadlock found when trying to get lock
			1205, // ER_LOCK_WAIT_TIMEOUT: Lock wait timeout exceeded
			1040, // ER_CON_COUNT_ERROR: Too many connections
			2002, // CR_CONNECTION_ERROR: Can't connect to local MySQL server
			2003, // CR_CONN_HOST_ERROR: Can't connect to MySQL server on '...'
			2006, // CR_SERVER_GONE_ERROR: MySQL server has gone away
			2013, // CR_SERVER_LOST: Lost connection to MySQL server during query
			1053, // ER_SERVER_SHUTDOWN: Server shutdown in progress
			1062: // ER_DUP_ENTRY: Duplicate entry (视为瞬时错误，因为可能是并发 Upsert 导致的 Race Condition，重试通常能解决)
			return ErrorTypeTransient

		case 1452, // ER_NO_REFERENCED_ROW_2: Cannot add or update a child row: a foreign key constraint fails
			1451, // ER_ROW_IS_REFERENCED_2: Cannot delete or update a parent row: a foreign key constraint fails
			1054, // ER_BAD_FIELD_ERROR: Unknown column
			1146, // ER_NO_SUCH_TABLE: Table doesn't exist
			1064, // ER_PARSE_ERROR: You have an error in your SQL syntax
			1292, // ER_TRUNCATED_WRONG_VALUE: Truncated incorrect ... value
			1406: // ER_DATA_TOO_LONG: Data too long for column
			return ErrorTypePersistent
		}
	}

	// 2. 检查网络错误
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return ErrorTypeTransient
		}
		// 其他网络错误通常也是瞬时的
		return ErrorTypeTransient
	}

	// 3. 检查常见字符串匹配 (兜底)
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "too many open files") {
		return ErrorTypeTransient
	}

	// 默认为 Unknown，策略上通常选择重试以防万一，或者根据业务决定
	// 在 Processor 中，Unknown 会被默认重试
	return ErrorTypeUnknown
}

// IsTransient 判断是否为瞬时错误
func IsTransient(err error) bool {
	return ClassifyError(err) == ErrorTypeTransient
}

// IsPersistent 判断是否为持久错误
func IsPersistent(err error) bool {
	return ClassifyError(err) == ErrorTypePersistent
}
