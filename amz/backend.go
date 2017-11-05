// Copyright 2014 The go-amazoncoin Authors
// This file is part of the go-amazoncoin library.
//
// The go-amazoncoin library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-amazoncoin library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-amazoncoin library. If not, see <http://www.gnu.org/licenses/>.

// Package eth implements the Amazoncoin protocol.
package eth

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/amazoncoinio/go-amazoncoin/accounts"
	"github.com/amazoncoinio/go-amazoncoin/common"
	"github.com/amazoncoinio/go-amazoncoin/common/hexutil"
	"github.com/amazoncoinio/go-amazoncoin/consensus"
	"github.com/amazoncoinio/go-amazoncoin/consensus/clique"
	"github.com/amazoncoinio/go-amazoncoin/consensus/ethash"
	"github.com/amazoncoinio/go-amazoncoin/core"
	"github.com/amazoncoinio/go-amazoncoin/core/bloombits"
	"github.com/amazoncoinio/go-amazoncoin/core/types"
	"github.com/amazoncoinio/go-amazoncoin/core/vm"
	"github.com/amazoncoinio/go-amazoncoin/amz/downloader"
	"github.com/amazoncoinio/go-amazoncoin/amz/filters"
	"github.com/amazoncoinio/go-amazoncoin/amz/gasprice"
	"github.com/amazoncoinio/go-amazoncoin/amzdb"
	"github.com/amazoncoinio/go-amazoncoin/event"
	"github.com/amazoncoinio/go-amazoncoin/internal/ethapi"
	"github.com/amazoncoinio/go-amazoncoin/log"
	"github.com/amazoncoinio/go-amazoncoin/miner"
	"github.com/amazoncoinio/go-amazoncoin/node"
	"github.com/amazoncoinio/go-amazoncoin/p2p"
	"github.com/amazoncoinio/go-amazoncoin/params"
	"github.com/amazoncoinio/go-amazoncoin/rlp"
	"github.com/amazoncoinio/go-amazoncoin/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
}

// Amazoncoin implements the Amazoncoin full node service.
type AmazonCoin struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan  chan bool    // Channel for shutting down the amazoncoin
	stopDbUpgrade func() error // stop chain db sequential key upgrade

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb amzdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	ApiBackend *EthApiBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	amazoncoinbase common.Address

	networkId     uint64
	netRPCService *ethapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and amazoncoinbase)
}

func (s *AmazonCoin) AddLesServer(ls LesServer) {
	s.lesServer = ls
}

// New creates a new Amazoncoin object (including the
// initialisation of the common Amazoncoin object)
func New(ctx *node.ServiceContext, config *Config) (*AmazonCoin, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run eth.AmazonCoin in light sync mode, use les.LightAmazonCoin")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	stopDbUpgrade := upgradeDeduplicateData(chainDb)
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	eth := &AmazonCoin{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, config, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		stopDbUpgrade:  stopDbUpgrade,
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		amazoncoinbase:      config.Amazoncoinbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks),
	}

	log.Info("Initialising Amazoncoin protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := core.GetBlockChainVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run geta upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	}

	vmConfig := vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
	eth.blockchain, err = core.NewBlockChain(chainDb, eth.chainConfig, eth.engine, vmConfig)
	if err != nil {
		return nil, err
	}
	// Rewind the chain in case of an incompatible config upgrade.
	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		eth.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	eth.bloomIndexer.Start(eth.blockchain.CurrentHeader(), eth.blockchain.SubscribeChainEvent)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	eth.txPool = core.NewTxPool(config.TxPool, eth.chainConfig, eth.blockchain)

	if eth.protocolManager, err = NewProtocolManager(eth.chainConfig, config.SyncMode, config.NetworkId, eth.eventMux, eth.txPool, eth.engine, eth.blockchain, chainDb); err != nil {
		return nil, err
	}
	eth.miner = miner.New(eth, eth.chainConfig, eth.EventMux(), eth.engine)
	eth.miner.SetExtra(makeExtraData(config.ExtraData))

	eth.ApiBackend = &EthApiBackend{eth, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	eth.ApiBackend.gpo = gasprice.NewOracle(eth.ApiBackend, gpoParams)

	return eth, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"geta",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (amzdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*amzdb.LDBDatabase); ok {
		db.Meter("eth/db/chaindata/")
	}
	return db, nil
}

// CreateConsensusEngine creates the required type of consensus engine instance for an Amazoncoin service
func CreateConsensusEngine(ctx *node.ServiceContext, config *Config, chainConfig *params.ChainConfig, db amzdb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	if chainConfig.Clique != nil {
		return clique.New(chainConfig.Clique, db)
	}
	// Otherwise assume proof-of-work
	switch {
	case config.PowFake:
		log.Warn("Ethash used in fake mode")
		return ethash.NewFaker()
	case config.PowTest:
		log.Warn("Ethash used in test mode")
		return ethash.NewTester()
	case config.PowShared:
		log.Warn("Ethash used in shared mode")
		return ethash.NewShared()
	default:
		engine := ethash.New(ctx.ResolvePath(config.EthashCacheDir), config.EthashCachesInMem, config.EthashCachesOnDisk,
			config.EthashDatasetDir, config.EthashDatasetsInMem, config.EthashDatasetsOnDisk)
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs returns the collection of RPC services the Amazoncoin package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *AmazonCoin) APIs() []rpc.API {
	apis := ethapi.GetAPIs(s.ApiBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicAmazonCoinAPI(s),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "eth",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *AmazonCoin) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *AmazonCoin) Amazoncoinbase() (eb common.Address, err error) {
	s.lock.RLock()
	amazoncoinbase := s.amazoncoinbase
	s.lock.RUnlock()

	if amazoncoinbase != (common.Address{}) {
		return amazoncoinbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			return accounts[0].Address, nil
		}
	}
	return common.Address{}, fmt.Errorf("amazoncoinbase address must be explicitly specified")
}

// set in js console via admin interface or wrapper from cli flags
func (self *AmazonCoin) SetAmazoncoinbase(amazoncoinbase common.Address) {
	self.lock.Lock()
	self.amazoncoinbase = amazoncoinbase
	self.lock.Unlock()

	self.miner.SetAmazoncoinbase(amazoncoinbase)
}

func (s *AmazonCoin) StartMining(local bool) error {
	eb, err := s.Amazoncoinbase()
	if err != nil {
		log.Error("Cannot start mining without amazoncoinbase", "err", err)
		return fmt.Errorf("amazoncoinbase missing: %v", err)
	}
	if clique, ok := s.engine.(*clique.Clique); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("Amazoncoinbase account unavailable locally", "err", err)
			return fmt.Errorf("singer missing: %v", err)
		}
		clique.Authorize(eb, wallet.SignHash)
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so noone will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(eb)
	return nil
}

func (s *AmazonCoin) StopMining()         { s.miner.Stop() }
func (s *AmazonCoin) IsMining() bool      { return s.miner.Mining() }
func (s *AmazonCoin) Miner() *miner.Miner { return s.miner }

func (s *AmazonCoin) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *AmazonCoin) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *AmazonCoin) TxPool() *core.TxPool               { return s.txPool }
func (s *AmazonCoin) EventMux() *event.TypeMux           { return s.eventMux }
func (s *AmazonCoin) Engine() consensus.Engine           { return s.engine }
func (s *AmazonCoin) ChainDb() amzdb.Database            { return s.chainDb }
func (s *AmazonCoin) IsListening() bool                  { return true } // Always listening
func (s *AmazonCoin) EthVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *AmazonCoin) NetVersion() uint64                 { return s.networkId }
func (s *AmazonCoin) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *AmazonCoin) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// Amazoncoin protocol implementation.
func (s *AmazonCoin) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		maxPeers -= s.config.LightPeers
		if maxPeers < srvr.MaxPeers/2 {
			maxPeers = srvr.MaxPeers / 2
		}
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Amazoncoin protocol.
func (s *AmazonCoin) Stop() error {
	if s.stopDbUpgrade != nil {
		s.stopDbUpgrade()
	}
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
