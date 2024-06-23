package cli

import "github.com/alexflint/go-arg"

type Args struct {
	Interval    uint   `arg:"required" help:"How often sync the IP (in seconds)"`
	ZoneID      string `arg:"-z,--zone-id,required" help:"The zone ID"`
	DNSRecordID string `arg:"-d,--dns-record,required" help:"ID of the DNS record to update"`
	Verbose     bool   `arg:"-v,--verbose"`
}

func ParseArgs() Args {
	var args Args
	arg.MustParse(&args)

	return args
}
