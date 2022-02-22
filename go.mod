module ukolovda/go-oracli

go 1.16

replace github.com/ukolovda/go-oracli => github.com/ukolovda/go-oracli v0.0.0-20220222081626-3d6c83830e90

//replace github.com/ukolovda/go-oracli/ora => github.com/ukolovda/go-oracli/ora export_settings

replace github.com/codegangsta/cli => github.com/urfave/cli v1.22.2-0.20191024042601-850de854cda0

require (
	github.com/jmoiron/sqlx v1.3.4
	github.com/mattn/go-oci8 v0.1.1
	github.com/tealeg/xlsx v1.0.5
	github.com/urfave/cli v1.22.5
)

require (
	github.com/andrew-d/go-termutil v0.0.0-20150726205930-009166a695a2
	github.com/ukolovda/go-oracli v0.0.0-20161216152736-555b609ceae1
)
