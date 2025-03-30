# CLI for generating a merged file for prompting LLMs

[![GitHub downloads](https://img.shields.io/github/downloads/frast/mergefiles/total)](https://github.com/frast/mergefiles/releases)

A CLI for generating a merged file for prompting LLMs.

## Usage

:speech_balloon: Merge all .go and .mod files in the current directory and all subdirs

```sh
mergefiles -ext .go -ext .mod
```

## Installation

You can download the latest binary from the [release page](https://github.com/frast/mergefiles/releases).

### Install via go

```shell
go install github.com/frast/mergefiles/mergefiles@latest
```

## Advanced usage

<details>
<summary>Click to expand </summary>

### Configuration

This cli tool reads configuration from `~/.config/mergefiles/config.yaml`.

Here is an example configuration:

```yaml
// Predefined prompts, use `-prompt` flag to switch prompt
prompts:
  default: |+
    You are ChatGPT, a large language model trained by OpenAI. 
    Answer as concisely as possible.
  go: |+
    You are an expert go software developer. 
    Answer as concisely as possible.
```

### Switch prompt

You can add more prompts in the config file:

```yaml
// Predefined prompts, use `-prompt` flag to switch prompt
prompts:
  default: |+
    You are ChatGPT, a large language model trained by OpenAI. 
    Answer as concisely as possible.
  go: |+
    You are an expert go software developer. 
    Answer as concisely as possible.
  java: |+
    You are an expert java software developer. 
    Answer as concisely as possible.
```

then use `-prompt` flag to switch prompt:

```sh
mergefiles -p java
```

> [!NOTE]
> The prompt can be a predefined prompt, or come up with one on the fly.
> e.g. `mergefiles -p java` or `chatgpt -p "You are a dog. You can only wowwow. That's it."`


</details>

## License

BSD