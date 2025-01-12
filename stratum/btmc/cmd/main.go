package main

import (
	"context"
	"math/big"
	"strconv"
	"time"

	"database/sql"
	pb "github.com/bytom/btmcpool/common/format/generated"
	"github.com/bytom/btmcpool/common/logger"
	"github.com/bytom/btmcpool/common/rpc/hostprovider"
	"github.com/bytom/btmcpool/common/rpc/http"
	"github.com/bytom/btmcpool/common/service"
	"github.com/bytom/btmcpool/common/vars"
	ss "github.com/bytom/btmcpool/stratum"
	"github.com/bytom/btmcpool/stratum/btmc"
	_ "github.com/go-sql-driver/mysql"
)

func addRecord(db *sql.DB, w ss.WriterRec) error {
	head := "INSERT INTO shares (timestamp, account, name, block) VALUES ("
	timestamp := "FROM_UNIXTIME(" + strconv.FormatInt(time.Now().Unix(), 10) + ")"
	tail := ",'" + w.Account + "', '" + w.Name + "'," + strconv.Itoa(w.Block) + ");"
	query := head + timestamp + tail
	insert, err := db.Query(query)
	defer insert.Close()
	if err != nil {
		return err
	}
	return nil
}

func daemon(db *sql.DB, c chan ss.WriterRec) {
	for s := range c {
		addRecord(db, s)
	}
}

func main() {
	vars.Init()

	stratumId := vars.GetInt("stratum.id", 0)
	service := service.New("stratum_btm"+"."+strconv.Itoa(stratumId), service.NewConfig(vars.GetString("mode", "")))

	maxConn := vars.GetInt("stratum.max_conn", 32768)
	// init connection controller
	connCtl := ss.NewConnCtl(
		vars.GetDuration("stratum.default_ban_period", 20*time.Minute),
		pb.CoinType_BTMC,
		vars.GetBool("ip.ban_enable", false),
		vars.GetInt("ip.max_throughput", 131072),
		vars.GetInt("ip.max_connection", 1000),
		vars.GetFloat64("ip.throughput_ratio", 1.2),
		vars.GetFloat64("ip.connection_ratio", 1.2),
		vars.GetStringSlice("ip.white_list", []string{}))
	user := vars.GetString("db.user", "user")
	password := vars.GetString("db.password", "password")
	dbName := vars.GetString("db.name", "pool")
	db, err := sql.Open("mysql", user+":"+password+"@tcp(127.0.0.1:3306)/"+dbName)
	if err != nil {
		logger.Error("can't connect to database")
		return
	}
	c := make(chan ss.WriterRec)
	go daemon(db, c)

	// init server global state
	state, err := ss.InitServerState(context.Background(), connCtl, stratumId, uint(maxConn), c)
	if err != nil {
		logger.Error("can't create server state")
		return
	}

	// configuration node & verifier
	node := vars.GetString("node.name", "btmc_testnet")
	nodeUrl := vars.GetString("node.url", "http://127.0.0.1:9888")
	hostprovider.InitStaticProvider(map[string][]string{node: {nodeUrl}})
	http.Init(time.Second)

	syncer, err := btmc.NewBtmcNodeSyncer(node, nodeUrl)
	if err != nil {
		logger.Error("can't create node syncer", "error", err)
		return
	}

	verifier, err := btmc.NewBtmcVerifier(state)
	if err != nil {
		logger.Error("can't create verifier", "error", err)
		return
	}

	// create btmSessionData obj
	dataBuilder := btmc.NewBtmcSessionDataBuilder(uint64(state.GetId()), maxConn)

	// create diffAdjust
	diffAdjust := ss.NewDiffAdjust(big.NewInt(vars.GetInt64("session.diff", 500000)))

	// start server
	if err := ss.NewServer(
		vars.GetInt("stratum.port", 8118),
		maxConn,
		state,
		syncer,
		vars.GetDuration("node.sync_interval", 100*time.Millisecond), // sync interval
		verifier,
		vars.GetDuration("session.timeout", 5*time.Minute),
		vars.GetDuration("session.sched_interval", 0),
		dataBuilder,
		diffAdjust,
		btmc.NewBtmDecoder(),
	); err != nil {
		logger.Error("can't create server", "error", err)
		return
	}

	service.Run(":" + vars.GetString("service.port", "8082"))
}
