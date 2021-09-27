# unifi dream machine pro iptables log rules

A simple Go application to modify the iptables rules on a Unifi Dream Machine Pro to enable dropped packet logging.

## Description

The Unifi Dream Machine Pro (as of 2021-09-19) has woefully inadequate firewall rule logging through the built in controls.  It is possible to enable logging for custom firewall rules, however there is no useful tag information written with the packet info to syslog to identify which rule caused the packet to be logged, nor information about whether the packet was accepted, rejected, or dropped.  Beyond this, it doesn't appear possible to even _see_ what the default rules are, much less enable logging for them - without reverting to looking through iptables over ssh.  (If I'm missing something, please let me know - a native solution would have been a lot easier).


## What does this do

See the source code for details, but in general:

1) run `iptables-save` to get a dump of the current iptables rules
2) look for lines that are:
	a) in one of the USER tables - WAN_IN, WAN_OUT, WAN_LOCAL, LAN_IN, LAN_OUT, LAN_LOCAL, GUEST_IN, GUEST_OUT, GUEST_LOCAL
	b) have a `-j DROP` rule and the `--comment \d+` pattern.  (The comment pattern appears to be a unique identifier assigned by UDMP and is used for their internal tracking)
3) insert a new `-j LOG` rule immediately preceeding the `-j DROP` rule, unless it already exists.
4) run `iptables-restore` to refresh the iptables rules with the additional log rules.

## Persistence
In my testing, the updated log rules appear to be persistent, unless a change is made to the firewall in the UDMP UI.  This script will check for existing log rules, and not actually update the firewall rules if no changes are necessary, so it should be perfectly safe to run this every few minutes via cron to check if the log rules need to be restored. 

## Clean-Up

Rules can be cleaned up by either:

1) Running ./udmp-iptables-log-rules -d
2) Making a firewall change in the UDMP UI (and not running ./udmp-itables-log rules again)

## Why Go?

I'm very new to Go, but it provided extremely simple cross-compilation to arm64, and creates a single no-dependency binary that can be easily uploaded to the UDMP for execution.  I could have (and almost) wrote this in Python, and leveraged the Python3 implementation in the unifi-os container, but this was nearly as easy to implement and removed that dependency. 


## Installation

Download and SCP the arm64 binary file to /mnt/data/ on your UDMP.  
Make sure that the binary has execute permissions `chmod +x udmp-iptables-log-rules`

You'll likely be interested in https://github.com/boostchicken/udm-utilities/ to run this (or configure a cron task) on reboots.

Note - I run this manually to enable/disable the logs should I be looking for something - I'm not sure what the extra logging writes do to the overall wear on the built-in emmc storage on the UDMP.  Also, in a home network I'm just not that likely to go looking at historical firewall logs. Caveot emptor. 

## Usage

```
./udmp-iptables-log-rules
Usage of ./udmp-iptables-log-rules
  -d	delete - cleans up previously created rules
  -v	print version info
```

## Logging

It may be useful to view the output of this command as part of the syslog.  This is most easily accomplished by piping the output to the logger utility.

`./udmp-iptables-log-rules 2>&1 | logger -t udmp-iptables-log-rules -p user.info`

## Building

`./build.sh`

## Buy me a beer

If you find this useful and feel so inclinced, https://paypal.me/slynn1324.  Otherwise, simply enjoy.  
