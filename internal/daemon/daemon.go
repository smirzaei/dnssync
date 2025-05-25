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
	// TODO: start the HTTP server

	updateInterval := time.Second * time.Duration(d.args.Interval)
	d.l.Info("starting daemon", zap.Duration("update_interval", updateInterval))

	var currentIP net.IP

	lookedUpIP, err := d.ipLookup.LookupPublicIP(ctx)
	lookupTime := time.Now()
	if err != nil {
		d.l.Error("initial public ip lookup failure", zap.Error(err))
		d.appMetrics.IncIPCheckTotal(metrics.IPCheckStatusFailure)
		d.appMetrics.ObserveIPCheckDuration(time.Since(lookupTime), metrics.IPCheckStatusFailure)
	} else {
		d.l.Info("initial ip lookup success", zap.String("ip", lookedUpIP.String()))
		d.appMetrics.IncIPCheckTotal(metrics.IPCheckStatusSuccess)
		d.appMetrics.ObserveIPCheckDuration(time.Since(lookupTime), metrics.IPCheckStatusSuccess)
		d.appMetrics.UpdateCurrentIP(lookedUpIP)

		// Attempt to set this as the current DNS IP
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
			d.l.Info("received exit signal, daemon shutting down")
			return nil
		case tickTime := <-ticker.C:
			d.l.Debug("tick received, performing IP check", zap.Time("tick_time", tickTime))

			checkStartTime := time.Now()
			newLookedUpIP, err := d.ipLookup.LookupPublicIP(ctx)
			checkDuration := time.Since(checkStartTime)
			if err != nil {
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
