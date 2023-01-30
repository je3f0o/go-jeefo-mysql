# Jeefo MySQL

This is a very simple and lightweight MySQL library top of 
`github.com/go-sql-driver/mysql`. It was written in my first three days of 
learning Go programming language.

## Getting started

### Installation
```go
import "github.com/je3f0o/go-jeefo-mysql"
```
Then execute `$ go mod tidy`. Or Go command to install the package.

```sh
$ go get -u github.com/je3f0o/go-jeefo-mysql
```

### Example
config.yml
```yaml
database:
  # Default values when using `mysql.NewConfig()`
  #host: 127.0.0.1
  #port: 3306
  socket: /var/lib/mysql/mysql.sock # for Unix socket connection
  name: my_database
  user: jeefo
  pass: 123
```

main.go
```go
package main

import (
  "os"
  
  "github.com/je3f0o/go-jeefo-mysql"
  "gopkg.in/yaml.v3"
)

type _json map[string]interface{}

type Config struct {
  Database *mysql.Config
}

func readFile(filepath string) *Config {
  f, err := os.Open(filepath)
  if err != nil {
    panic(err)
  }
  defer f.Close()
  
  cfg := &Config{Database: mysql.NewConfig()}
  err = yaml.NewDecoder(f).Decode(cfg)
  if err != nil {
    panic(err)
  }
  return cfg
}

func main() {
  cfg := readFile("config.yml")
  mysql.Init(cfg.Database)
  // and ready to go...

  // If you want to see logging query string with values before executing
  mysql.Debug = true

  // Some examples...
  //
  // insert single row
  mysql.InsertRow("users", _json{"email": "user@domain.tld"})

  // Select single row
  user := mysql.First("users", _json{"email": "user@domain.tld"})

  // Update single row
  data  := _json{ "email": "username@other-domain.tld" }
  where := _json{ "user_id": user["id"] }
  mysql.UpdateFirst("users", data, where)

  // Delete single row
  mysql.DeleteFirst("users", _json{ "id": user["id"] })

  // more, look at the documentation...
}
```

### Documentation
See full [API](https://je3f0o.github.io/go-jeefo-mysql/) for more documantation.

## LICENSE
[MIT](https://github.com/je3f0o/go-jeefo-mysql/blob/master/LICENSE)