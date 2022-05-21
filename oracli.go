package main

import (
	"context"
	"database/sql"
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
				cli.BoolFlag{
					Name:   "truncate",
					EnvVar: "DO_TRUNCATE",
					Usage:  "truncate tables before inserting",
				},
				cli.BoolFlag{
					Name:   "replica-mode",
					EnvVar: "REPLICA_MODE",
					Usage:  "set replica mode before loading",
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
					c.Bool("truncate"),
					c.Bool("replica-mode"),
				)
				export2(ora.ParseConnStr(c), format, cfg)
			},
		},
	}

	app.Run(os.Args)
}

type PsqlFormat struct {
	writer      io.Writer
	buffer      strings.Builder
	columns     []string
	doTruncate  bool
	replicaMode bool
}

func NewPsqlFormat(w io.Writer, doTruncate bool, replicaMode bool) *PsqlFormat {

	return &PsqlFormat{
		writer:      w,
		columns:     make([]string, 0),
		doTruncate:  doTruncate,
		replicaMode: replicaMode,
	}
}

func (f *PsqlFormat) WriteFileHeader() error {
	_, err := fmt.Fprint(f.writer, "begin transaction;\n")
	if err != nil {
		return err
	}
	if f.replicaMode {
		_, err = fmt.Fprint(f.writer, "set constraints all deferred;\n")
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(f.writer, "set session_replication_role to replica;\n\n")
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *PsqlFormat) WriteTableHeader(tableName string, columns []string) error {
	f.columns = columns

	_, err := fmt.Fprintf(f.writer, "-- %s\n\n", tableName)
	if err != nil {
		return err
	}

	if f.doTruncate {
		_, err := fmt.Fprintf(f.writer, "truncate table %s cascade;\n\n", tableName)
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(f.writer, "copy %s (%s) from stdin;\n", tableName, strings.Join(columns, ","))
	return err
}

func (f *PsqlFormat) FlushTable() error {
	if f.buffer.Len() > 0 {
		_, err := f.writer.Write([]byte(f.buffer.String()))
		if err != nil {
			return err
		}
		f.buffer.Reset()
	}
	//f.writer.Flush()
	_, err := fmt.Fprintf(f.writer, "\\.\n\n")
	return err
}

func (f *PsqlFormat) WriteFileFooter() error {
	var err error
	if f.replicaMode {
		_, err = fmt.Fprint(f.writer, "set session_replication_role to default;\n\n")
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprint(f.writer, "commit;\n\n")
	if err != nil {
		return err
	}
	//f.writer.Flush()
	//return f.writer.Error()
	return nil
}

func pgEscapeBytes(data []byte) string {
	return fmt.Sprintf(`\\x%X`, data)
}

func pgEscapeString(data string) string {
	var s = data
	s = strings.Replace(s, "\\", `\\`, -1)
	s = strings.Replace(s, "\t", `\t`, -1)
	s = strings.Replace(s, "\r", `\r`, -1)
	s = strings.Replace(s, "\n", `\n`, -1)
	s = strings.Replace(s, "\b", `\b`, -1)
	s = strings.Replace(s, "\v", `\v`, -1)
	return s
}

func (f *PsqlFormat) WriteRow(rowIndex int64, values map[string]interface{}) error {
	//record := []string{}
	for index, col := range f.columns {
		if index > 0 {
			f.buffer.WriteRune('\t')
		}
		//fmt.Printf("Value: %v\n", values[col])
		switch value := (values[col]).(type) {
		case nil:
			f.buffer.WriteString(`\N`)
		case []byte:
			f.buffer.WriteString(pgEscapeBytes(value))
		case int64:
			f.buffer.WriteString(fmt.Sprintf("%d", value))
		case float64:
			f.buffer.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
		case time.Time:
			f.buffer.WriteString(value.Format(time.RFC3339))
		case bool:
			if value == true {
				f.buffer.WriteString("true")
			} else {
				f.buffer.WriteString("false")
			}
		default:
			f.buffer.WriteString(pgEscapeString(fmt.Sprintf("%v", value)))
		}
	}
	f.buffer.WriteRune('\n')
	//err := f.writer.Write(record)
	//if err != nil {
	//	return err
	//}

	// Сбрасываем буфер каждые 100 записей
	if f.buffer.Len() > 0 {
		_, err := f.writer.Write([]byte(f.buffer.String()))
		f.buffer.Reset()
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *PsqlFormat) AddSequenceFix(sequenceName string, newValue int64) error {
	_, err := fmt.Fprintf(f.writer, "SELECT setval('%s', %d);\n\n", sequenceName, newValue)
	//err := format.AddSequenceFix(sequenceName, idResult.MaxId)
	if err == nil {
		_, err = fmt.Fprintf(os.Stderr, "Sequence %s switched to %d\n", sequenceName, newValue)
	}
	return err
}

// Supports storing data in different formats
type DataFormat2 interface {
	WriteFileHeader() error
	WriteTableHeader(tableName string, columns []string) error
	WriteRow(rowIndex int64, values map[string]interface{}) error
	AddSequenceFix(sequenceName string, newValue int64) error
	FlushTable() error
	WriteFileFooter() error
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

	if err = format.WriteTableHeader(tableName, columnNames); err != nil {
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

	if err = format.FlushTable(); err != nil {
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

	if err := format.WriteFileHeader(); err != nil {
		return err
	}

	for _, tableName := range names {

		if tableName != "DEFAULT" { // Особое имя, которого на самом деле нет
			section := cfg.Section(tableName)
			sqlText := section.Key("sql").String()
			if sqlText == "" {
				sqlText = fmt.Sprintf("select * from %s", tableName)
			}

			err = exportTable(tx, format, tableName, sqlText)
			//return rows.Err()

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error export for %q: %v\n", tableName, err)
				continue
			}
			id := section.Key("id").String()
			sequenceName := section.Key("s").String()
			if id != "" && sequenceName != "" {
				sqlText = fmt.Sprintf("select coalesce(max(%s), 0) max_id from %s", id, tableName)
				//fmt.Fprintf(os.Stderr, "Sequence check sql: %q\n", sql)
				// You can also get a single result, a la QueryRow
				var idResult IdResult
				err = tx.Get(&idResult, sqlText)
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
	err = format.WriteFileFooter()
	return err
}
