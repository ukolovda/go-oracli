First of all, this is a blatant copy from https://github.com/lukasmartinelli/pgclimb but modified to use with an Oracle Database.

An Oracle utility to export data into different data formats with support for templates.

Features:
- Export data to [JSON](#json-document), [JSON Lines](#json-lines), [CSV](#csv-and-tsv), [XLSX](#xlsx), [XML](#xml)
- Use [Templates](#templates) to support custom formats (HTML, Markdown, Text)

Use Cases:
- SQLPLUS alternative for getting data out of Oracle
- Publish data sets
- Create Excel reports from the database
- Generate HTML reports
- Export XML data for further processing with XSLT
- Transform data to JSON for graphing it with JavaScript libraries
- Generate readonly JSON APIs

# TODOs
- [ ] Document and test everything from here to the end (now it is just a copy from pgclimb)
- [ ] Make connection parameters more Oracle-like
- [ ] Provide Oracle DSN as option: USER/PASSWORD@HOST:PORT/SID
- [ ] Use more of the built-in functions from go-oci8
- [ ] Cross-compiling, Travis CI 
- [ ] Oracle-specific installation notes


## Install

TODO: You can download a single binary for Linux, OSX or Windows.

**Install from source**

```bash
go get github.com/tapester/oracli
```

**Important** 
At least on windows, you need to copy and paste OCI.DLL from your database version to the folder where oracli is used.

## Supported Formats

The example queries operate on the open data [employee salaries of Montgomery County Maryland](https://data.montgomerycountymd.gov/api/views/54rh-89p8/rows.csv). 
To connect to your ~~beloved~~ Oracle database set the [appropriate connection options](#database-connection).

### CSV and TSV

```bash
# Write CSV file to stdout with comma as default delimiter
oracli -c "SELECT * FROM employee_salaries" csv

# Save CSV file with custom delimiter and header row to file
oracli -o salaries.csv \
    -c "SELECT full_name, position_title FROM employee_salaries" \
    csv --delimiter ";" --header

# Create TSV file with SQL query from stdin
oracli -o positions.tsv tsv <<EOF
SELECT position_title, COUNT(*) FROM employee_salaries
GROUP BY position_title
ORDER BY 1
EOF
```

### JSON Document

Creating a single JSON document of a query is helpful if you
interface with other programs like providing data for JavaScript or creating
a readonly JSON API.

```bash
# Query all salaries into JSON array
oracli -c "SELECT * FROM employee_salaries" json

# Query all employees of a position as nested JSON object
cat << EOF > employees_by_position.sql
SELECT s.position_title, json_agg(s) AS employees
FROM employee_salaries s
GROUP BY s.position_title
ORDER BY 1
EOF

# Load query from file and store it as JSON array in file
oracli -f employees_by_position.sql \
    -o employees_by_position.json \
    json
```

### JSON Lines

[Newline delimited JSON](http://jsonlines.org/) is a good format to exchange
structured data in large quantities which does not fit well into the CSV format.
Instead of storing the entire JSON array each line is a valid JSON object.

```bash
# Query all salaries as separate JSON objects
oracli -c "SELECT * FROM employee_salaries" jsonlines

# In this example we interface with jq to pluck the first employee of each position
oracli -f employees_by_position.sql jsonlines | jq '.employees[0].full_name'
```

### XLSX

Excel files are really useful to exchange data with non programmers
and create graphs and filters. You can fill different datasets into different spreedsheets and distribute one single Excel file.

```bash
# Store all salaries in XLSX file
oracli -o salaries.xlsx -c "SELECT * FROM employee_salaries" xlsx

# Create XLSX file with multiple sheets
oracli -o salary_report.xlsx \
    -c "SELECT DISTINCT position_title FROM employee_salaries" \
    xlsx --sheet "positions"
oracli -o salary_report.xlsx \
    -c "SELECT full_name FROM employee_salaries" \
    xlsx --sheet "employees"
```

### XML

You can output XML to process it with other programs like [XLST](http://www.w3schools.com/xsl/).
To have more control over the XML output you should use the `oracli` template functionality directly to generate XML.

```bash
# Output XML for each row
oracli -o salaries.xml -c "SELECT * FROM employee_salaries" xml
```

A good default XML export is currently lacking because the XML format can be controlled using templates.
If there is enough demand I will implement a solid default XML support without relying on templates.

## Templates

Templates are the most powerful feature of `oracli` and allow you to implement other formats that are not built in. In this example we will create a HTML report of the salaries.

Create a template `salaries.tpl`.

```html
<!DOCTYPE html>
<html>
    <head><title>Montgomery County MD Employees</title></head>
    <body>
        <h2>Employees</h2>
        <ul>
            {{range .}}
            <li>{{.full_name}}</li>
            {{end}}
        </ul>
    </body>
</html>
```

And now run the template.

```
oracli -o salaries.html \
    -c "SELECT * FROM employee_salaries" \
    template salaries.tpl
```

## Database Connection

Database connection details can be provided via environment variables or as separate flags.

name        | default     | flags               | description
------------|-------------|---------------------|-----------------
`DB_NAME`   | `ora`  | `-d`, `--dbname`    | database name
`DB_HOST`   | `localhost` | `--host`            | host name
`DB_PORT`   | `1521`      | `-p`, `--port`      | port
`DB_USER`   | `oracle`  | `-U`, `--username`  | database user
`DB_PASS`   |             | `--pass`            | password (or empty if none)

## Advanced Use Cases

### Different ways of Querying

Like `psql` you can specify a query at different places.

```bash
# Read query from stdin
echo "SELECT * FROM employee_salaries" | pgclimb
# Specify simple queries directly as arguments
pgclimb -c "SELECT * FROM employee_salaries"
# Load query from file
pgclimb -f query.sql
```

### Control Output

`oracli` will write the result to `stdout` by default.
By specifying the `-o` option you can write the output to a file.

```bash
pgclimb -o salaries.tsv -c "SELECT * FROM employee_salaries" tsv
```

### Using JSON aggregation

This is not a `oracli` feature but shows you how to create more complex JSON objects.

Let's query communities and join an additional birth rate table.

```bash
pgclimb -c "SELECT id, name, \\
    (SELECT array_to_json(array_agg(t)) FROM ( \\
            SELECT year, births FROM public.births \\
            WHERE community_id = c.id \\
            ORDER BY year ASC \\
        ) AS t \\
    ) AS births, \\
    FROM communities) AS c" jsonlines
```

# Contribute

## Dependencies

Go get the required dependencies for building `pgclimb`.

```bash
go get github.com/codegangsta/cli
go get github.com/mattn/go-oci8
go get github.com/jmoiron/sqlx
go get github.com/tealeg/xlsx
go get github.com/andrew-d/go-termutil
```

## Cross-compiling

We use [gox](https://github.com/mitchellh/gox) to create distributable
binaries for Windows, OSX and Linux.

```bash
docker run --rm -v "$(pwd)":/usr/src/oracli -w /usr/src/oracli tcnksm/gox:1.4.2-light
```

## Integration Tests

Run `test.sh` to run integration tests of the program with a PostgreSQL server. Take a look at the `.travis.yml`.
