# NetScout
<img src="https://i.imgur.com/pTmMYmZ.png">

NetScout is an OSINT tool that finds domains, subdomains, directories, endpoints and files for a given seed URL.
It consists of the following components:
- BinaryEdge client: Gets subdomains
- DNS: Attempts to perform a DNS zone transfer to extract subdomains
- Crawler: Gets URLs and directories from the found subdomains + the seed url
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
=======================================================================
 ███▄    █ ▓█████▄▄▄█████▓  ██████  ▄████▄   ▒█████   █    ██ ▄▄▄█████▓
 ██ ▀█   █ ▓█   ▀▓  ██▒ ▓▒▒██    ▒ ▒██▀ ▀█  ▒██▒  ██▒ ██  ▓██▒▓  ██▒ ▓▒
▓██  ▀█ ██▒▒███  ▒ ▓██░ ▒░░ ▓██▄   ▒▓█    ▄ ▒██░  ██▒▓██  ▒██░▒ ▓██░ ▒░
▓██▒  ▐▌██▒▒▓█  ▄░ ▓██▓ ░   ▒   ██▒▒▓▓▄ ▄██▒▒██   ██░▓▓█  ░██░░ ▓██▓ ░ 
▒██░   ▓██░░▒████▒ ▒██▒ ░ ▒██████▒▒▒ ▓███▀ ░░ ████▓▒░▒▒█████▓   ▒██▒ ░ 
░ ▒░   ▒ ▒ ░░ ▒░ ░ ▒ ░░   ▒ ▒▓▒ ▒ ░░ ░▒ ▒  ░░ ▒░▒░▒░ ░▒▓▒ ▒ ▒   ▒ ░░   
░ ░░   ░ ▒░ ░ ░  ░   ░    ░ ░▒  ░ ░  ░  ▒     ░ ▒ ▒░ ░░▒░ ░ ░     ░    
   ░   ░ ░    ░    ░      ░  ░  ░  ░        ░ ░ ░ ▒   ░░░ ░ ░   ░      
         ░    ░  ░              ░  ░ ░          ░ ░     ░              
=======================================================================
Usage:
  -u string
        A string representing the URL
  -d int
        An integer representing the depth of the crawl
  -t int
        An integer representing the amount of threads to use for the scans (default 5)
  -delay int
        An integer representing the delay between requests in miliseconds
  -lock-host
        A boolean - if set, it will only save URLs with the same host as the seed
  -o string
        A string representing the name of the output file
  -v
        A boolean - if set, it will display all found URLs

  -skip-axfr
        A bool - if set, it will skip the DNS zone trasnfer attempt
  -skip-binaryedge
        A bool - if set, it will skip BinaryEdge subdomain scan
  -skip-google-dork 
        A bool - if set, it will skip the Google filetype scan
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

## Testing
The crawler tests require the [DVWA (Damn Vulnerable Web App)](https://github.com/citizen-stig/dockerdvwa/tree/master) to be running locally with port 80 exposed.

