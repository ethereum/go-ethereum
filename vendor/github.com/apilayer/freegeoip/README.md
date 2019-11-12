![freegeoip ipstack](https://raw.githubusercontent.com/apilayer/freegeoip/master/freegeo-warning.png)

# freegeoip - Important Announcement

*[The old freegeoip API is now deprecated and will be discontinued on July 1st, 2018]*

Launched more than 6 years ago, the freegeoip.net API has grown into one of the biggest and most widely used APIs for IP to location services worldwide. The API is used by thousands of developers, SMBs and large corporations around the globe and is currently handling more than 2 billion requests per day. After years of operation and the API remaining almost unchanged, today we announce the complete re-launch of freegeoip into a faster, more advanced and more scalable API service called ipstack (https://ipstack.com). All users that wish to continue using our IP to location service will be required to sign up to obtain a free API access key and perform a few simple changes to their integration. While the new API offers the ability to return data in the same structure as the old freegeoip API, the new API structure offers various options of delivering much more advanced data for IP Addresses.

## Required Changes to Legacy Integrations (freegeoip.net/json/xml) 

As of March 31 2018 the old freegeoip API is deprecated and a completely re-designed API is now accessible at http://api.ipstack.com. While the new API offers the same capabilities as the old one and also has the option of returning data in the legacy format, the API URL has now changed and all users are required to sign up for a free API Access Key to use the service.

1. Get a free ipstack Account and Access Key

Head over to https://ipstack.com and follow the instructions to create your account and obtain your access token. If you only need basic IP to Geolocation data and do not require more than 10,000 requests per month, you can use the free account. If you'd like more advanced features or more requests than included in the free account you will need to choose one of the paid options. You can find an overview of all available plans at https://ipstack.com/product

2. Integrate the new API URL

The new API comes with a completely new endpoint (api.ipstack.com) and requires you to append your API Access Key to the URL as a GET parameter. For complete integration instructions, please head over to the API Documentation at https://ipstack.com/documentation. While the new API offers a completely reworked response structure with many additional data points, we also offer the option to receive results in the old freegeoip.net format in JSON or XML.

To receive your API results in the old freegeoip format, please simply append &legacy=1 to the new API URL. 

JSON Example: http://api.ipstack.com/186.116.207.169?access_key=YOUR_ACCESS_KEY&output=json&legacy=1

XML Example: http://api.ipstack.com/186.116.207.169?access_key=YOUR_ACCESS_KEY&output=xml&legacy=1

## New features with ipstack
While the new ipstack service now runs on a commercial/freemium model, we have worked hard at building a faster, more scalable, and more advanced IP to location API product. You can read more about all the new features by navigating to https://ipstack.com, but here's a list of the most important changes and additions:

- We're still free for basic usage

While we now offer paid / premium options for our more advanced users, our core product and IP to Country/Region/City product is still completely free of charge for up to 10,000 requests per month. If you need more advanced data or more requests, you can choose one of the paid plans listed at https://ipstack.com/product

-  Batch Requests

Need to validate more than 1 IP Address in a single API Call? Our new Bulk Lookup Feature (available on our paid plans) allows you to geolocate up to 50 IP Addresses in a single API Call.

- Much more Data

While the old freegeoip API was limited to provide only the most basic IP to location data, our new API provides more than 20 additional data points including Language, Time Zone, Current Time, Currencies, Connection & ASN Information, and much more. To learn more about all the data points available, please head over to the ipstack website.

- Security & Fraud Prevention Tools

Do you want to prevent fraudulent traffic from arriving at your website or from abusing your service? Easily spot malicious / proxy / VPN traffic by using our new Security Module, which outputs a lot of valuable security information about an IP Address.

Next Steps

- Deprecation of the old API

While we want to keep the disruption to our current users as minimal as possible, we are planning to shut the old API down on July 1st, 2018. This should give all users enough time to adapt to changes, and should we still see high volumes of traffic going to the old API by that date, we may decide to extend it further. In any case, we highly recommend you switch to the new API as soon as possible. We will keep you posted here about any changes to the planned shutdown date.

- Any Questions? Please get in touch!

It's very important to ensure a smooth transition to ipstack for all freegeoip API users. If you are a developer that has published a plugin/addon that includes the legacy API, we recommend you get in touch with us and also share this announcement with your users. If you have any questions about the transition or the new API, please get in touch with us at support@ipstack.com











# freegeoip - Deprecated Documentation

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy)

This is the source code of the freegeoip software. It contains both the web server that empowers freegeoip.net, and a package for the [Go](http://golang.org) programming language that enables any web server to support IP geolocation with a simple and clean API.

See http://en.wikipedia.org/wiki/Geolocation for details about geolocation.

Developers looking for the Go API can skip to the [Package freegeoip](#packagefreegeoip) section below.

## Running

This section is for people who desire to run the freegeoip web server on their own infrastructure. The easiest and most generic way of doing this is by using Docker. All examples below use Docker.

### Docker

#### Install Docker

Docker has [install instructions for many platforms](https://docs.docker.com/engine/installation/),
including
- [Ubuntu](https://docs.docker.com/engine/installation/linux/docker-ce/ubuntu/)
- [CentOS](https://docs.docker.com/engine/installation/linux/docker-ce/centos/)
- [Mac](https://docs.docker.com/docker-for-mac/install/)

#### Run the API in a container

```bash
docker run --restart=always -p 8080:8080 -d apilayer/freegeoip
```

#### Test

```bash
curl localhost:8080/json/1.2.3.4
# => {"ip":"1.2.3.4","country_code":"US","country_name":"United States", # ...
```

### Other Linux, OS X, FreeBSD, and Windows

There are [pre-compiled binaries](https://github.com/apilayer/freegeoip/releases) available.

### Production configuration

For production workloads you may want to use different configuration for the freegeoip web server, for example:

* Enabling the "internal server" for collecting metrics and profiling/tracing the freegeoip web server on demand
* Monitoring the internal server using [Prometheus](https://prometheus.io), or exporting your metrics to [New Relic](https://newrelic.com)
* Serving the freegeoip API over HTTPS (TLS) using your own certificates, or provisioned automatically using [LetsEncrypt.org](https://letsencrypt.org)
* Configuring [HSTS](https://en.wikipedia.org/wiki/HTTP_Strict_Transport_Security) to restrict your browser clients to always use HTTPS
* Configuring the read and write timeouts to avoid stale clients consuming server resources
* Configuring the freegeoip web server to read the client IP (for logs, etc) from the X-Forwarded-For header when running behind a reverse proxy
* Configuring [CORS](https://en.wikipedia.org/wiki/Cross-origin_resource_sharing) to restrict access to your API to specific domains
* Configuring a specific endpoint path prefix other than the default "/" (thus /json, /xml, /csv) to serve the API alongside other APIs on the same host
* Optimizing your round trips by enabling [TCP Fast Open](https://en.wikipedia.org/wiki/TCP_Fast_Open) on your OS and the freegeoip web server
* Setting up usage limits (quotas) for your clients (per client IP) based on requests per time interval; we support various backends such as in-memory map (for single instance), or redis or memcache for distributed deployments
* Serve the default [GeoLite2 City](http://dev.maxmind.com/geoip/geoip2/geolite2/) free database that is downloaded and updated automatically in background on a configurable schedule, or
* Serve the commercial [GeoIP2 City](https://www.maxmind.com/en/geoip2-city) database from MaxMind, either as a local file that you provide and update periodically (so the server can reload it), or configured to be downloaded periodically using your API key

See the [Server Options](#serveroptions) section below for more information on configuring the server.

For automation, check out the [freegeoip chef cookbook](https://supermarket.chef.io/cookbooks/freegeoip) or the (legacy) [Ansible Playbook](./cmd/freegeoip/ansible-playbook) for Ubuntu 14.04 LTS.

<a name="serveroptions">

### Server Options

To see all the available options, use the `-help` option:

```bash
docker run --rm -it apilayer/freegeoip -help
```

If you're using LetsEncrypt.org to provision your TLS certificates, you have to listen for HTTPS on port 443. Following is an example of the server listening on 3 different ports: metrics + pprof (8888), http (80), and https (443):

```bash
docker run -p 8888:8888 -p 80:8080 -p 443:8443 -d apilayer/freegeoip \
	-internal-server=:8888 \
	-http=:8080 \
	-https=:8443 \
	-hsts=max-age=31536000 \
	-letsencrypt \
	-letsencrypt-hosts=myfancydomain.io
```

 You can configure the freegeiop web server via command line flags or environment variables. The names of environment variables are the same for command line flags, but prefixed with FREEGEOIP, all upperscase, separated by underscores. If you want to use environment variables instead:

```bash
$ cat prod.env
FREEGEOIP_INTERNAL_SERVER=:8888
FREEGEOIP_HTTP=:8080
FREEGEOIP_HTTPS=:8443
FREEGEOIP_HSTS=max-age=31536000
FREEGEOIP_LETSENCRYPT=true
FREEGEOIP_LETSENCRYPT_HOSTS=myfancydomain.io

$ docker run --env-file=prod.env -p 8888:8888 -p 80:8080 -p 443:8443 -d apilayer/freegeoip
```

By default, HTTP/2 is enabled over HTTPS. You can disable by passing the `-http2=false` flag.

Also, the Docker image of freegeoip does not provide the web page from freegeiop.net, it only provides the API. If you want to serve that page, you can pass the `-public=/var/www` parameter in the command line. You can also tell Docker to mount that directory as a volume on the host machine and have it serve your own page, using Docker's `-v` parameter.

If the freegeoip web server is running behind a reverse proxy or load balancer, you have to run it passing the `-use-x-forwarded-for` parameter and provide the `X-Forwarded-For` HTTP header in all requests. This is for the freegeoip web server be able to log the client IP, and to perform geolocation lookups when an IP is not provided to the API, e.g. `/json/` (uses client IP) vs `/json/1.2.3.4`.

## Database

The current implementation uses the free [GeoLite2 City](http://dev.maxmind.com/geoip/geoip2/geolite2/) database from MaxMind.

In the past we had databases from other providers, and at some point even our own database comprised of data from different sources. This means it might change in the future.

If you have purchased the commercial database from MaxMind, you can point the freegeoip web server or (Go API, for dev) to the URL containing the file, or local file, and the server will use it.

In case of files on disk, you can replace the file with a newer version and the freegeoip web server will reload it automatically in background. If instead of a file you use a URL (the default), we periodically check the URL in background to see if there's a new database version available, then download the reload it automatically.

All responses from the freegeiop API contain the date that the database was downloaded in the X-Database-Date HTTP header.

## API

The freegeoip API is served by endpoints that encode the response in different formats.

Example:

```bash
curl freegeoip.net/json/
```

Returns the geolocation information of your own IP address, the source IP address of the connection.

You can pass a different IP or hostname. For example, to lookup the geolocation of `github.com` the server resolves the name first, then uses the first IP address available, which might be IPv4 or IPv6:

```bash
curl freegeoip.net/json/github.com
```

Same semantics are available for the `/xml/{ip}` and `/csv/{ip}` endpoints.

JSON responses can be encoded as JSONP, by adding the `callback` parameter:

```bash
curl freegeoip.net/json/?callback=foobar
```

The callback parameter is ignored on all other endpoints.

## Metrics and profiling

The freegeoip web server can provide metrics about its usage, and also supports runtime profiling and tracing.

Both are disabled by default, but can be enabled by passing the `-internal-server` parameter in the command line. Metrics are generated for [Prometheus](http://prometheus.io) and can be queried at `/metrics` even with curl.

HTTP pprof is available at `/debug/pprof` and the examples from the [pprof](https://golang.org/pkg/net/http/pprof/) package documentation should work on the freegeiop web server.

<a name="packagefreegeoip">

## Package freegeoip

The freegeoip package for the Go programming language provides two APIs:

- A database API that requires zero maintenance of the IP database;
- A geolocation `http.Handler` that can be used/served by any http server.

tl;dr if all you want is code then see the `example_test.go` file.

Otherwise check out the godoc reference.

[![GoDoc](https://godoc.org/github.com/apilayer/freegeoip?status.svg)](https://godoc.org/github.com/apilayer/freegeoip)
[![Build Status](https://secure.travis-ci.org/apilayer/freegeoip.png)](http://travis-ci.org/apilayer/freegeoip)
[![GoReportCard](https://goreportcard.com/badge/github.com/apilayer/freegeoip)](https://goreportcard.com/report/github.com/apilayer/freegeoip)

### Features

- Zero maintenance

The DB object alone can download an IP database file from the internet and service lookups to your program right away. It will auto-update the file in background and always magically work.

- DevOps friendly

If you do care about the database and have the commercial version of the MaxMind database, you can update the database file with your program running and the DB object will load it in background. You can focus on your stuff.

- Extensible

Besides the database part, the package provides an `http.Handler` object that you can add to your HTTP server to service IP geolocation lookups with the same simplistic API of freegeoip.net. There's also an interface for crafting your own HTTP responses encoded in any format.

### Install

Download the package:

	go get -d github.com/apilayer/freegeoip/...

Install the web server:

	go install github.com/apilayer/freegeoip/cmd/freegeoip

Test coverage is quite good, and test code may help you find the stuff you need.
