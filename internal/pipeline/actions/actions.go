// Package actions provides built-in pipeline action implementations.
package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/klytics/m365kit/internal/formats/convert"
	"github.com/klytics/m365kit/internal/formats/docx"
	"github.com/klytics/m365kit/internal/formats/xlsx"
	"github.com/klytics/m365kit/internal/pipeline"
)

// RegisterAll registers all built-in actions with the given executor.
func RegisterAll(exec *pipeline.Executor) {
	exec.RegisterAction("word.read", WordReadAction)
	exec.RegisterAction("word.write", WordWriteAction)
	exec.RegisterAction("excel.read", ExcelReadAction)
	exec.RegisterAction("ai.summarize", AISummarizeAction)
	exec.RegisterAction("ai.analyze", AIAnalyzeAction)
	exec.RegisterAction("ai.extract", AIExtractAction)
	exec.RegisterAction("email.send", EmailSendAction)
	exec.RegisterAction("convert", ConvertAction)
	exec.RegisterAction("outlook.inbox", OutlookInboxAction)
	exec.RegisterAction("outlook.download", OutlookDownloadAction)
	exec.RegisterAction("acl.audit", ACLAuditAction)
}

// WordReadAction reads a Word document and returns its text content.
func WordReadAction(ctx context.Context, step pipeline.Step, input string) (string, error) {
	filePath := input
	if filePath == "" {
		return "", fmt.Errorf("word.read requires an input file path")
	}

	doc, err := docx.ParseFile(filePath)
	if err != nil {
		return "", err
	}

	format := "text"
	if f, ok := step.Options["format"]; ok {
		format = f
	}

	switch format {
	case "json":
		data, err := json.Marshal(doc)
		if err != nil {
			return "", fmt.Errorf("could not serialize document: %w", err)
		}
		return string(data), nil
	case "markdown":
		return doc.Markdown(), nil
	default:
		return doc.PlainText(), nil
	}
}

// WordWriteAction writes a Word document. The input should be the text content to write.
func WordWriteAction(ctx context.Context, step pipeline.Step, input string) (string, error) {
	outputPath := step.Template
	if outputPath == "" {
		if p, ok := step.Options["output"]; ok {
			outputPath = p
		}
	}
	if outputPath == "" {
		return "", fmt.Errorf("word.write requires a template (output path) or options.output")
	}

	doc := &docx.Document{
		Nodes: []docx.Node{
			{Type: docx.NodeParagraph, Text: input},
		},
	}

	data, err := docx.WriteDocument(doc)
	if err != nil {
		return "", fmt.Errorf("could not generate document: %w", err)
	}

	if err := writeFileBytes(outputPath, data); err != nil {
		return "", err
	}

	return outputPath, nil
}

// ExcelReadAction reads an Excel file and returns its data as JSON.
func ExcelReadAction(ctx context.Context, step pipeline.Step, input string) (string, error) {
	filePath := input
	if filePath == "" {
		return "", fmt.Errorf("excel.read requires an input file path")
	}

	wb, err := xlsx.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Filter to specific sheet if requested
	if sheetName, ok := step.Options["sheet"]; ok {
		sheet, err := wb.GetSheet(sheetName)
		if err != nil {
			return "", err
		}
		wb = &xlsx.Workbook{Sheets: []xlsx.Sheet{*sheet}}
	}

	data, err := json.Marshal(wb.Sheets)
	if err != nil {
		return "", fmt.Errorf("could not serialize spreadsheet data: %w", err)
	}

	return string(data), nil
}

// EmailSendAction is a placeholder for email sending functionality.
func EmailSendAction(ctx context.Context, step pipeline.Step, input string) (string, error) {
	return "", fmt.Errorf("email.send is not yet implemented — this action is planned for a future release")
}

// ConvertAction converts a file from one format to another.
func ConvertAction(ctx context.Context, step pipeline.Step, input string) (string, error) {
	inputPath := input
	if inputPath == "" {
		return "", fmt.Errorf("convert requires an input file path")
	}

	toFmt := ""
	if t, ok := step.Options["to"]; ok {
		toFmt = t
	}
	if toFmt == "" {
		return "", fmt.Errorf("convert requires options.to (target format)")
	}

	outputPath := ""
	if o, ok := step.Options["output"]; ok {
		outputPath = o
	}
	if outDir, ok := step.Options["out_dir"]; ok && outputPath == "" {
		base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
		outputPath = filepath.Join(outDir, base+"."+toFmt)
	}

	result, err := convert.Convert(inputPath, outputPath, toFmt)
	if err != nil {
		return "", err
	}

	if outputPath != "" {
		return outputPath, nil
	}
	return result, nil
}

// OutlookInboxAction lists inbox messages (pipeline placeholder — requires auth context).
func OutlookInboxAction(ctx context.Context, step pipeline.Step, input string) (string, error) {
	return "", fmt.Errorf("outlook.inbox requires authenticated Graph client — use 'kit outlook inbox' directly or ensure auth is configured")
}

// OutlookDownloadAction downloads email attachments (pipeline placeholder — requires auth context).
func OutlookDownloadAction(ctx context.Context, step pipeline.Step, input string) (string, error) {
	return "", fmt.Errorf("outlook.download requires authenticated Graph client — use 'kit outlook download' directly or ensure auth is configured")
}

// ACLAuditAction audits SharePoint permissions (pipeline placeholder — requires auth context).
func ACLAuditAction(ctx context.Context, step pipeline.Step, input string) (string, error) {
	return "", fmt.Errorf("acl.audit requires authenticated Graph client — use 'kit acl audit' directly or ensure auth is configured")
}

func writeFileBytes(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("could not write %s: %w", path, err)
	}
	return nil
}
