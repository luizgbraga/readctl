# readctl

A terminal companion for serious reading. Discuss books with an AI that can research literary criticism, academic papers, and contextual analysis.

## Install

```bash
go build ./cmd/readctl
```

## Configure

```bash
./readctl config
```

You'll be prompted for Anthropic API Key, LLM Model and Firecrawl API Key (optional, to enable web search). Config is stored at `~/.config/readctl/config.yaml`.

## Run

```bash
./readctl
```

## Usage

- Add books and create topics to discuss
- Chat with Claude about the book
- Use `/mode` to switch conversation styles (Scholar, Socratic, Dialectical, Provocateur)
- Use `/doc` to generate a document from the conversation

