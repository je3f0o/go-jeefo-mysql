package mysql

import (
	"strconv"
	"time"
)

const layout = "2006-01-02 15:04:05.000"

// Parse a SQL datetime string in the format "2006-01-02 15:04:05.000" and 
// convert to a `time.Time`.
//
// Parameters:
//   - `value`: a string representation of a date and time
// Returns:
//   - `time.Time`: representation of the input string. If the input string is 
//   not in the expected format, it will panic.
//
// Example:
//   type _json map[string]interface{}
//
//   where := _json{ "access_token": access_token }
//   data  := mysql.First(oauth2.TokensTable, where, _json{
//     "columns": []string{
//       "user_id",
//       "access_token_expires_at",
//       "refresh_token_expires_at",
//     },
//   })
//   if data == nil { return }
//   expires_at := mysql.ParseDatetime(data["access_token_expires_at"])
//   // code...
func ParseDatetime(value interface{}) time.Time {
  t, err := time.Parse(layout, value.(string))
  if err != nil { panic(err) }
  return t
}

// Converts a string to uint32
//
// Parameters:
//   - `value`: representation of an integer
// Returns:
//   - `uint32`: converted integer as uint32
func ParseUint32(value interface{}) uint32 {
  i, err := strconv.Atoi(value.(string))
  if err != nil { panic(err) }
  return uint32(i)
}