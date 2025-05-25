package daemon

import (
	"context"
	"net"
	"time"

	"github.com/smirzaei/dnssync/internal/cli"
	"github.com/smirzaei/dnssync/internal/http"
	"github.com/smirzaei/dnssync/internal/ip"
	"github.com/smirzaei/dnssync/internal/metrics"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Daemon struct {
	l          *zap.Logger
	args       cli.Args
	ipLookup   *ip.IPLookup
	ipUpdater  *ip.IPUpdater
	appMetrics *metrics.AppMetrics
	httpServer *http.HTTPServer
}

func NewDaemon(logger *zap.Logger, args cli.Args) (*Daemon, error) {
	ipLookup := ip.NewIPLookup(logger)
	ipUpdater, err := ip.NewIPUpdater(logger, args.CloudflareApiKey, args.ZoneID)
	if err != nil {
		return nil, err
	}
	appMetrics := metrics.NewAppMetrics()
	httpConf := http.HTTPServerConfig{
		Port: int(args.HTTPPort),
	}
	httpServer := http.NewHTTPServer(logger, httpConf, appMetrics)

	d := Daemon{
		l:          logger,
		args:       args,
		ipLookup:   ipLookup,
		ipUpdater:  ipUpdater,
		appMetrics: appMetrics,
		httpServer: httpServer,
	}

	return &d, nil
}

func (d *Daemon) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if err := d.httpServer.Run(gCtx); err != nil {
			d.l.Error("HTTP server failed", zap.Error(err))
			return err
		}
		d.l.Info("HTTP server stopped")
		return nil
	})

	g.Go(func() error {
		if err := d.runUpdater(gCtx); err != nil {
			d.l.Error("IP updater failed", zap.Error(err))
			return err
		}
		d.l.Info("IP updater stopped")
		return nil
	})

	d.l.Info("daemon and http server running, waiting for group")
	if err := g.Wait(); err != nil {
		if err != context.Canceled && err != context.DeadlineExceeded {
			d.l.Error("errgroup encountered an error", zap.Error(err))
		}
		return err
	}

	d.l.Info("daemon shut down gracefully")
	return nil
}

func (d *Daemon) runUpdater(ctx context.Context) error {
	updateInterval := time.Second * time.Duration(d.args.Interval)
	d.l.Info("starting IP update loop", zap.Duration("update_interval", updateInterval))

	var currentIP net.IP

	// Initial IP check and update
	lookupTime := time.Now()
	lookedUpIP, err := d.ipLookup.LookupPublicIP(ctx)
	if err != nil {
		d.l.Error("initial public ip lookup failure", zap.Error(err))
		d.appMetrics.IncIPCheckTotal(metrics.IPCheckStatusFailure)
		d.appMetrics.ObserveIPCheckDuration(time.Since(lookupTime), metrics.IPCheckStatusFailure)
	} else {
		d.l.Info("initial ip lookup success", zap.String("ip", lookedUpIP.String()))
		d.appMetrics.IncIPCheckTotal(metrics.IPCheckStatusSuccess)
		d.appMetrics.ObserveIPCheckDuration(time.Since(lookupTime), metrics.IPCheckStatusSuccess) // Placeholder
		d.appMetrics.UpdateCurrentIP(lookedUpIP)

		updateStartTime := time.Now()
		err = d.ipUpdater.UpdateIP(ctx, d.args.DNSRecord, lookedUpIP)
		updateDuration := time.Since(updateStartTime)
		var updateStatus metrics.IPUpdateStatus
		if err != nil {
			d.l.Error("initial failed to update ip address", zap.Error(err))
			updateStatus = metrics.IPUpdateStatusFailure
		} else {
			d.l.Info("initial successfully updated the new ip", zap.String("ip", lookedUpIP.String()))
			updateStatus = metrics.IPUpdateStatusSuccess
			currentIP = lookedUpIP
		}
		d.appMetrics.IncIPUpdateTotal(updateStatus)
		d.appMetrics.ObserveIPUpdateDuration(updateDuration, updateStatus)
	}

	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.l.Info("IP update loop shutting down due to context cancellation")
			return ctx.Err()
		case tickTime := <-ticker.C:
			d.l.Debug("tick received, performing IP check", zap.Time("tick_time", tickTime))

			checkStartTime := time.Now()
			newLookedUpIP, err := d.ipLookup.LookupPublicIP(ctx)
			checkDuration := time.Since(checkStartTime)
			if err != nil {
				if ctx.Err() != nil {
					d.l.Info("IP lookup aborted due to context cancellation")
					return ctx.Err()
				}
				d.l.Error("public ip lookup failure", zap.Error(err))
				d.appMetrics.IncIPCheckTotal(metrics.IPCheckStatusFailure)
				d.appMetrics.ObserveIPCheckDuration(checkDuration, metrics.IPCheckStatusFailure)
				continue
			}

			d.l.Debug("ip lookup success", zap.String("ip", newLookedUpIP.String()))

			d.appMetrics.IncIPCheckTotal(metrics.IPCheckStatusSuccess)
			d.appMetrics.ObserveIPCheckDuration(checkDuration, metrics.IPCheckStatusSuccess)
			d.appMetrics.UpdateCurrentIP(newLookedUpIP)

			if net.IP.Equal(currentIP, newLookedUpIP) {
				d.l.Debug("ip hasn't changed from last known DNS record IP", zap.String("current_dns_ip", currentIP.String()))
				continue
			}

			d.l.Info("new public ip detected", zap.String("old_dns_ip", currentIP.String()), zap.String("new_ip", newLookedUpIP.String()))

			updateStartTime := time.Now()
			err = d.ipUpdater.UpdateIP(ctx, d.args.DNSRecord, newLookedUpIP)
			updateDuration := time.Since(updateStartTime)

			if err != nil {
				d.l.Error("failed to update ip address in DNS", zap.Error(err))
				d.appMetrics.IncIPUpdateTotal(metrics.IPUpdateStatusFailure)
				d.appMetrics.ObserveIPUpdateDuration(updateDuration, metrics.IPUpdateStatusFailure)
				continue
			}

			d.l.Info("successfully updated DNS record with new ip", zap.String("ip", newLookedUpIP.String()))
			d.appMetrics.IncIPUpdateTotal(metrics.IPUpdateStatusSuccess)
			d.appMetrics.ObserveIPUpdateDuration(updateDuration, metrics.IPUpdateStatusSuccess)
			currentIP = newLookedUpIP
		}
	}

}
