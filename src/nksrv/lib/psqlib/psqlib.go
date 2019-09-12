package psqlib

// psql imageboard module

import (
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/crypto/ed25519"

	"nksrv/lib/altthumber"
	"nksrv/lib/cacheengine"
	"nksrv/lib/fstore"
	. "nksrv/lib/logx"
	"nksrv/lib/mail/form"
	"nksrv/lib/nilthumbnailer"
	"nksrv/lib/psql"
	"nksrv/lib/thumbnailer"
	"nksrv/lib/webcaptcha"
)

type PSQLIB struct {
	db                 psql.PSQL
	log                Logger
	src                fstore.FStore
	thm                fstore.FStore
	nntpfs             fstore.FStore
	nntpce             cacheengine.CacheEngine
	thumbnailer        thumbnailer.Thumbnailer
	tplan_thread       thumbnailer.ThumbPlan
	tplan_reply        thumbnailer.ThumbPlan
	tplan_sage         thumbnailer.ThumbPlan
	altthumb           altthumber.AltThumber
	ffo                formFileOpener
	fpp                form.ParserParams
	instance           string
	maxArticleBodySize int64
	webcaptcha         *webcaptcha.WebCaptcha
	webFrontendKey     ed25519.PrivateKey

	ngp_global    newGroupPolicy
	ngp_anypuller newGroupPolicy
	ngp_anyserver newGroupPolicy

	st_prep [st_max]*sql.Stmt

	// newthread prepared statements and locking
	ntStmts map[int]*sql.Stmt
	ntMutex sync.RWMutex

	// newpost prepared statements and locking
	npStmts map[npTuple]*sql.Stmt
	npMutex sync.RWMutex

	puller_nonce int64
}

type Config struct {
	DB             *psql.PSQL
	Logger         *LoggerX
	NodeName       string
	WebFrontendKey string
	SrcCfg         *fstore.Config
	ThmCfg         *fstore.Config
	NNTPFSCfg      *fstore.Config
	TBuilder       thumbnailer.ThumbnailerBuilder
	TCfgThread     *thumbnailer.ThumbConfig
	TCfgReply      *thumbnailer.ThumbConfig
	TCfgSage       *thumbnailer.ThumbConfig
	AltThumber     *altthumber.AltThumber
	WebCaptcha     *webcaptcha.WebCaptcha
	NGPGlobal      string
	NGPAnyPuller   string
	NGPAnyServer   string
}

// readonly for now

func nonEmptyStrOrPanic(s string) string {
	if s == "" {
		panic("empty string")
	}
	return s
}

func NewPSQLIB(cfg Config) (p *PSQLIB, err error) {
	p = new(PSQLIB)

	st_once.Do(loadStatements)
	if st_loaderr != nil {
		return nil, st_loaderr
	}

	p.log = NewLogToX(*cfg.Logger, fmt.Sprintf("psqlib.%p", p))

	p.db = *cfg.DB

	p.src, err = fstore.OpenFStore(*cfg.SrcCfg)
	if err != nil {
		return nil, err
	}
	//p.src.CleanTemp()

	p.thm, err = fstore.OpenFStore(*cfg.ThmCfg)
	if err != nil {
		return nil, err
	}
	//p.thm.CleanTemp()

	p.nntpfs, err = fstore.OpenFStore(*cfg.NNTPFSCfg)
	if err != nil {
		return nil, err
	}
	//p.nntpfs.RemoveDir(nntpIncomingTempDir)
	if err = p.nntpfs.MakeDir(nntpIncomingDir); err != nil {
		return nil, err
	}
	if err = p.nntpfs.MakeDir(nntpPullerDir); err != nil {
		return nil, err
	}

	if cfg.TBuilder != nil {

		p.thumbnailer, err = cfg.TBuilder.BuildThumbnailer(&p.thm)
		if err != nil {
			return nil, err
		}

		p.tplan_thread = thumbnailer.ThumbPlan{
			Name:        "t",
			ThumbConfig: *cfg.TCfgThread,
		}
		p.tplan_reply = thumbnailer.ThumbPlan{
			Name:        "r",
			ThumbConfig: *cfg.TCfgReply,
		}
		if cfg.TCfgSage != nil {
			p.tplan_sage = thumbnailer.ThumbPlan{
				Name:        "s",
				ThumbConfig: *cfg.TCfgSage,
			}
		} else {
			p.tplan_sage = p.tplan_reply
		}

	} else {
		p.thumbnailer = nilthumbnailer.NilThumbnailer{}
	}

	p.nntpce = cacheengine.NewCacheEngine(nntpcachemgr{p})

	p.altthumb = *cfg.AltThumber

	p.ffo = formFileOpener{&p.src}

	p.instance = nonEmptyStrOrPanic(cfg.NodeName)
	if cfg.WebFrontendKey != "" {
		seed, e := hex.DecodeString(cfg.WebFrontendKey)
		if e != nil {
			panic("bad web frontend key")
		}
		p.webFrontendKey = ed25519.NewKeyFromSeed(seed)
	}

	p.fpp = form.DefaultParserParams
	// TODO make configurable
	p.fpp.MaxFileCount = 1000
	p.fpp.MaxFileAllSize = 1 << 30

	p.maxArticleBodySize = (2 << 30) - 1 // TODO config

	p.webcaptcha = cfg.WebCaptcha

	p.ngp_global, err = makeNewGroupPolicy(cfg.NGPGlobal)
	if err != nil {
		return
	}
	p.ngp_anyserver, err = makeNewGroupPolicy(cfg.NGPAnyServer)
	if err != nil {
		return
	}
	p.ngp_anyserver, err = makeNewGroupPolicy(cfg.NGPAnyPuller)
	if err != nil {
		return
	}

	p.ntStmts = make(map[int]*sql.Stmt)
	p.npStmts = make(map[npTuple]*sql.Stmt)

	return
}

func (sp *PSQLIB) Prepare() (err error) {
	err = sp.prepareStatements()
	if err != nil {
		return
	}

	return
}

func (dbib *PSQLIB) InitAndPrepare() (err error) {
	valid, err := dbib.CheckDB()
	if err != nil {
		return fmt.Errorf("error checking: %v", err)
	}
	if !valid {
		dbib.log.LogPrint(NOTICE,
			"uninitialized db, attempting to initialize")

		err = dbib.InitDB()
		if err != nil {
			return fmt.Errorf("error initializing: %v", err)
		}

		valid, err = dbib.CheckDB()
		if err != nil {
			return fmt.Errorf("error checking (2): %v", err)
		}
		if !valid {
			return errors.New("database still not valid after initialization")
		}
	}

	err = dbib.Prepare()
	if err != nil {
		return
	}

	return
}

func NewInitAndPrepare(cfg Config) (db *PSQLIB, err error) {
	db, err = NewPSQLIB(cfg)
	if err != nil {
		return
	}

	err = db.InitAndPrepare()
	if err != nil {
		return
	}

	return
}
