package daemon

import (
	"context"
	"time"

	"github.com/smirzaei/dnssync/internal/cli"
	"github.com/smirzaei/dnssync/internal/ip"
	"go.uber.org/zap"
)

type Daemon struct {
	l         *zap.Logger
	args      cli.Args
	ipLookup  *ip.IPLookup
	ipUpdater *ip.IPUpdater
}

func NewDaemon(logger *zap.Logger, args cli.Args) (*Daemon, error) {
	ipLookup := ip.NewIPLookup(logger)
	ipUpdater, err := ip.NewIPUpdater(logger, args.CloudflareApiKey, args.ZoneID)
	if err != nil {
		return nil, err
	}

	d := Daemon{
		l:         logger,
		args:      args,
		ipLookup:  ipLookup,
		ipUpdater: ipUpdater,
	}

	return &d, nil
}

func (d *Daemon) Run(ctx context.Context) error {
	updateInterval := time.Second * time.Duration(d.args.Interval)
	d.l.Info("updating in the background", zap.Duration("interval", updateInterval))

	for {
		select {
		case <-ctx.Done():
			d.l.Info("received exit signal")
			return nil
		case <-time.After(time.Duration(d.args.Interval) * time.Second):
			d.l.Debug("look up ip")
			ip, err := d.ipLookup.LookupPublicIP(ctx)
			if err != nil {
				d.l.Error("public ip lookup failure", zap.Error(err))
				continue
			}

			d.l.Debug("ip lookup success", zap.String("ip", ip.String()))
		}
	}
}
