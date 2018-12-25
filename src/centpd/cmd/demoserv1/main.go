package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"centpd/lib/altthumber"
	di "centpd/lib/demoib"
	"centpd/lib/emime"
	fl "centpd/lib/filelogger"
	"centpd/lib/fstore"
	. "centpd/lib/logx"
	"centpd/lib/nntp"
	"centpd/lib/psql"
	"centpd/lib/psqlib"
)

func main() {
	var err error
	// initialize flags
	dbconnstr := flag.String("dbstr", "", "postgresql connection string")
	nntpbind := flag.String("nntpbind", "", "nntp server bind string")

	flag.Parse()

	// logger
	lgr, err := fl.NewFileLogger(os.Stderr, DEBUG, fl.ColorAuto)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fl.NewFileLogger error: %v\n", err)
		os.Exit(1)
	}
	mlg := NewLogToX(lgr, "main")
	mlg.LogPrint(DEBUG, "testing DEBUG log message")
	mlg.LogPrint(INFO, "testing INFO log message")
	mlg.LogPrint(NOTICE, "testing NOTICE log message")
	mlg.LogPrint(WARN, "testing WARN log message")
	mlg.LogPrint(ERROR, "testing ERROR log message")
	mlg.LogPrint(CRITICAL, "testing CRITICAL log message")

	err = emime.LoadMIMEDatabase("mime.types")
	if err != nil {
		mlg.LogPrintln(CRITICAL, "LoadMIMEDatabase err:", err)
		return
	}

	db, err := psql.OpenPSQL(psql.Config{
		Logger:  lgr,
		ConnStr: *dbconnstr,
	})
	if err != nil {
		mlg.LogPrintln(CRITICAL, "psql.OpenPSQL error:", err)
		return
	}
	defer db.Close()

	valid, err := db.IsValidDB()
	if err != nil {
		mlg.LogPrintln(CRITICAL, "psql.OpenPSQL error:", err)
		return
	}
	// if not valid, try to create
	if !valid {
		mlg.LogPrint(NOTICE, "uninitialized PSQL db, attempting to initialize")

		db.InitDB()

		// revalidate
		valid, err = db.IsValidDB()
		if err != nil {
			mlg.LogPrintln(CRITICAL, "second psql.OpenPSQL error:", err)
			return
		}
		if !valid {
			mlg.LogPrintln(CRITICAL, "psql.IsValidDB failed second validation")
			return
		}
	}

	err = db.CheckVersion()
	if err != nil {
		mlg.LogPrintln(CRITICAL, "psql.CheckVersion: ", err)
		return
	}

	altthm := altthumber.AltThumber(di.DemoAltThumber{})

	dbib, err := psqlib.NewPSQLIB(psqlib.Config{
		DB:         &db,
		Logger:     &lgr,
		SrcCfg:     &fstore.Config{"_demo/demoib0/src"},
		ThmCfg:     &fstore.Config{"_demo/demoib0/thm"},
		NNTPFSCfg:  &fstore.Config{"_demo/demoib0/nntp"},
		AltThumber: &altthm,
	})
	if err != nil {
		mlg.LogPrintln(CRITICAL, "psqlib.NewPSQLIB error:", err)
		return
	}

	valid, err = dbib.CheckIb0()
	if err != nil {
		mlg.LogPrintln(CRITICAL, "psqlib.CheckIb0:", err)
		return
	}
	if !valid {
		mlg.LogPrint(NOTICE, "uninitialized PSQLIB db, attempting to initialize")

		dbib.InitIb0()

		valid, err = dbib.CheckIb0()
		if err != nil {
			mlg.LogPrintln(CRITICAL, "second psqlib.CheckIb0:", err)
			return
		}
		if !valid {
			mlg.LogPrintln(CRITICAL, "psqlib.CheckIb0 failed second validation")
			return
		}
	}

	srv := nntp.NewNNTPServer(dbib, lgr)

	var proto, host string
	u, e := url.ParseRequestURI(*nntpbind)
	if e == nil {
		proto, host = u.Scheme, u.Host
	} else {
		proto, host = "tcp", *nntpbind
	}

	// graceful shutdown by signal
	killc := make(chan os.Signal, 2)
	signal.Notify(killc, os.Interrupt, syscall.SIGTERM)
	go func(c chan os.Signal) {
		for {
			s := <-c
			switch s {
			case os.Interrupt, syscall.SIGTERM:
				signal.Reset(os.Interrupt, syscall.SIGTERM)
				fmt.Fprintf(os.Stderr, "killing server\n")
				if srv != nil {
					srv.Close()
				}
				return
			}
		}
	}(killc)

	mlg.LogPrintf(
		NOTICE, "starting nntp server on proto(%s) host(%s)", proto, host)
	err = srv.ListenAndServe(proto, host, nntp.ListenParam{})
	if err != nil {
		mlg.LogPrintf(ERROR, "ListenAndServe returned: %v", err)
	}
}
