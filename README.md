# seslog2

`seslog2` is nginx syslog server.
It collects nginx's access logs and write them to ClickHouse DB.

## Features
* Only the data that is declared is written (apart from a couple of special fields)
* Data is written to a table defined by a tag in nginx
* Tables and fields in ClickHouse are created automatically
* You can specify the TTL for the table directly in the tag
* You can write to a table with the Null engine, also specified in the tag
* You can specify data types for fields yourself, or defaults for known variables will work, in extreme cases, data will be written as strings
* The main concept is preserved - the user of the service needs to configure only nginx
* Data in ClickHouse is written as a batch
* The data is saved to the hard disk if it could not be written to the database (write_on_fail)

## Make
```bash
make build
```

## Run
```bash
cp seslog.example.json seslog.json
build/seslog-server
```

## Install as systemd service
```bash
make build
sudo make install
```

## Install to nginx
Define the data you want to write to the database.
See file `examples/log_format/test1.json` for an example.
Use `seslog-logformatter` to create nginx log_format file:
```bash
build/seslog-logformatter examples/log_format/test1.json > /etc/nginx/test1_log_format
```

Include new `log_format` to your nginx config (http section)
```
http {
    ...
    include test1_log_format;
    ...
}
```

Then add access_log (preferred section)  
```
access_log syslog:server=<YOUR_SESLOG_IP>:5514,tag=<YOUR_TAG> test1_log_format;
```

For example:
```
access_log syslog:server=127.0.0.1:5514,tag=php_admin_panel test1_log_format if=$sesloggable;
```

## Tips
Please use `$loggable` (or anything like that) variable to avoid useless logging

e.g. (http context):
```
map $request_uri $sesloggable {
    default                                             1;
    ~*\.(ico|css|js|gif|jpg|jpeg|png|svg|woff|ttf|eot)$ 0;
}
```

## SELECT data
Select TOP-30 Referrer Domains
```clickhouse
SELECT
    domain(http_referer) AS ref_domain,
    count() AS cnt
FROM php_admin_panel
WHERE date = today()
GROUP BY ref_domain
ORDER BY cnt DESC
LIMIT 30
```

## Nginx tag
Nginx tag is used to define the table in which your data will be written.
However, using the tag, you can define some table options in the database.
### Special suffixes
* no suffix - default settings. `MergeTree` engine, 60 day TTL
* `_null` - `Null` engine, 0 TTL
* `_array` - Keep data as two arrays of strings (keys, vals), not implemented now
* `_ttl_<some_ttl_here>` - `MergeTree` engine, <some_ttl_here> TTL

Examples:
```
access_log syslog:server=127.0.0.1:5514,tag=table1 table1_log_format if=$sesloggable;
access_log syslog:server=127.0.0.1:5514,tag=table1_null table1_log_format if=$sesloggable;
access_log syslog:server=127.0.0.1:5514,tag=table1_ttl_3month table1_log_format if=$sesloggable;
```