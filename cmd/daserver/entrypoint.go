package main

import (
	"fmt"

	"github.com/urfave/cli/v2"

	plasma "github.com/Layr-Labs/op-plasma-eigenda"
	"github.com/Layr-Labs/op-plasma-eigenda/eigenda"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/opio"
)

func StartDAServer(cliCtx *cli.Context) error {
	if err := CheckRequired(cliCtx); err != nil {
		return err
	}

	cfg := ReadCLIConfig(cliCtx)
	if err := cfg.Check(); err != nil {
		return err
	}

	logCfg := oplog.ReadCLIConfig(cliCtx)

	l := oplog.NewLogger(oplog.AppOut(cliCtx), logCfg)
	oplog.SetGlobalLogHandler(l.Handler())

	l.Info("Initializing Plasma DA server...")

	var store plasma.PlasmaStore

	if cfg.FileStoreEnabled() {
		l.Info("Using file storage", "path", cfg.FileStoreDirPath)
		store = NewFileStore(cfg.FileStoreDirPath)
	} else if cfg.S3Enabled() {
		l.Info("Using S3 storage", "bucket", cfg.S3Bucket)
		s3, err := NewS3Store(cliCtx.Context, cfg.S3Bucket)
		if err != nil {
			return fmt.Errorf("failed to create S3 store: %w", err)
		}
		store = s3
	} else if cfg.EigenDAEnabled() {
		l.Info("Using EigenDA storage", "RPC", cfg.EigenDAConfig.RPC)
		eigenda, err := NewEigenDAStore(
			cliCtx.Context,
			eigenda.NewEigenDAClient(
				l,
				cfg.EigenDAConfig,
			),
		)
		if err != nil {
			return fmt.Errorf("failed to create EigenDA store: %w", err)
		}
		store = eigenda
	}

	server := plasma.NewDAServer(cliCtx.String(ListenAddrFlagName), cliCtx.Int(PortFlagName), store, l)

	if err := server.Start(); err != nil {
		return fmt.Errorf("failed to start the DA server")
	} else {
		l.Info("Started DA Server")
	}

	defer func() {
		if err := server.Stop(); err != nil {
			l.Error("failed to stop DA server", "err", err)
		}
	}()

	opio.BlockOnInterrupts()

	return nil
}