# NetScout
NetScout is an OSINT tool that finds domains, subdomains, directories, endpoints and files.
It consists of the following components:
- BinaryEdge client: Gets subdomains
- DNS AXFR (TODO): Attempts to perform a DNS transfer to extract subdomains
- Crawler: Gets links from the found subdomains + the seed url
- SERP client: Gets links for files. It uses Google dorking techniques to search for specific file types based on file extensions found by the crawler.  

## Setup
### Requirements
- Go 1.21.0
- BinaryEdge API key (optional)
- SERP API key (optional)

### Setting API keys
NetScout expects the API keys to be set as environment variables:
- ```export BINARYEDGE_API_KEY="<key>"```
- ```export SERP_API_KEY="<key>"```

## Examples 
Usage:
```
Usage of NetScout:
  -d int
        An integer representing the depth of the crawl
  -delay int
        An integer representing the delay between requests in miliseconds
  -lock-host
        A boolean - if set, it will only save URLs with the same host as the seed
  -o string
        A string representing the name of the output file
  -skip-binaryedge
        A bool - if set, it will skip BinaryEdge subdomain scan
  -skip-google-dork
        A bool - if set, it will skip the Google filetype scan
  -t int
        An integer representing the amount of threads to use for the scans (default 5)
  -u string
        A string representing the URL
  -v    A boolean - if set, it will display all found URLs
```

Sets seed url, depth, and output file:
```sh
netscout -u https://crawler-test -d 2 -o netscout.txt
```

Skips BinaryEdge and Google dork:
```sh
netscout -u https://crawler-test.com -d 2 --skip-binaryedge --skip-google-dork -o netscout.txt
```

Sets thread count to 5 and req delay to 1000ms
```sh
netscout -u https://crawler-test.com -d 2 -t 5 --delay 1000 -o netscout.txt
```
