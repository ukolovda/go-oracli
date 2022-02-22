package main

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/tapester/go-oracli/formats"
	"github.com/tapester/go-oracli/ora"
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
			Name:  "output, o",
			Value: "",
			Usage: "Output filename",
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
	}

	app.Run(os.Args)
}
