# Installation

## Homebrew (macOS/Linux)

```bash
brew install klytics/tap/kit
```

## Go Install

```bash
go install github.com/klytics/m365kit/cmd@latest
```

## NPM (TypeScript package only)

```bash
npm install @m365kit/core
```

## Docker

```bash
docker run --rm -v $(pwd):/data ghcr.io/klytics/m365kit kit word read /data/document.docx
```

## Build from Source

```bash
git clone https://github.com/klytics/m365kit.git
cd m365kit
make build
./bin/kit --help
```
