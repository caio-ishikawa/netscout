# Mimir

*"Norse deity known for his wisdom and knowledge of the past, present, and future."*

## Example usage:
- mimir -t present -u http://test.com -d 3 --graph
  - Crawls URL with depth declared from -d flag
  - Outputs single tree graph representing the crawl

- mimir -t past -u http://test.com -d 3 --graph
  - Attempts to find URL in WaybackMachine, and crawls with depth declared from -d flag
  - Outputs one tree diagram representing the crawl

## Components:
- Crawler (Golang): Crawls websites given a depth parameter. If --graph flag is set, it will output a json containing the URL map.
- Visualizer (JS): If --graph flag is set, it will fetch the json output and create an HTML doc containing an interactive tree.

## Ideas
- BinaryEdge integration
- SecurityTrails integration

