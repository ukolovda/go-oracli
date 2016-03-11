package ora

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/codegangsta/cli"
	/*_ "gopkg.in/rana/ora.v3"*/
	_ "github.com/mattn/go-oci8"
)

//setup a database connection and create the import schema
func Connect(connStr string) (*sqlx.DB, error) {	
	fmt.Printf(connStr)
	fmt.Printf("\n")
	db, err := sqlx.Open("oci8", connStr)
	if err != nil {
		fmt.Println(err)
		return db, err
	}

	err = db.Ping()
	if err != nil {
		fmt.Printf("Error connecting to the database: %s\n", err)
		return db, nil
	}
	fmt.Println("Connection Successful\n")
	return db, nil
}

//parse sql connection string from cli flags
func ParseConnStr(c *cli.Context) string {
	return fmt.Sprintf("%s/%s@%s:%s/%s",
		c.GlobalString("username"),
		c.GlobalString("pass"),
		c.GlobalString("host"),
		c.GlobalString("port"),
		c.GlobalString("dbname"),
	)
}
