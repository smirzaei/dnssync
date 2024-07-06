DNS Sync
===

A very simple utility to sync your public IP address with a DNS record in Cloudflare.

### Usage

Example:

```
 dnssync --interval 60 --zone-id $ZONE_ID --dns-record foo.example.com --api-key $API_KEY
```

Help:

```
Usage: dnssync --interval INTERVAL --zone-id ZONE-ID --dns-record DNS-RECORD --api-key API-KEY [--verbose]

Options:
  --interval INTERVAL    How often sync the IP (in seconds)
  --zone-id ZONE-ID, -z ZONE-ID
                         The zone ID. Ref: https://developers.cloudflare.com/fundamentals/setup/find-account-and-zone-ids/
  --dns-record DNS-RECORD, -r DNS-RECORD
                         The DNS record name. E.g. foo.example.com
  --api-key API-KEY, -k API-KEY
                         Cloudflare API key. Ref: https://dash.cloudflare.com/profile/api-tokens
  --verbose, -v
  --help, -h             display this help and exit
```

* Find your zone-id [here](https://developers.cloudflare.com/fundamentals/setup/find-account-and-zone-ids/).
* [Create API token](https://developers.cloudflare.com/fundamentals/api/get-started/create-token/):

### License

MIT
