# smtpRelay
small SMTP relay

[![Go Report Card](https://goreportcard.com/badge/github.com/Supme/smtpRelay)](https://goreportcard.com/report/github.com/Supme/smtpRelay)

[Download binary](https://github.com/Supme/smtpRelay/releases)

example config.ini
``` $ini
# Db for queue emails
QueueDbDialect = "mssql"
QueueDbConnect = "server=127.0.0.1;user id=SmtpRelayLog;password=pa55w0rd;database=SmtpRelayLog;connection timeout=300"

# Db for log
StatusDbDialect = "mssql"
StatusDbConnect = "server=127.0.0.1;user id=SmtpRelayLog;password=pa55w0rd;database=SmtpRelayLog;connection timeout=300"

Hostname = "localhost" # hostname for HELO, if blank, then get mashine hostname

AllowIP = ["127.0.0.1", "192.168.1.20"] # allow sender IP

AllowDomains = ["dmbasis.ru"] # allow send email domain

SMTPListenAddr = ":2500" # SMTP server listen on (example: "127.0.0.1:2500")

SendStream = 10
ResendStream = 2

MaxRepeatSend = 2
RepeatIntervalMinutes = 5

Debug = false
```

execute with default config file ./config ini
```
./smtpRelay
```

or change config file path
```
./smtpRelay -c /path/to/youconfig.ini
```

For run as service in Windows use [NSSM](http://nssm.cc/download)



in log database create table "statuses" structure

| id | queued_at | sending_at | from | rcpt | message_type | message_id | status |
|----|:-----------:|:------------:|:------:|:------:|:------------:|:------------:|--------:|

message_type is string from "X-Postmaster-Msgtype" header

message_id is string from "Message-Id" header

in queue database create table "queues" structure

| id | created_at | updated_at | message_type | message_id | from | from_hostname | rcpt   | rcpt_hostname | data | repeat | later_status|
|----|:-----------:|:------------:|:------:|:------:|:------:|:------------:|:------------:|:------------:|:------------:|:------------:|--------:|
