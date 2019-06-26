# Swarm Guide

Swarm's documentation in sphinx, hosted on read the docs:
http://swarm-guide.readthedocs.io

## Building the source

After building the source you will find `index.html` in `./build/html/` folder.

### Requirements

- GNU Make
- Docker or Python (pip)

### Using Docker

After you have `docker` available just call `make html-docker`.

### Native with Python

Execute
```
pip install -r requirements.txt
make html
```
