# NetScout

NetScout is an OSINT tool that finds domains, subdomains, directories, endpoints and files.


## Examples 
```sh
netscout -u https://crawler-test -d 2 -o netscout.txt
```

```sh
netscout -u https://crawler-test.com -d 2 --skip-binaryedge --skip-google-dork -o netscout.txt
```

```sh
netscout -u https://crawler-test.com -d 2 -t 5 --delay 1000 -o netscout.txt
```
