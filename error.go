package mysql

import m "github.com/go-sql-driver/mysql"

type Error struct {
  Query      string
  Values     []interface{}
  MySQLError *m.MySQLError
}