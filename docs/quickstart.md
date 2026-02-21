# Quick Start

## Read a Word Document

```bash
kit word read contract.docx
kit word read contract.docx --json
kit word read contract.docx --markdown
```

## Read an Excel File

```bash
kit excel read data.xlsx
kit excel read data.xlsx --sheet "Revenue" --json
kit excel read data.xlsx --csv
```

## AI-Powered Summary

```bash
export ANTHROPIC_API_KEY=sk-ant-...
kit word read contract.docx | kit ai summarize
kit ai summarize document.txt --focus "risks,dates"
```

## Extract Entities

```bash
kit ai extract --fields "parties,dates,amounts" contract.docx
```

## Ask Questions

```bash
kit ai ask "What are the payment terms?" contract.docx
```

## Run a Pipeline

```bash
kit pipeline run workflow.yaml --verbose
```
