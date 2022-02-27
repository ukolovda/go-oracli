package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/go-ini/ini"
	"github.com/jmoiron/sqlx"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	termutil "github.com/andrew-d/go-termutil"
	_ "github.com/mattn/go-oci8"
	"github.com/ukolovda/go-oracli/formats"
	"github.com/ukolovda/go-oracli/ora"
	"github.com/urfave/cli"
)

func changeHelpTemplateArgs(args string) {
	cli.CommandHelpTemplate = strings.Replace(cli.CommandHelpTemplate, "[arguments...]", args, -1)
}

func parseTemplate(filename string) string {
	rawTemplate, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalln(err)
	}
	return string(rawTemplate)
}

func parseWriter(c *cli.Context) io.Writer {
	outputFilename := c.GlobalString("output")

	if outputFilename != "" {
		f, err := os.Create(outputFilename)
		exitOnError(err)
		return f
	}
	return os.Stdout
}

func exportFormat(c *cli.Context, format formats.DataFormat) {
	connStr := ora.ParseConnStr(c)
	query, err := parseQuery(c)
	exitOnError(err)
	err = formats.Export(query, connStr, format)
	exitOnError(err)
}

func parseQuery(c *cli.Context) (string, error) {
	filename := c.GlobalString("file")
	if filename != "" {
		query, err := ioutil.ReadFile(filename)
		return string(query), err
	}

	command := c.GlobalString("command")
	if command != "" {
		return command, nil
	}

	if !termutil.Isatty(os.Stdin.Fd()) {
		query, err := ioutil.ReadAll(os.Stdin)
		return string(query), err
	}

	return "", errors.New("You need to specify a SQL query.")
}

func exitOnError(err error) {
	log.SetFlags(0)
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "oracli"
	app.Version = "0.1"
	app.Usage = "Export data from Oracle into different data formats. Use either DSN or the individual parameters."

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "dsn",
			Value:  "",
			Usage:  "username/password@host:port/dbname",
			EnvVar: "DB_DSN",
		},
		cli.StringFlag{
			Name:   "dbname, d",
			Value:  "ora",
			Usage:  "database",
			EnvVar: "DB_NAME",
		},
		cli.StringFlag{
			Name:   "host",
			Value:  "localhost",
			Usage:  "host name",
			EnvVar: "DB_HOST",
		},
		cli.StringFlag{
			Name:   "port, p",
			Value:  "1521",
			Usage:  "port",
			EnvVar: "DB_PORT",
		},
		cli.StringFlag{
			Name:   "username, U",
			Value:  "oracle",
			Usage:  "username",
			EnvVar: "DB_USER",
		},
		cli.StringFlag{
			Name:   "password, pass",
			Value:  "",
			Usage:  "password",
			EnvVar: "DB_PASS",
		},
		cli.StringFlag{
			Name:   "query, command, c",
			Value:  "",
			Usage:  "SQL query to execute",
			EnvVar: "DB_QUERY",
		},
		cli.StringFlag{
			Name:  "file, f",
			Value: "",
			Usage: "SQL query filename",
		},
		cli.StringFlag{
			Name:   "ini",
			Value:  "",
			Usage:  "Export INI filename",
			EnvVar: "SETTINGS_INI",
		},
		cli.StringFlag{
			Name:   "output, o",
			Value:  "",
			Usage:  "Output filename",
			EnvVar: "OUTPUT_FILE",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "template",
			Usage: "Export data with custom template",
			Action: func(c *cli.Context) {
				changeHelpTemplateArgs("<template>")

				templateArg := c.Args().First()
				if templateArg == "" {
					cli.ShowCommandHelp(c, "template")
					os.Exit(1)
				}

				rawTemplate := parseTemplate(templateArg)
				writer := parseWriter(c)
				exportFormat(c, formats.NewTemplateFormat(writer, rawTemplate))
			},
		},
		{
			Name:  "jsonlines",
			Usage: "Export newline-delimited JSON objects",
			Action: func(c *cli.Context) {
				format := formats.NewJSONLinesFormat(parseWriter(c))
				exportFormat(c, format)
			},
		},
		{
			Name:  "json",
			Usage: "Export JSON document",
			Action: func(c *cli.Context) {
				format := formats.NewJSONArrayFormat(parseWriter(c))
				exportFormat(c, format)
			},
		},
		{
			Name:  "csv",
			Usage: "Export CSV",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "delimiter",
					Value: ",",
					Usage: "column delimiter",
				},
				cli.BoolFlag{
					Name:  "header",
					Usage: "output header row",
				},
			},
			Action: func(c *cli.Context) {
				delimiter, _ := utf8.DecodeRuneInString(c.String("delimiter"))
				format := formats.NewCsvFormat(
					parseWriter(c),
					delimiter,
					c.Bool("header"),
				)
				exportFormat(c, format)
			},
		},
		{
			Name:  "tsv",
			Usage: "Export TSV",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "header",
					Usage: "output header row",
				},
			},
			Action: func(c *cli.Context) {
				format := formats.NewCsvFormat(
					parseWriter(c),
					'\t',
					c.Bool("header"),
				)
				exportFormat(c, format)
			},
		},
		{
			Name:  "xml",
			Usage: "Export XML",
			Action: func(c *cli.Context) {
				format := formats.NewXMLFormat(parseWriter(c))
				exportFormat(c, format)
			},
		},
		{
			Name:  "xlsx",
			Usage: "Export XLSX spreadsheets",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "sheet",
					Value: "data",
					Usage: "spreadsheet name",
				},
			},
			Action: func(c *cli.Context) {
				format, err := formats.NewXlsxFormat(
					c.GlobalString("output"),
					c.String("sheet"),
				)
				exitOnError(err)
				exportFormat(c, format)
			},
		},
		{
			Name:  "psql",
			Usage: "Export PSQL",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "inifile",
					Value:  "",
					EnvVar: "INIFILE",
					Usage:  "Settings Inifile",
				},
			},
			Action: func(c *cli.Context) {
				iniName := c.String("inifile")
				cfg, err := ini.Load(iniName)
				if err != nil {
					fmt.Printf("Fail to read ini file %q: %v", iniName, err)
					os.Exit(1)
				}
				format := NewPsqlFormat(
					parseWriter(c),
				)
				export2(ora.ParseConnStr(c), format, cfg)
			},
		},
	}

	app.Run(os.Args)
}

type PsqlFormat struct {
	rawWriter io.Writer
	writer    *csv.Writer
	columns   []string
}

func NewPsqlFormat(w io.Writer) *PsqlFormat {

	writer := csv.NewWriter(w)
	writer.Comma = ','
	return &PsqlFormat{
		rawWriter: w,
		writer:    writer,
		columns:   make([]string, 0),
	}
}

func (f *PsqlFormat) WriteHeader(tableName string, columns []string) error {
	f.columns = columns

	_, err := fmt.Fprintf(f.rawWriter, "-- %s\n", tableName)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(f.rawWriter, "copy %s (%s) from stdin;\n", tableName, strings.Join(columns, ","))
	return err
}

func (f *PsqlFormat) Flush() error {
	f.writer.Flush()
	_, err := fmt.Fprintf(f.rawWriter, "\\.\n\n")
	return err
}

func BytesToPgBytea(data []byte) string {
	return fmt.Sprintf(`\x%X`, data)
}

func (f *PsqlFormat) WriteRow(rowIndex int64, values map[string]interface{}) error {
	record := []string{}
	for _, col := range f.columns {
		//fmt.Printf("Value: %v\n", values[col])
		switch value := (values[col]).(type) {
		case nil:
			record = append(record, `\N`)
		case []byte:
			record = append(record, BytesToPgBytea(value))
		case int64:
			record = append(record, fmt.Sprintf("%d", value))
		case float64:
			record = append(record, strconv.FormatFloat(value, 'f', -1, 64))
		case time.Time:
			record = append(record, value.Format(time.RFC3339))
		case bool:
			if value == true {
				record = append(record, "true")
			} else {
				record = append(record, "false")
			}
		default:
			record = append(record, fmt.Sprintf("%v", value))
		}

	}
	err := f.writer.Write(record)
	if err != nil {
		return err
	}

	// Сбрасываем буфер каждые 100 записей
	if rowIndex%100 == 0 {
		f.writer.Flush()
		err = f.writer.Error()
	}
	return err
}

func (f *PsqlFormat) AddSequenceFix(sequenceName string, newValue int64) error {
	_, err := fmt.Fprintf(f.rawWriter, "SELECT setval('%s', %d);\n\n", sequenceName, newValue)
	//err := format.AddSequenceFix(sequenceName, idResult.MaxId)
	if err == nil {
		_, err = fmt.Fprintf(os.Stderr, "Sequence %s switched to %d\n", sequenceName, newValue)
	}
	return err
}

// Supports storing data in different formats
type DataFormat2 interface {
	WriteHeader(tableName string, columns []string) error
	WriteRow(rowIndex int64, values map[string]interface{}) error
	AddSequenceFix(sequenceName string, newValue int64) error
	Flush() error
}

func exportTable(db *sqlx.Tx, format DataFormat2, tableName string, sql string) error {

	startTime := time.Now()

	fmt.Fprintf(os.Stderr, "%s (%s)\n", tableName, sql)

	rows, err := db.Queryx(sql)
	if err != nil {
		return err
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {
		return err
	}

	if err = format.WriteHeader(tableName, columnNames); err != nil {
		return err
	}

	var rowIndex int64 = 0
	for rows.Next() {
		values := make(map[string]interface{})
		if err = rows.MapScan(values); err != nil {
			return err
		}

		if err = format.WriteRow(rowIndex, values); err != nil {
			return err
		}

		// Индикация каждые 10000 записей
		if rowIndex%10000 == 0 {
			os.Stderr.Write([]byte("."))
		}

		rowIndex += 1
	}

	duration := time.Now().Sub(startTime)

	fmt.Fprintf(os.Stderr, " OK, %d rows, time: %v\n", rowIndex, duration.Seconds())

	if err = format.Flush(); err != nil {
		return err
	}

	return nil
}

type IdResult struct {
	MaxId int64 `db:"MAX_ID"`
}

func export2(connStr string, format DataFormat2, cfg *ini.File) error {
	db, err := ora.Connect(connStr)
	if err != nil {
		return err
	}

	defer db.Close()

	names := cfg.SectionStrings()

	txOptions := sql.TxOptions{ReadOnly: true}
	tx := db.MustBeginTx(context.Background(), &txOptions)

	for _, tableName := range names {

		if tableName != "DEFAULT" { // Особое имя, которого на самом деле нет
			section := cfg.Section(tableName)
			sql := section.Key("sql").String()
			if sql == "" {
				sql = fmt.Sprintf("select * from %s", tableName)
			}

			err = exportTable(tx, format, tableName, sql)
			//return rows.Err()

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error export for %q: %v\n", tableName, err)
				continue
			}
			id := section.Key("id").String()
			sequenceName := section.Key("s").String()
			if id != "" && sequenceName != "" {
				sql = fmt.Sprintf("select coalesce(max(%s), 0) max_id from %s", id, tableName)
				//fmt.Fprintf(os.Stderr, "Sequence check sql: %q\n", sql)
				// You can also get a single result, a la QueryRow
				var idResult IdResult
				err = tx.Get(&idResult, sql)
				if idResult.MaxId > 0 {
					err := format.AddSequenceFix(sequenceName, idResult.MaxId)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error export for %q: %v\n", tableName, err)
					}
					//} else {
					//	fmt.Fprintf(os.Stderr, "id is empty\n")
				}
			}
		}

	}
	return nil
}
