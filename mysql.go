// This is a very simple and lightweight MySQL library for learning the Go 
// programming language. It was written in my first three days of learning Go.
//
// Please note that this library currently does not support multiple different 
// database connections. Most microservices typically only have one database, 
// but for projects that require multiple different database connections, this 
// library may be updated in the future to support them.
package mysql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"

	m "github.com/go-sql-driver/mysql"
)

type Config struct {
  Host     string `yaml:"host,omitempty"`
  Port     int16  `yaml:"port,omitempty"`
  Socket   string `yaml:"socket,omitempty"`
  DBName   string `yaml:"name"`
  Username string `yaml:"user"`
  Password string `yaml:"pass"`
}

type _where struct {
  query  string
  values []interface{}
}

var db *sql.DB

// Set to `true` will be logging every query with values before executing.
var Debug = false

// Returns a pointer to a newly allocated `Config` struct with default values
// for:
//   - `Host` "127.0.0.1"
//   - `Port` 3306
func NewConfig() *Config {
  return &Config{
    Host: "127.0.0.1",
    Port: 3306,
  }
}

// Escapes a SQL identifier for safe use in a query.
//
// Parameters:
//   - `id`: SQL identifier a table or column name to be escaped
//   - `ignore_dot`: Optional Boolean value, which when set to `true` the dot 
//                    (.) character is not escaped
// Returns:
//   - string : the escaped identifier
// Example:
//   EscapeId("INFORMATION_SCHEMA.COLUMNS")       // output: `INFORMATION_SCHEMA`.`COLUMNS`
//   EscapeId("some.weird.table.or.column", true) // output: `some.weird.table.or.column`
func EscapeId(id string, ignore_dot ...bool) string {
  if len(ignore_dot) > 0 && ignore_dot[0] {
    return "`" + strings.Replace(id, "`", "``", -1) + "`"
  }

  parts := strings.Split(id, ".")
  for i, part := range parts {
    parts[i] = "`" + strings.Replace(part, "`", "``", -1) + "`"
  }
  return strings.Join(parts, ".")
}

// Initialize database connection with given configuration.
func Init(cfg *Config) {
  var target string

  if cfg.Socket != "" {
    target = fmt.Sprintf("unix(%s)", cfg.Socket)
  } else {
    target = fmt.Sprintf("tcp(%s:%d)", cfg.Host, cfg.Port)
  }

  args := []interface{}{cfg.Username, cfg.Password, target, cfg.DBName }
  connect_string := fmt.Sprintf("%s:%s@%s/%s?charset=utf8", args...)
  var err error
  db, err = sql.Open("mysql", connect_string)
  if err != nil { panic(err) }

  err = db.Ping()
  if err != nil { panic(err) }
}

// Retrieve data from specified `table` with the given `where` condition and 
// options.
//
// Parameters:
//   - `table`: name of the table to perform the SELECT query on
//   - `where`: conditions to be used in the WHERE clause of the query
//   - `options`: Optional map specify additional options
// Options:
//   - `column`: string, specify single column to return
//   - `columns`: string array for multiple columns to return
//   - `order`: string, order of the results
//   - `offset`: int, this option will be discarded without limit
//   - `limit`: int, maximum number of results
//
// Returns:
//   - []map[string]interface{}: rows data returned by the query
//
// Example:
//   type _json map[string]interface{}
//
//   where   := _json{"user_id": user_id}
//   options := _json{"order": "created_at DESC", "limit": 30}
//   rows    := mysql.Select("producst", where, optioins)
func Select(
  table string,
  where map[string]interface{},
  args ...map[string]interface{},
) []map[string]interface{} {
  var options map[string]interface{}
  if len(args) > 0 { options = args[0] }

  cols := prepare_columns(options)
  w := prepare_where(where)

  order  := order_query(options)
  limit  := limit_query(options, true)
  format := "SELECT %s FROM %s%s%s%s;"
  query := fmt.Sprintf(format, cols, EscapeId(table), w.query, order, limit)
  rows  := ExecQuery(query, w.values...)
  defer rows.Close()

  columns, err := rows.Columns()
  if err != nil { panic(err) }

  values := make([]sql.RawBytes, len(columns))
  // Make a slice of pointers to the values
  valuePtrs := make([]interface{}, len(columns))
  for i := range values {
    valuePtrs[i] = &values[i]
  }

  var results []map[string]interface{}
  for rows.Next() {
    if err := rows.Scan(valuePtrs...); err != nil {
      panic(err)
    }
    // Create a map to hold the column names and values
    result := map[string]interface{}{}
    for i, col := range columns {
      result[col] = string(values[i])
    }
    results = append(results, result)
  }

  return results
}

// Same api with `Select(...)` method except it will override `options["limit"]` 
// to set 1 and returns a single row if found.
func First(
  table string,
  where map[string]interface{},
  options ...map[string]interface{},
) map[string]interface{} {
  set_limit_option(&options)
  results := Select(table, where, options...)
  if len(results) == 1 {
    return results[0]
  }
  return nil
}

// Inserts data into a table.
//
// Parameters:
//   - `table`: The name of the table to insert into
//   - `data`: A map of the column names and values to be inserted into the 
//               table
//
// Returns:
//   - sql.Result: Result of the insert statement execution
// TODO: update this method to support multiple rows
func Insert(table string, data map[string]interface{}) sql.Result {
  var values       []any
  var columns      []string
  var placeholders []string

  for k, v := range data {
    values       = append(values, v)
    columns      = append(columns, EscapeId(k))
    placeholders = append(placeholders, "?")
  }

  cols  := strings.Join(columns, ", ")
  vals  := strings.Join(placeholders, ", ")
  args  := []interface{}{ EscapeId(table), cols, vals }
  query := fmt.Sprintf("INSERT INTO %s(%s) VALUES(%s)", args...)

  return Exec(query, values...)
}

// Insert a single row data into a table.
//
// Parameters:
//   - `table`: The name of the table to insert into
//   - `data`: A map of the column names and values to be inserted into the 
//               table
//
// Returns:
//   - sql.Result: Result of the insert statement execution
func InsertRow(table string, data map[string]interface{}) sql.Result {
  set, values := prepare_set(data)
  query := fmt.Sprintf("INSERT INTO %s SET %s;", table, set)
  return Exec(query, values...)
}

// Updates the data in a table with specified conditions.
//
// Parameters:
//   - `table`: The name of the table to update
//   - `data`: A map of field names and new values to update in the table
//   - `where`: A map of conditions to determine which rows to update in the 
//              table
//   - `options`: An optional set of options to specify order and limit for the 
//                update query
//
// Returns:
//   - sql.Result: Result of the update query
func Update(
  table string,
  data, where map[string]interface{},
  args ...map[string]interface{},
) sql.Result {
  var options map[string]interface{}
  if len(args) > 0 { options = args[0] }

  set, values := prepare_set(data)
  w := prepare_where(where)
  values = append(values, w.values...)
  
  order  := order_query(options)
  limit  := limit_query(options, false)

  params := []interface{}{ EscapeId(table), set, w.query, order, limit }
  query  := fmt.Sprintf("UPDATE %s SET %s%s%s%s;", params...)
  return Exec(query, values...)
}

// Same api with `Update(...)` method except it will override `options["limit"]` 
// to set 1.
func UpdateFirst(
  table string,
  data, where map[string]interface{},
  options ...map[string]interface{},
) sql.Result {
  set_limit_option(&options)
  return Update(table, data, where, options...)
}

// Deletes data from a specified table.
//
// Parameters:
//   - `table`: The name of the table
//   - `where`: The conditions to specify which records to delete
//   - `options`: Additional options, such as "order" or "limit"
// Returns:
//   - sql.Result: Result of the delete operation
func Delete(
  table string,
  where map[string]interface{},
  args ...map[string]interface{},
) sql.Result {
  var options map[string]interface{}
  if len(args) > 0 { options = args[0] }

  w := prepare_where(where)
  order := ""
  if val, ok := options["order"].(string); ok {
    order = " ORDER BY " + val
  }

  limit := ""
	if val, ok := options["limit"].(int); ok {
		limit = fmt.Sprintf(" LIMIT %d", val)
	}

  query := fmt.Sprintf("DELETE FROM %s%s%s%s;", table, w.query, order, limit)
  return Exec(query, w.values...)
}

// Same api with `Delete(...)` method except it will override `options["limit"]` 
// to set 1.
func DeleteFirst(
  table string,
  where map[string]interface{},
  options ...map[string]interface{},
) sql.Result {
  set_limit_option(&options)
  return Delete(table, where, options...)
}

// Executes an user defined query with values. Which is useful when user wants 
// to use `sql.Rows.Scan(...)` method to convert datatypes.
//
// Parameters:
//   - `query`: the query to be executed
//   - `values`: parameters to be passed to the query
// Returns:
//   - *sql.Rows: SQL rows cursor
func ExecQuery(query string, values ...interface{}) *sql.Rows {
  if Debug { log.Println(query, values) }
  rows, err := db.Query(query, values...)
  if err != nil { handle_error(err, query, values) }
  return rows
}

// Executes an user defined query.
//
// Parameters:
//   - `query`: the query to be executed
//   - `values`: parameters to be passed to the query
// Returns:
//   - sql.Result: A Result summarizes an executed SQL query
func Exec(query string, values ...interface{}) sql.Result {
  if Debug { log.Println(query, values) }
  result, err := db.Exec(query, values...)
  if err != nil { handle_error(err, query, values) }
  return result
}

func handle_error(err error, query string, values ...interface{}) {
  if mysql_err, ok := err.(*m.MySQLError); ok {
    panic(&Error{query, values, mysql_err})
  }
  panic(err)
}

func order_query(options map[string]interface{}) string {
  order := ""
  if val, ok := options["order"].(string); ok {
    order = " ORDER BY " + val
  }
  return order
}

func limit_query(
  options map[string]interface{},
  has_offset bool,
) string {
	if _limit, ok := options["limit"].(int); ok {
    if has_offset {
      offset := 0
      if value, ok := options["offset"].(int); ok {
        offset = value
      }
      return fmt.Sprintf(" LIMIT %d, %d", offset, _limit)
    }
    return fmt.Sprintf(" LIMIT %d", _limit)
	}
	return ""
}

func prepare_columns(options map[string]interface{}) string {
  field, ok := options["column"].(string)
  if ok { return EscapeId(field) }

  fields, ok := options["columns"].([]string)
  if !ok { return "*" }

  for i, f := range fields {
    fields[i] = EscapeId(f)
  }
  return strings.Join(fields, ", ")
}

func prepare_where(where map[string]interface{}) _where {
  var values []interface{}
	var query string
	if where != nil {
    conditions := []string{}

    for key, value := range where {
      key = EscapeId(key)
      if value == nil {
        conditions = append(conditions, key+" IS NULL")
      } else if reflect.TypeOf(value).Kind() == reflect.Slice {
        v := reflect.ValueOf(value)
        placeholders := []string{}
        for i := 0; i < v.Len(); i++ {
          values = append(values, v.Index(i).Interface())
          placeholders = append(placeholders, "?")
        }
        query := fmt.Sprintf("%sIN(%s)", key, strings.Join(placeholders, ", "))
        conditions = append(conditions, query)
      } else {
        if reflect.TypeOf(value).Kind() == reflect.Map {
          bytes, _ := json.Marshal(value)
          value = string(bytes)
        }
        values = append(values, value)
        conditions = append(conditions, key+" = ?")
      }
    }

    query = " WHERE "+strings.Join(conditions, " AND ")
  }

  return _where{query: query, values: values}
}

func prepare_set(data map[string]interface{}) (string, []interface{}) {
	var values []interface{}
	var columns = make([]string, len(data))
  var i int
	for key, value := range data {
		if value == nil {
			columns[i] = fmt.Sprintf("%s = NULL", EscapeId(key))
		} else {
			values     = append(values, value)
			columns[i] = fmt.Sprintf("%s = ?", EscapeId(key))
		}
    i++
	}
	return strings.Join(columns, ", "), values
}

func set_limit_option(options *[]map[string]interface{}) {
  switch len(*options) {
  case 0: *options = []map[string]interface{}{ {"limit": 1} }
  case 1: (*options)[0]["limit"] = 1
  }
}